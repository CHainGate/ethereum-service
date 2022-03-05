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
	"github.com/robfig/cron/v3"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
)

type OptsType struct {
	Main              string
	Test              string
	AccountPrivateKey string
}

type PaymentIntentRequest struct {
	Mode   int   `json:"mode"`
	Amount int64 `json:"amount"`
}

type PaymentIntent struct {
	mode    int
	account Account
	amount  *big.Int
}

type Account struct {
	publicKey  *ecdsa.PublicKey
	privateKey *ecdsa.PrivateKey
	address    common.Address
	used       bool
}

var (
	opts           *OptsType
	clientMain     *ethclient.Client
	clientTest     *ethclient.Client
	accounts       []Account
	paymentIntents []PaymentIntent
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Could not find env file [%v], using defaults", err)
	}

	o := &OptsType{}
	flag.StringVar(&o.Main, "MAIN", lookupEnv("MAIN"), "Mainnet")
	flag.StringVar(&o.Test, "TEST", lookupEnv("TEST"), "Testnet")
	flag.StringVar(&o.AccountPrivateKey, "ACCOUNT_PRIVATE_KEY", lookupEnv("ACCOUNT_PRIVATE_KEY"), "Account private Key")

	router := mux.NewRouter()
	router.HandleFunc("/test", test).Methods("GET", "OPTIONS")
	router.HandleFunc("/payment-intent", paymentIntent).Methods("POST", "OPTIONS")

	createCronJob()
	createMainClientConnection(o.Main)
	createTestClientConnection(o.Test)
	opts = o

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

func getBalanceAt(client *ethclient.Client, address common.Address) (*big.Int, error) {
	return client.BalanceAt(context.Background(), address, nil)
}

func createCronJob() {
	c := cron.New()
	_, err := c.AddFunc("@every 15s", checkAllAddresses)
	if err != nil {
		fmt.Println("Unable to add cron trigger")
	}
	c.Start()
}

func createTestClientConnection(connectionURITest string) {
	var err error
	clientTest, err = createClient(connectionURITest)
	if err == nil {
		fmt.Println("Testnet connection works")
	}
}

func createMainClientConnection(connectionURIMain string) {
	var err error
	clientMain, err = createClient(connectionURIMain)
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

func checkBalance(client *ethclient.Client, payIntent PaymentIntent) {
	balance, err := getBalanceAt(client, payIntent.account.address)
	if err != nil {
		log.Fatal(err)
	}

	if balance.Cmp(payIntent.amount) >= 0 {
		log.Printf("PAYMENT REACHED!!!!")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payIntent.amount.String())
		/*
			gasPrice, err := client.SuggestGasPrice(context.Background())
			if err != nil {
				log.Fatal(err)
			}

			nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
			if err != nil {
				log.Fatal(err)
			}

			chainID, err := client.NetworkID(context.Background())
			if err != nil {
				log.Fatal(err)
			}

			signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
			if err != nil {
				log.Fatal(err)
			}

			err = client.SendTransaction(context.Background(), signedTx)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("tx sent: %s", signedTx.Hash().Hex()) // tx sent: 0x77006fcb3938f648e2cc65bafd27dec30b9bfbe9df41f78498b9c8b7322a249e*/

	} else {
		log.Printf("PAYMENT still not reached")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payIntent.amount.String())
	}
}

func paymentIntent(w http.ResponseWriter, r *http.Request) {
	request := PaymentIntentRequest{Mode: 2, Amount: 1}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Fatal(err)
	}
	intent := PaymentIntent{request.Mode, getAccount(), big.NewInt(request.Amount)}
	paymentIntents = append(paymentIntents, intent)
}

func getFreeAccount() (Account, error) {
	if opts.AccountPrivateKey != "" {
		pk, err := crypto.HexToECDSA(opts.AccountPrivateKey)
		if err != nil {
			log.Fatal(err)
		}

		acc := Account{
			privateKey: pk,
			publicKey:  &pk.PublicKey,
			address:    common.HexToAddress(crypto.PubkeyToAddress(pk.PublicKey).Hex()),
		}

		return acc, nil
	}
	return Account{}, fmt.Errorf("unable to get free address")
}

func getAllSpectatedAccounts() []Account {
	return accounts
}

func getAllPaymentIntents() []PaymentIntent {
	return paymentIntents
}

func checkAllAddresses() {
	paymentIntents = getAllPaymentIntents()
	for _, s := range paymentIntents {
		switch s.mode {
		case 1:
			checkBalance(clientMain, s)
		case 2:
			checkBalance(clientTest, s)
		default:
			log.Fatal("Mode not supported!")
		}
	}
}

func getAccount() Account {
	acc, err := getFreeAccount()
	if err != nil {
		acc = createAccount()
	}
	return acc
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
	accounts = append(accounts, account)
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
