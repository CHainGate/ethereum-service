package testutils

import (
	"context"
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	geth "github.com/ethereum/go-ethereum/mobile"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"log"
	"math/big"
	"testing"
)

func NewTestChain(t *testing.T, auth *bind.TransactOpts) *ethclient.Client {
	address := auth.From
	backend, _, ethservice := NewTestBackend(t, address)
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

func NewTestBackend(t *testing.T, address common.Address) (*node.Node, []*types.Block, *eth.Ethereum) {
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
