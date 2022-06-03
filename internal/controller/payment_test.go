package controller

import (
	"context"
	"crypto/ecdsa"
	"ethereum-service/internal/bc"
	"ethereum-service/internal/config"
	"ethereum-service/internal/repository"
	"ethereum-service/internal/testutils"
	"ethereum-service/model"
	"ethereum-service/utils"
	"log"
	"math/big"
	"regexp"
	"testing"
	"time"

	"github.com/CHainGate/backend/pkg/enum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/h2non/gock.v1"
)

func TestCreatePayment(t *testing.T) {
	config.ReadOpts()
	config.Chain = &config.ChainConfig{
		ChainId:  big.NewInt(1337),
		GasPrice: big.NewInt(params.InitialBaseFee),
	}
	expectedPayAmountFloat := 0.0001
	expectedPayAmountBigInt := utils.GetWEIFromETH(&expectedPayAmountFloat)
	mock, gormDb := testutils.NewMock()
	defer gock.Off() // Flush pending mocks after test execution
	gock.New("http://localhost:8001").
		Get("/api/price-conversion").
		Reply(200).
		JSON(map[string]float64{"Price": expectedPayAmountFloat})
	repository.InitAccount(gormDb)
	repository.InitPayment(gormDb)
	mock = testutils.SetupGetFreeAccount(mock)
	mock = testutils.SetupUpdateAccount(mock, 0)
	mock = testutils.SetupCreatePaymentWithoutIdCheck(mock)
	p, _, _ := CreatePayment(enum.Main, 100.0, "USD", model.CreateAccount(enum.Main).Address)
	if p.CurrentPaymentState.StatusName != enum.Waiting {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Waiting.String())
	}
	if p.CurrentPaymentState.PayAmount.Int.Cmp(expectedPayAmountBigInt) != 0 {
		t.Fatalf("Payment has the wrong amount. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.PayAmount.Int.String(), expectedPayAmountBigInt.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCheckBalanceNotifyPartially(t *testing.T) {
	config.ReadOpts()
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	p := testutils.GetWaitingPayment()
	CheckBalanceNotify(&p, big.NewInt(10), nil, nil)
	if p.CurrentPaymentState.StatusName != enum.PartiallyPaid {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.PartiallyPaid.String())
	}
}

func TestCheckBalanceFalselyNotifyPaid(t *testing.T) {
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	config.ReadOpts()
	config.Chain = &config.ChainConfig{
		ChainId:  big.NewInt(1337),
		GasPrice: big.NewInt(params.InitialBaseFee),
	}
	_, client := testutils.CustomChainSetup(t)
	config.ClientMain = client
	config.ClientTest = client
	mock, gormDb := testutils.NewMock()
	repository.InitPayment(gormDb)
	repository.InitAccount(gormDb)
	p := testutils.GetWaitingPayment()
	mock = testutils.SetupUpdateAccountFree(mock, 0)
	mock = testutils.SetupUpdatePaymentStateToFailed(mock)
	CheckBalanceNotify(&p, &p.CurrentPaymentState.PayAmount.Int, nil, nil)
	if p.CurrentPaymentState.StatusName != enum.Failed {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Failed.String())
	}
	if p.Account.Used {
		t.Fatalf("Accound is still used. Account is \"%v\", but should be \"%v\"", p.Account.Used, false)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCheckBalanceNotifyPaid(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	config.ReadOpts()
	config.Chain = &config.ChainConfig{
		ChainId:  big.NewInt(1337),
		GasPrice: big.NewInt(params.InitialBaseFee),
	}
	genesisAcc, client := testutils.CustomChainSetup(t)
	config.ClientMain = client
	config.ClientTest = client
	mock, gormDb := testutils.NewMock()
	repository.InitPayment(gormDb)
	repository.InitAccount(gormDb)
	p := testutils.GetWaitingPayment()
	txInitial := testutils.CreateInitialPayment(client, genesisAcc, &p.CurrentPaymentState.PayAmount.Int, p.Account.Address)
	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}
	mock = testutils.SetupUpdatePaymentStateToPaid(mock, &p.CurrentPaymentState.PayAmount.Int)
	CheckBalanceNotify(&p, &p.CurrentPaymentState.PayAmount.Int, nil, nil)
	if p.CurrentPaymentState.StatusName != enum.Paid {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Paid.String())
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCheckPaymentPaidBeforeExpire(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	config.ReadOpts()
	config.Chain = &config.ChainConfig{
		ChainId:  big.NewInt(1337),
		GasPrice: big.NewInt(params.InitialBaseFee),
	}
	genesisAcc, client := testutils.CustomChainSetup(t)
	config.ClientMain = client
	config.ClientTest = client
	mock, gormDb := testutils.NewMock()
	repository.InitPayment(gormDb)
	repository.InitAccount(gormDb)
	p := testutils.GetWaitingPayment()
	p.CreatedAt = p.CreatedAt.Add(time.Duration(-16) * time.Minute)
	txInitial := testutils.CreateInitialPayment(client, genesisAcc, &p.CurrentPaymentState.PayAmount.Int, p.Account.Address)
	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}
	mock = testutils.SetupUpdatePaymentStateToPaid(mock, &p.CurrentPaymentState.PayAmount.Int)
	CheckPayment(&p, nil, nil, nil)
	if p.CurrentPaymentState.StatusName != enum.Paid {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Paid.String())
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCheckPaymentExpire(t *testing.T) {
	defer gock.Off() // Flush pending mocks after test execution
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	config.ReadOpts()
	config.Chain = &config.ChainConfig{
		ChainId:  big.NewInt(1337),
		GasPrice: big.NewInt(params.InitialBaseFee),
	}
	_, client := testutils.CustomChainSetup(t)
	config.ClientMain = client
	config.ClientTest = client
	mock, gormDb := testutils.NewMock()
	repository.InitPayment(gormDb)
	repository.InitAccount(gormDb)
	p := testutils.GetWaitingPayment()
	p.CreatedAt = p.CreatedAt.Add(time.Duration(-16) * time.Minute)
	mock = testutils.SetupUpdateAccountFree(mock, 0)
	mock = testutils.SetupUpdatePaymentStateToExpired(mock)
	CheckPayment(&p, nil, nil, big.NewInt(0))
	if p.CurrentPaymentState.StatusName != enum.Expired {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Expired.String())
	}
	if p.Account.Used {
		t.Fatalf("Accound is still used. Account is \"%v\", but should be \"%v\"", p.Account.Used, false)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestEthClientAddressInteraction(t *testing.T) {
	client, err := ethclient.Dial("https://cloudflare-eth.com")
	if err != nil {
		log.Fatal(err)
	}
	acc := model.CreateAccount(enum.Main)

	address := common.HexToAddress(acc.Address)
	balance, err := bc.GetUserBalanceAt(client, address, &acc.Remainder.Int) // nil is latest block
	if err != nil {
		log.Fatal(err)
	}

	if balance.Uint64() != 0 {
		t.Fatalf(`%v, should be balance : %v`, balance.Uint64(), 0)
	}
}

func TestCreateAccount(t *testing.T) {
	acc := model.CreateAccount(enum.Main)

	if acc.Address == "" {
		t.Fatalf(`%v, want to be different than %v`, acc.Address, "")
	}
	if acc.PrivateKey == "" {
		t.Fatalf(`%v, want to be different than %v`, acc.Address, "")
	}
	if acc.Used == false {
		t.Fatalf(`%v, want to be %v`, acc.Address, "true")
	}

	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	if !re.MatchString(acc.Address) {
		t.Fatalf(`%v, must match regex: %v`, acc.Address, re.String())
	}

	pk, _ := utils.GetPrivateKey(acc.PrivateKey)

	publicKey := pk.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	generatedAddress := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	if acc.Address != generatedAddress {
		t.Fatalf(`%v, generates a different address than the saved one: %v`, acc.Address, generatedAddress)
	}
}

func TestCheckBalanceCronWaiting(t *testing.T) {
	_, client := testutils.CustomChainSetup(t)
	p := testutils.GetWaitingPayment()
	CheckBalanceStartup(client, &p)
	if p.CurrentPaymentState.StatusName != enum.Waiting {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Waiting.String())
	}
}

func TestCheckBalanceCronPartiallyPaid(t *testing.T) {
	config.ReadOpts()
	defer gock.Off() // Flush pending mocks after test execution
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	mock, gormDb := testutils.NewMock()
	repository.InitPayment(gormDb)
	mock = testutils.SetupUpdatePaymentState(mock)
	genesisAcc, client := testutils.CustomChainSetup(t)
	p := testutils.GetWaitingPayment()
	txInitial := testutils.CreateInitialPayment(client, genesisAcc, big.NewInt(10), p.Account.Address)
	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}
	CheckBalanceStartup(client, &p)
	if p.CurrentPaymentState.StatusName != enum.PartiallyPaid {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.PartiallyPaid.String())
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCheckBalanceCronPaidAndConfirmed(t *testing.T) {
	config.ReadOpts()
	defer gock.Off() // Flush pending mocks after test execution
	// Paid
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	// Forwarded
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	// Confirmed
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	// Finished
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	mock, gormDb := testutils.NewMock()
	repository.InitPayment(gormDb)
	repository.InitAccount(gormDb)
	p := testutils.GetWaitingPayment()
	amountAfterPayment := big.NewInt(0).Sub(&p.CurrentPaymentState.PayAmount.Int, &p.CurrentPaymentState.PayAmount.Int)
	remainder := model.NewBigInt(big.NewInt(0).Add(amountAfterPayment, utils.GetChaingateEarnings(&p.CurrentPaymentState.PayAmount.Int)))
	mock = testutils.SetupUpdatePaymentStateToPaid(mock, &p.CurrentPaymentState.PayAmount.Int)
	mock = testutils.SetupUpdatePaymentStateToConfirmed(mock, &p.CurrentPaymentState.PayAmount.Int)
	mock = testutils.SetupUpdateAccountWithRemainder(mock, 1, &remainder.Int)
	mock = testutils.SetupUpdatePaymentStateToForwarded(mock, &p.CurrentPaymentState.PayAmount.Int)
	mock = testutils.SetupUpdateAccountFree(mock, 1)
	mock = testutils.SetupUpdatePaymentStateToFinished(mock, &p.CurrentPaymentState.PayAmount.Int, 1, remainder)
	genesisAcc, client := testutils.CustomChainSetup(t)
	txInitial := testutils.CreateInitialPayment(client, genesisAcc, &p.CurrentPaymentState.PayAmount.Int, p.Account.Address)
	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}
	CheckBalanceStartup(client, &p)
	if p.CurrentPaymentState.StatusName != enum.Paid {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Paid.String())
	}
	HandleConfirming(client, &p)
	if p.CurrentPaymentState.StatusName != enum.Forwarded {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Forwarded.String())
	}
	finish(&p)
	if p.CurrentPaymentState.StatusName != enum.Finished {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Finished.String())
	}
	if p.Account.Used {
		t.Fatalf("Accound is still used. Account is \"%v\", but should be \"%v\"", p.Account.Used, false)
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCheckForwardEarnings(t *testing.T) {
	config.ReadOpts()
	defer gock.Off() // Flush pending mocks after test execution
	// Paid
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	// Forwarded
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	// Confirmed
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	// Finished
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	mock, gormDb := testutils.NewMock()
	repository.InitPayment(gormDb)
	repository.InitAccount(gormDb)
	p := testutils.GetWaitingPayment()
	overpayAmount := big.NewInt(0).Mul(&p.CurrentPaymentState.PayAmount.Int, big.NewInt(1000))
	mock = testutils.SetupUpdatePaymentStateToPaid(mock, overpayAmount)
	mock = testutils.SetupUpdatePaymentStateToConfirmed(mock, overpayAmount)
	mock = testutils.SetupUpdateAccount(mock, 1)
	mock = testutils.SetupUpdatePaymentStateToForwarded(mock, overpayAmount)
	mock = testutils.SetupUpdateAccount(mock, 2)
	mock = testutils.SetupUpdateAccountFree(mock, 2)
	mock = testutils.SetupUpdatePaymentStateToFinished(mock, overpayAmount, 2, model.NewBigIntFromInt(0))
	genesisAcc, client := testutils.CustomChainSetup(t)
	txInitial := testutils.CreateInitialPayment(client, genesisAcc, overpayAmount, p.Account.Address)
	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}
	CheckBalanceStartup(client, &p)
	HandleConfirming(client, &p)
	if p.CurrentPaymentState.StatusName != enum.Forwarded {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Forwarded.String())
	}
	finish(&p)
	if p.CurrentPaymentState.StatusName != enum.Finished {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Finished.String())
	}
	bal, err := bc.GetBalanceAt(client, common.HexToAddress(config.Opts.TargetWallet))

	if err != nil {
		t.Fatalf("Unable to check balance of %v", config.Opts.TargetWallet)
	}
	if p.Account.Used {
		t.Fatalf("Accound is still used. Account is \"%v\", but should be \"%v\"", p.Account.Used, false)
	}
	if p.Account.Remainder.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf("No amount was forwarded. Amount on wallet is \"%v\", but should be \"%v\"", p.Account.Remainder.String(), 0)
	}

	chainGateEarnings := utils.GetChaingateEarnings(&p.CurrentPaymentState.PayAmount.Int)
	p.Account.Remainder.Add(&p.Account.Remainder.Int, chainGateEarnings)

	finalAmount := big.NewInt(0).Add(overpayAmount, chainGateEarnings)
	fees := big.NewInt(0).Mul(big.NewInt(21000), config.Chain.GasPrice)
	fees = big.NewInt(0).Mul(fees, big.NewInt(1))
	finalAmount = big.NewInt(0).Sub(finalAmount, fees)
	finalAmount = big.NewInt(0).Sub(finalAmount, &p.CurrentPaymentState.PayAmount.Int)

	if bal.Cmp(finalAmount) != 0 {
		t.Fatalf("Not the right amount was forwarded. Amount on wallet is \"%v\", but should be \"%v\"", bal.String(), finalAmount.String())
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
