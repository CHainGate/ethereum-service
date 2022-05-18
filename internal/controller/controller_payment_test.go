package controller

import (
	"context"
	"crypto/ecdsa"
	"ethereum-service/internal/config"
	"ethereum-service/internal/repository"
	"ethereum-service/internal/testutils"
	"ethereum-service/model"
	"ethereum-service/utils"
	"fmt"
	"log"
	"math/big"
	"regexp"
	"testing"

	"github.com/CHainGate/backend/pkg/enum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	geth "github.com/ethereum/go-ethereum/mobile"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"gopkg.in/h2non/gock.v1"
)

func TestCreatePayment(t *testing.T) {
	config.ReadOpts()
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
	mock = testutils.SetupUpdateAccount(mock)
	mock = testutils.SetupCreatePaymentWithoutIdCheck(mock)
	p, _, _ := CreatePayment(enum.Main, 100.0, "USD", model.CreateAccount(enum.Main).Address)
	if p.CurrentPaymentState.StatusName != enum.Waiting.String() {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Waiting.String())
	}
	if p.CurrentPaymentState.PayAmount.Int.Cmp(expectedPayAmountBigInt) != 0 {
		t.Fatalf("Payment has the wrong amount. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.PayAmount.Int.String(), expectedPayAmountBigInt.String())
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
	balance, err := GetUserBalanceAt(client, address, &acc.Remainder.Int) // nil is latest block
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

func customChainSetup(t *testing.T) (*model.Account, *ethclient.Client) {
	genesisAcc := model.CreateAccount(enum.Main)
	pk, _ := utils.GetPrivateKey(genesisAcc.PrivateKey)
	auth, _ := NewAuth(pk, context.Background())
	client := NewTestChain(t, auth)
	config.Chain = &config.ChainConfig{
		ChainId:  big.NewInt(1337),
		GasPrice: big.NewInt(params.InitialBaseFee),
	}
	return genesisAcc, client
}

func TestSingleForward(t *testing.T) {
	genesisAcc, client := customChainSetup(t)
	chaingateAcc, payAmount := setupFirstPayment(t, client, genesisAcc)
	createForward(t, client, chaingateAcc, payAmount, 1)
}

func TestWalletReusage(t *testing.T) {
	genesisAcc, client := customChainSetup(t)
	chaingateAcc, payAmount := setupFirstPayment(t, client, genesisAcc)
	createForward(t, client, chaingateAcc, payAmount, 1)

	txInitial := createInitialPayment(client, genesisAcc, payAmount, chaingateAcc.Address)
	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}
	createForward(t, client, chaingateAcc, payAmount, 2)
}

func setupFirstPayment(t *testing.T, client *ethclient.Client, genesisAcc *model.Account) (*model.Account, *big.Int) {
	chaingateAcc := model.CreateAccount(enum.Main)
	payAmount := big.NewInt(100000000000000)
	txInitial := createInitialPayment(client, genesisAcc, payAmount, chaingateAcc.Address)

	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}

	return chaingateAcc, payAmount
}

func createInitialPayment(client *ethclient.Client, genesisAcc *model.Account, payAmount *big.Int, targetAddress string) *types.Transaction {
	nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(genesisAcc.Address))
	if err != nil {
		log.Fatal(err)
	}

	var gasPrice *big.Int
	if config.Chain == nil {
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		gasPrice = config.Chain.GasPrice
	}
	gasTipCap, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	var chainID *big.Int
	if config.Chain == nil {
		chainID, err = client.NetworkID(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		chainID = config.Chain.ChainId
	}

	gasLimit := uint64(21000)

	toAddress := common.HexToAddress(targetAddress)

	// Transaction fees and Gas explained: https://docs.avax.network/learn/platform-overview/transaction-fees
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasFeeCap: gasPrice,  //gasPrice,     // maximum price per unit of gas that the transaction is willing to pay
		GasTipCap: gasTipCap, //tipCap,       // maximum amount above the baseFee of a block that the transaction is willing to pay to be included
		Gas:       gasLimit,
		To:        &toAddress,
		Value:     payAmount,
	})

	key, err := utils.GetPrivateKey(genesisAcc.PrivateKey)
	if err != nil {
		log.Fatal(err)
	}
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
	if err != nil {
		log.Fatal(err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}

	return signedTx
}

func createForward(t *testing.T, client *ethclient.Client, chaingateAcc *model.Account, payAmount *big.Int, iteration uint64) model.Payment {
	shouldChainGateEarnings := big.NewInt(1000000000000)
	merchantAcc := model.CreateAccount(enum.Main)
	p := testutils.GetPaidPayment()
	p.UserWallet = merchantAcc.Address
	p.CurrentPaymentState.PayAmount = model.NewBigInt(payAmount)
	p.Account = chaingateAcc

	fromBalance, err := GetUserBalanceAt(client, common.HexToAddress(chaingateAcc.Address), &p.Account.Remainder.Int)
	if fromBalance.Cmp(payAmount) != 0 {
		t.Fatalf(`Balance on generated wallet %v, should be %v`, fromBalance, payAmount)
	}

	forward(client, &p)

	if p.Account.Used == false {
		t.Fatalf(`The used wallet is: %v, should be %v`, p.Account.Used, false)
	}

	if p.Account.Nonce != iteration {
		t.Fatalf(`Nonce is: %v, should be %v`, p.Account.Nonce, iteration)
	}

	if fromBalance.Cmp(payAmount) != 0 {
		t.Fatalf(`Balance on generated wallet is: %v, should be %v`, fromBalance, payAmount)
	}

	toBalance, err := GetUserBalanceAt(client, common.HexToAddress(merchantAcc.Address), &merchantAcc.Remainder.Int)
	fromUserBalance, err := GetUserBalanceAt(client, common.HexToAddress(chaingateAcc.Address), &chaingateAcc.Remainder.Int)
	fromRealBalance, err := GetBalanceAt(client, common.HexToAddress(chaingateAcc.Address))
	if err != nil {
		t.Fatalf("Can't get balance %v", err)
	}

	fees := big.NewInt(0).Mul(big.NewInt(21000), config.Chain.GasPrice)
	chainGateEarnings := utils.GetChaingateEarnings(&p.CurrentPaymentState.PayAmount.Int)
	if chainGateEarnings.Cmp(shouldChainGateEarnings) != 0 {
		t.Fatalf(`CHainGate earnings is %v, should be %v`, chainGateEarnings, shouldChainGateEarnings)
	}

	finalAmount := big.NewInt(0).Sub(payAmount, fees.Add(fees, chainGateEarnings))
	if toBalance.Cmp(finalAmount) != 0 {
		t.Fatalf(`%v, should be %v`, toBalance, finalAmount)
	}
	if fromUserBalance.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf(`%v, should be %v`, fromUserBalance, big.NewInt(0))
	}
	if fromRealBalance.Cmp(&p.Account.Remainder.Int) != 0 {
		t.Fatalf(`%v, should be %v`, fromRealBalance, p.Account.Remainder)
	}

	return p
}

func NewTestChain(t *testing.T, auth *bind.TransactOpts) *ethclient.Client {
	address := auth.From
	backend, _, ethservice := newTestBackend(t, address)
	rpc, err := backend.Attach()

	if err != nil {
		log.Fatalf("creating rpc: %v", err)
	}
	primaryMiner := ethservice.Miner()
	go primaryMiner.Start(auth.From)

	client := ethclient.NewClient(rpc)

	t.Cleanup(func() {
		client.Close()
		primaryMiner.Stop()
	})

	return client
}

func newTestBackend(t *testing.T, address common.Address) (*node.Node, []*types.Block, *eth.Ethereum) {
	// Generate test chain.
	genesis, blocks := generateTestChain(address, big.NewInt(0).Mul(big.NewInt(1000000000000000000), big.NewInt(100000000)))
	// Create node
	n, err := node.New(&node.Config{})
	if err != nil {
		t.Fatalf("can't create new node: %v", err)
	}
	// Create Ethereum Service
	c := &ethconfig.Config{
		SyncMode: downloader.FullSync,
		Genesis:  genesis,
	}
	c.Ethash.PowMode = ethash.ModeFake
	ethservice, err := eth.New(n, c)
	if err != nil {
		t.Fatalf("can't create new ethereum service: %v", err)
	}
	// Import the test chain.
	if err := n.Start(); err != nil {
		t.Fatalf("can't start test node: %v", err)
	}
	if _, err := ethservice.BlockChain().InsertChain(blocks[1:]); err != nil {
		t.Fatalf("can't import test blocks: %v", err)
	}
	return n, blocks, ethservice
}

func generateTestChain(testAddr common.Address, testBalance *big.Int) (*core.Genesis, []*types.Block) {
	geth.SetVerbosity(0)
	db := rawdb.NewMemoryDatabase()
	c := params.AllEthashProtocolChanges
	genesis := &core.Genesis{
		GasLimit:   9223372036854775807,
		Difficulty: big.NewInt(1),
		Config:     c,
		Alloc:      core.GenesisAlloc{testAddr: {Balance: testBalance}},
		ExtraData:  []byte("test genesis"),
		Timestamp:  9000,
		BaseFee:    big.NewInt(params.InitialBaseFee),
	}
	generate := func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		g.SetExtra([]byte("test"))
	}
	gblock := genesis.ToBlock(db)
	engine := ethash.NewFaker()
	blocks, _ := core.GenerateChain(c, gblock, engine, db, 1, generate)
	blocks = append([]*types.Block{gblock}, blocks...)
	return genesis, blocks
}

func NewAuth(key *ecdsa.PrivateKey, ctx context.Context) (*bind.TransactOpts, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	if err != nil {
		return nil, err
	}

	auth.Context = ctx
	return auth, nil
}

func TestCheckBalanceWaiting(t *testing.T) {
	_, client := customChainSetup(t)
	p := testutils.GetWaitingPayment()
	CheckBalance(client, &p)
	if p.CurrentPaymentState.StatusName != enum.Waiting.String() {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Waiting.String())
	}
}

func TestCheckBalancePartiallyPaid(t *testing.T) {
	config.ReadOpts()
	defer gock.Off() // Flush pending mocks after test execution
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	mock, gormDb := testutils.NewMock()
	repository.InitPayment(gormDb)
	mock = testutils.SetupUpdatePaymentState(mock)
	genesisAcc, client := customChainSetup(t)
	p := testutils.GetWaitingPayment()
	txInitial := createInitialPayment(client, genesisAcc, big.NewInt(10), p.Account.Address)
	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}
	CheckBalance(client, &p)
	if p.CurrentPaymentState.StatusName != enum.PartiallyPaid.String() {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.PartiallyPaid.String())
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCheckBalancePaid(t *testing.T) {
	config.ReadOpts()
	defer gock.Off() // Flush pending mocks after test execution
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	mock, gormDb := testutils.NewMock()
	repository.InitPayment(gormDb)
	repository.InitAccount(gormDb)
	p := testutils.GetWaitingPayment()
	mock = testutils.SetupUpdatePaymentStateToPaid(mock, &p.CurrentPaymentState.PayAmount.Int)
	mock = testutils.SetupUpdatePaymentStateToForwarded(mock, &p.CurrentPaymentState.PayAmount.Int)
	mock = testutils.SetupUpdateAccountFree(mock, 1)
	amountAfterPayment := big.NewInt(0).Sub(&p.CurrentPaymentState.PayAmount.Int, &p.CurrentPaymentState.PayAmount.Int)
	remainder := model.NewBigInt(big.NewInt(0).Add(amountAfterPayment, utils.GetChaingateEarnings(&p.CurrentPaymentState.PayAmount.Int)))
	mock = testutils.SetupUpdatePaymentStateToFinished(mock, &p.CurrentPaymentState.PayAmount.Int, 1, remainder)
	genesisAcc, client := customChainSetup(t)
	txInitial := createInitialPayment(client, genesisAcc, &p.CurrentPaymentState.PayAmount.Int, p.Account.Address)
	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}
	CheckBalance(client, &p)
	if p.CurrentPaymentState.StatusName != enum.Finished.String() {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Finished.String())
	}
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestCheckForwardEarnings(t *testing.T) {
	config.ReadOpts()
	defer gock.Off() // Flush pending mocks after test execution
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		Reply(200)
	mock, gormDb := testutils.NewMock()
	repository.InitPayment(gormDb)
	repository.InitAccount(gormDb)
	p := testutils.GetWaitingPayment()
	overpayAmount := big.NewInt(0).Mul(&p.CurrentPaymentState.PayAmount.Int, big.NewInt(1000))
	mock = testutils.SetupUpdatePaymentStateToPaid(mock, overpayAmount)
	mock = testutils.SetupUpdatePaymentStateToForwarded(mock, overpayAmount)
	mock = testutils.SetupUpdateAccountFree(mock, 2)
	mock = testutils.SetupUpdatePaymentStateToFinished(mock, overpayAmount, 2, model.NewBigIntFromInt(0))
	genesisAcc, client := customChainSetup(t)
	txInitial := createInitialPayment(client, genesisAcc, overpayAmount, p.Account.Address)
	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}
	CheckBalance(client, &p)
	if p.CurrentPaymentState.StatusName != enum.Finished.String() {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.Finished.String())
	}
	bal, err := GetBalanceAt(client, common.HexToAddress(config.Opts.TargetWallet))
	bal2, err := GetBalanceAt(client, common.HexToAddress(p.Account.Address))
	fmt.Printf("final: %s\n", bal2.String())
	fmt.Printf("final2: %s\n", bal.String())

	if err != nil {
		t.Fatalf("Unable to check balance of %v", config.Opts.TargetWallet)
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
