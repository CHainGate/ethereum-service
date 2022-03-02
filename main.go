package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/test", test).Methods("GET", "OPTIONS")
	router.HandleFunc("/address", createAddress).Methods("GET", "OPTIONS")

	client, err := ethclient.Dial("")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("we have a connection")
	_ = client // we'll use this in the upcoming sections
	account := common.HexToAddress("")
	balance, err := client.BalanceAt(context.Background(), account, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(balance)

	log.Printf("listing on port %v", 9000)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(9000), router))
}

func test(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Test:")
	err := json.NewEncoder(w).Encode("Test")
	if err != nil {
		return
	}
	return
}

func createAddress(w http.ResponseWriter, r *http.Request) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}

	privateKeyBytes := crypto.FromECDSA(privateKey)
	fmt.Println("SAVE BUT DO NOT SHARE THIS (Private Key):", hexutil.Encode(privateKeyBytes))

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	fmt.Println("Public Key:", hexutil.Encode(publicKeyBytes))

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	fmt.Println("Address:", address)
}
