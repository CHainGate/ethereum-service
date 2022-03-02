package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"strconv"
)

type OptsType struct {
	Main string
	Test string
}

type PaymentIntent struct {
	mode int
}

type Account struct {
	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
	address    common.Address
}

var (
	clientMain *ethclient.Client
	clientTest *ethclient.Client
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Could not find env file [%v], using defaults", err)
	}

	o := &OptsType{}
	flag.StringVar(&o.Main, "MAIN", lookupEnv("MAIN"), "Mainnet")
	flag.StringVar(&o.Test, "TEST", lookupEnv("TEST"), "Testnet")

	router := mux.NewRouter()
	router.HandleFunc("/test", test).Methods("GET", "OPTIONS")
	router.HandleFunc("/payment-intent", paymentIntent).Methods("POST", "OPTIONS")

	clientMain, err = ethclient.Dial(o.Main)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Mainnet connection works")
	clientTest, err = ethclient.Dial(o.Test)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Testnet connection works")

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

func getBalanceAt(client *ethclient.Client, address common.Address) {
	balance, err := client.BalanceAt(context.Background(), address, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(balance)
}

func paymentIntent(w http.ResponseWriter, r *http.Request) {
	data := PaymentIntent{mode: 2}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Fatal(err)
	}

	switch data.mode {
	case 1:
		getBalanceAt(clientMain, getAddress())
	case 2:
		getBalanceAt(clientTest, getAddress())
	default:
		log.Printf("Mode not supported %v", err)
		w.Header().Set("Content-Type", "application/json")
		writeErr(w, http.StatusBadRequest, "Currency isn't supported %v", err)
	}
}

func getFreeAddress() (common.Address, error) {
	return common.HexToAddress(""), fmt.Errorf("unable to get free address")
}

func getAddress() common.Address {
	address, err := getFreeAddress()
	if err == nil {
		address = createAccount().address
	}
	return address
}

func createAccount() Account {
	account := Account{}
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	account.privateKey = privateKey
	privateKeyBytes := crypto.FromECDSA(account.privateKey)
	fmt.Println("SAVE BUT DO NOT SHARE THIS (Private Key):", hexutil.Encode(privateKeyBytes))

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	account.publicKey = publicKeyECDSA
	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	fmt.Println("Public Key:", hexutil.Encode(publicKeyBytes))

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	fmt.Println("Address:", address)
	account.address = common.HexToAddress(address)
	return account
}

func lookupEnv(key string, defaultValues ...string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	for _, v := range defaultValues {
		if v != "" {
			return v
		}
	}
	return ""
}

func writeErr(w http.ResponseWriter, code int, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	log.Printf(msg)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(code)
}
