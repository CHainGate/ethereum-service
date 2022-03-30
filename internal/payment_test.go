package internal

import (
	"context"
	"crypto/ecdsa"
	"ethereum-service/model"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	geth "github.com/ethereum/go-ethereum/mobile"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"log"
	"math/big"
	"regexp"
	"testing"
)

func TestEthClientAddressInteraction(t *testing.T) {
	client, err := ethclient.Dial("https://cloudflare-eth.com")
	if err != nil {
		log.Fatal(err)
	}
	acc := createAccount()

	address := common.HexToAddress(acc.Address)
	balance, err := GetUserBalanceAt(client, address, &acc.Remainder.Int) // nil is latest block
	if err != nil {
		log.Fatal(err)
	}

	if balance.Uint64() != 0 {
		t.Fatalf(`%q, should be balance : %q`, balance.Uint64(), 0)
	}
}

func TestCreateAccount(t *testing.T) {
	acc := createAccount()

	if acc.Address == "" {
		t.Fatalf(`%q, want to be different than %q`, acc.Address, "")
	}
	if acc.PrivateKey == "" {
		t.Fatalf(`%q, want to be different than %q`, acc.Address, "")
	}
	if acc.Used == false {
		t.Fatalf(`%q, want to be %q`, acc.Address, "true")
	}

	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	if !re.MatchString(acc.Address) {
		t.Fatalf(`%q, must match regex: %q`, acc.Address, re.String())
	}

	pk, _ := GetPrivateKey(acc.PrivateKey)

	publicKey := pk.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	generatedAddress := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	if acc.Address != generatedAddress {
		t.Fatalf(`%q, generates a different address than the saved one: %q`, acc.Address, generatedAddress)
	}
}

func createInitialPayment(client *ethclient.Client, acc model.Account, chainID *big.Int, gasPrice *big.Int, targetAddress string) *types.Transaction {
	nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(acc.Address))
	if err != nil {
		log.Fatal(err)
	}

	if gasPrice == nil {
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	}

	gasTipCap, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	if chainID == nil {
		chainID, err = client.NetworkID(context.Background())
		if err != nil {
			log.Fatal(err)
		}
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
		Value:     big.NewInt(100000000000000),
	})

	key, err := GetPrivateKey(acc.PrivateKey)
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

func createPayment(targetAddress string, payAmount *big.Int, acc model.Account) model.Payment {
	initialBalance := big.NewInt(100000000000000)
	p := model.Payment{
		Mode:          "Main",
		Account:       acc,
		PriceAmount:   100,
		PriceCurrency: "USD",
		UserWallet:    targetAddress,
	}

	currentPaymentState := model.PaymentState{
		StatusName:     "waiting",
		AccountID:      p.AccountID,
		AmountReceived: model.NewBigInt(initialBalance),
		PayAmount:      model.NewBigInt(payAmount),
		PaymentID:      p.ID,
	}
	p.CurrentPaymentState = currentPaymentState
	p.PaymentStates = append(p.PaymentStates, currentPaymentState)

	return p
}

func TestForward(t *testing.T) {
	genesisAcc := createAccount()
	pk, _ := GetPrivateKey(genesisAcc.PrivateKey)
	auth, _ := NewAuth(pk, context.Background())
	client := NewTestChain(t, auth)

	payAmount := big.NewInt(100000000000000)
	shouldChainGateEarnings := big.NewInt(1000000000000)
	gasPrice := big.NewInt(params.InitialBaseFee)
	chainID := big.NewInt(1337)

	chaingateAcc := createAccount()
	merchantAcc := createAccount()
	p := createPayment(merchantAcc.Address, payAmount, *chaingateAcc)

	txInitial := createInitialPayment(client, *genesisAcc, chainID, gasPrice, chaingateAcc.Address)

	_, err := bind.WaitMined(context.Background(), client, txInitial)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}

	fromBalance, err := GetUserBalanceAt(client, common.HexToAddress(chaingateAcc.Address), &p.Account.Remainder.Int)
	if fromBalance.Cmp(payAmount) != 0 {
		t.Fatalf(`Balance on generated wallet %q, should be %q`, fromBalance, payAmount)
	}

	tx := forward(client, &p, chainID, gasPrice)

	if p.Account.Used == true {
		t.Fatalf(`The used wallet is: %v, should be %v`, p.Account.Used, false)
	}

	if p.Account.Nonce != 1 {
		t.Fatalf(`Nonce is: %v, should be %v`, p.Account.Nonce, 1)
	}

	if fromBalance.Cmp(payAmount) != 0 {
		t.Fatalf(`Balance on generated wallet is: %v, should be %v`, fromBalance, payAmount)
	}

	_, err = bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		t.Fatalf("Can't wait until transaction is mined %v", err)
	}

	toBalance, err := GetUserBalanceAt(client, common.HexToAddress(merchantAcc.Address), &merchantAcc.Remainder.Int)
	fromUserBalance, err := GetUserBalanceAt(client, common.HexToAddress(chaingateAcc.Address), &chaingateAcc.Remainder.Int)
	fromRealBalance, err := GetBalanceAt(client, common.HexToAddress(chaingateAcc.Address))
	if err != nil {
		t.Fatalf("Can't get balance %v", err)
	}

	fees := big.NewInt(0).Mul(big.NewInt(21000), gasPrice)
	chainGateEarnings := getChaingateEarnings(p, 1)
	if chainGateEarnings.Cmp(shouldChainGateEarnings) != 0 {
		t.Fatalf(`CHainGate earnings is %v, should be %v`, chainGateEarnings, shouldChainGateEarnings)
	}

	finalAmount := payAmount.Sub(payAmount, fees.Add(fees, chainGateEarnings))
	if toBalance.Cmp(finalAmount) != 0 {
		t.Fatalf(`%v, should be %v`, toBalance, finalAmount)
	}
	if fromUserBalance.Cmp(big.NewInt(0)) != 0 {
		t.Fatalf(`%v, should be %v`, fromUserBalance, big.NewInt(0))
	}
	if fromRealBalance.Cmp(&p.Account.Remainder.Int) != 0 {
		t.Fatalf(`%v, should be %v`, fromRealBalance, p.Account.Remainder)
	}
}

func NewTestChain(t testing.TB, auth *bind.TransactOpts) *ethclient.Client {
	t.Helper()

	geth.SetVerbosity(1)
	address := auth.From
	db := rawdb.NewMemoryDatabase()
	genesis := &core.Genesis{
		Config:    params.AllEthashProtocolChanges,
		Alloc:     core.GenesisAlloc{address: {Balance: big.NewInt(1000000000000000000)}},
		ExtraData: []byte("test genesis"),
		Timestamp: 9000,
		BaseFee:   big.NewInt(params.InitialBaseFee),
	}
	generate := func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		g.SetExtra([]byte("test"))
	}
	gblock := genesis.ToBlock(db)
	engine := ethash.NewFaker()
	blocks, _ := core.GenerateChain(params.AllEthashProtocolChanges, gblock, engine, db, 1, generate)
	blocks = append([]*types.Block{gblock}, blocks...)

	// Create node
	n, err := node.New(&node.Config{})
	if err != nil {
		t.Fatalf("can't create new node: %v", err)
	}
	// Create Ethereum Service
	config := &ethconfig.Config{Genesis: genesis}
	config.Ethash.PowMode = ethash.ModeFake
	ethservice, err := eth.New(n, config)
	if err != nil {
		t.Fatalf("can't create new ethereum service: %v", err)
	}
	// Import the test chain.
	if err = n.Start(); err != nil {
		t.Fatalf("can't start test node: %v", err)
	}
	if _, err = ethservice.BlockChain().InsertChain(blocks[1:]); err != nil {
		t.Fatalf("can't import test blocks: %v", err)
	}

	rpc, err := n.Attach()
	if err != nil {
		t.Fatalf("creating rpc: %v", err)
	}
	m := ethservice.Miner()
	go m.Start(auth.From)

	client := ethclient.NewClient(rpc)

	t.Cleanup(func() {
		client.Close()
		m.Stop()
	})

	return client
}

func NewAuth(key *ecdsa.PrivateKey, ctx context.Context) (*bind.TransactOpts, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(1337))
	if err != nil {
		return nil, err
	}

	auth.Context = ctx
	return auth, nil
}
