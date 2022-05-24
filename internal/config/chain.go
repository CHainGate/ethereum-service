package config

import (
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
)

type ChainConfig struct {
	ChainId  *big.Int
	GasPrice *big.Int
}

var (
	ClientMain *ethclient.Client
	ClientTest *ethclient.Client
	Chain      *ChainConfig
)

func CreateTestClientConnection(connectionURITest string) {
	var err error
	ClientTest, err = createClient(connectionURITest)
	if err == nil {
		fmt.Println("Testnet connection works")
	}
}

func CreateMainClientConnection(connectionURIMain string) {
	var err error
	ClientMain, err = createClient(connectionURIMain)
	if err == nil {
		fmt.Println("Mainnet connection works")
	}
}

func createClient(connectionURI string) (*ethclient.Client, error) {
	client, err := ethclient.Dial(connectionURI)
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Connection works with %s", connectionURI)
	}
	return client, err
}
