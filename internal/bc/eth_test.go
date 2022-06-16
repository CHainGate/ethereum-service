package bc

import (
	"context"
	"ethereum-service/internal/config"
	"ethereum-service/internal/testutils"
	"ethereum-service/model"
	"ethereum-service/utils"
	"math/big"
	"strings"
	"testing"

	"github.com/CHainGate/backend/pkg/enum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TestSingleForward(t *testing.T) {
	config.ReadOpts()
	genesisAcc, client := testutils.CustomChainSetup(t)
	chaingateAcc, payAmount := SetupFirstPayment(t, client, genesisAcc)
	CreateForward(t, client, chaingateAcc, payAmount, 1)
}

func TestIsFalsyPaidOnChain(t *testing.T) {
	config.ReadOpts()
	_, client := testutils.CustomChainSetup(t)
	p := testutils.GetPaidPayment()
	paid, balance := IsPaidOnChain(&p, client)
	if paid {
		t.Fatalf(`It should't be paid with %v`, balance)
	}
	if balance.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf(`balance should be %v is %v`, 0, balance)
	}
}

func TestIsPaidOnChain(t *testing.T) {
	config.ReadOpts()
	genesisAcc, client := testutils.CustomChainSetup(t)
	chaingateAcc, payAmount := SetupFirstPayment(t, client, genesisAcc)
	p := testutils.GetPaidPayment()
	p.CurrentPaymentState.PayAmount = model.NewBigInt(payAmount)
	p.Account = *chaingateAcc
	paid, balance := IsPaidOnChain(&p, client)
	if !paid {
		t.Fatalf(`It should be paid with %v`, balance)
	}
}

// https://rpc.info/
func TestGetClientByMode(t *testing.T) {
	config.CreateMainClientConnection("https://mainnet.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161")
	config.CreateTestClientConnection("https://rinkeby.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161")
	client := GetClientByMode(enum.Main)
	testClient := GetClientByMode(enum.Test)
	networkId, err := client.NetworkID(context.Background())
	if err != nil {
		t.Fatalf("Unable to get networkID %v", err)
	}
	if networkId.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf(`It isn't mainnet it is' %v, should be %v`, networkId, 1)
	}
	networkIdTest, err := testClient.NetworkID(context.Background())
	if err != nil {
		t.Fatalf("Unable to get networkID %v", err)
	}
	if networkIdTest.Cmp(big.NewInt(4)) != 0 {
		t.Fatalf(`It isn't mainnet it is' %v, should be %v`, networkIdTest, 4)
	}
}

func TestCheckForwardEarnings(t *testing.T) {
	config.ReadOpts()
	p := testutils.GetForwardedPayment()
	overpayAmount := big.NewInt(0).Mul(&p.CurrentPaymentState.PayAmount.Int, big.NewInt(1000))
	genesisAcc, client := testutils.CustomChainSetup(t)
	txInitial := testutils.CreateInitialPayment(client, genesisAcc, overpayAmount, p.Account.Address)
	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}
	p.Account.Remainder = model.NewBigInt(overpayAmount)
	check, tx := CheckForwardEarnings(client, &p.Account)
	if !check {
		t.Fatalf("Money should be forwarded, but function says no")
	}
	if strings.ToLower(tx.To().Hex()) != strings.ToLower(config.Opts.TargetWallet) {
		t.Fatalf("Function was sent to: %v , but should be sent to: %v", strings.ToLower(tx.To().Hex()), strings.ToLower(config.Opts.TargetWallet))
	}
}

func TestWalletReusage(t *testing.T) {
	config.ReadOpts()
	genesisAcc, client := testutils.CustomChainSetup(t)
	chaingateAcc, payAmount := SetupFirstPayment(t, client, genesisAcc)
	p := CreateForward(t, client, chaingateAcc, payAmount, 1)

	txInitial := testutils.CreateInitialPayment(client, genesisAcc, payAmount, chaingateAcc.Address)
	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}
	CreateForward(t, client, &p.Account, payAmount, 2)
}

func TestCheckIfAmountIsTooLowFalse(t *testing.T) {
	config.ReadOpts()
	final := big.NewInt(1)
	_, client := testutils.CustomChainSetup(t)
	if CheckIfAmountIsTooLow(client, final) == nil {
		t.Fatalf(`The amount should be too low with %v`, final.String())
	}
}

func TestCheckIfAmountIsTooLowCorrect(t *testing.T) {
	config.ReadOpts()
	_, client := testutils.CustomChainSetup(t)
	final := big.NewInt(100000000000000)
	err := CheckIfAmountIsTooLow(client, final)
	if err != nil {
		println(err.Error())
		t.Fatalf(`The amount should accepted with %v`, final.String())
	}
}

func CreateForward(t *testing.T, client *ethclient.Client, chaingateAcc *model.Account, payAmount *big.Int, iteration uint64) model.Payment {
	shouldChainGateEarnings := big.NewInt(1000000000000)
	merchantAcc := model.CreateAccount(enum.Main)
	p := testutils.GetPaidPayment()
	p.MerchantWallet = merchantAcc.Address
	p.CurrentPaymentState.PayAmount = model.NewBigInt(payAmount)
	p.Account = *chaingateAcc

	fromBalance, err := GetUserBalanceAt(client, common.HexToAddress(chaingateAcc.Address), &p.Account.Remainder.Int)
	if fromBalance.Cmp(payAmount) != 0 {
		t.Fatalf(`Balance on generated wallet %v, should be %v`, fromBalance, payAmount)
	}

	Forward(client, &p)

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
	fromUserBalance, err := GetUserBalanceAt(client, common.HexToAddress(chaingateAcc.Address), &p.Account.Remainder.Int)
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

func SetupFirstPayment(t *testing.T, client *ethclient.Client, genesisAcc *model.Account) (*model.Account, *big.Int) {
	cgAcc := model.CreateAccount(enum.Main)
	payAmount := big.NewInt(100000000000000)
	txInitial := testutils.CreateInitialPayment(client, genesisAcc, payAmount, cgAcc.Address)

	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}

	return cgAcc, payAmount
}
