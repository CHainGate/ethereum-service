package main

import (
	"context"
	"encoding/json"
	"ethereum-service/database"
	"ethereum-service/internal"
	"ethereum-service/openApi"
	"ethereum-service/services"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

type OptsType struct {
	Main              string
	Test              string
	AccountPrivateKey string
}

var (
	opts       *OptsType
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

	PaymentApiService := services.NewPaymentApiService()
	PaymentApiController := openApi.NewPaymentApiController(PaymentApiService)

	router := openApi.NewRouter(PaymentApiController)

	// https://ribice.medium.com/serve-swaggerui-within-your-golang-application-5486748a5ed4
	sh := http.StripPrefix("/api/swaggerui/", http.FileServer(http.Dir("./swaggerui/")))
	router.PathPrefix("/api/swaggerui/").Handler(sh)

	router.HandleFunc("/test", test).Methods("GET", "OPTIONS")
	//router.HandleFunc("/payment-intent", paymentIntent).Methods("POST", "OPTIONS")

	createCronJob()
	createMainClientConnection(o.Main)
	createTestClientConnection(o.Test)
	opts = o
	internal.CreateInternalOps()
	database.DbInit()

	log.Printf("listing on port %v", 9000)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(9000), router))
}

func test(w http.ResponseWriter, r *http.Request) {
	fmt.Println(getTest())
	err := json.NewEncoder(w).Encode(getTest())
	if err != nil {
		return
	}
	return
}

func getTest() string {
	return "Test:"
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

func checkBalance(client *ethclient.Client, payIntent internal.Payment) {
	balance, err := getBalanceAt(client, payIntent.Account.Address)
	if err != nil {
		log.Fatal(err)
	}

	//TODO: is there a better solution?
	sort.Slice(payIntent.PaymentStates, func(i, j int) bool {
		return payIntent.PaymentStates[i].CreatedAt.After(payIntent.PaymentStates[j].CreatedAt)
	})

	if balance.Cmp(&payIntent.PaymentStates[0].PayAmount) >= 0 {
		log.Printf("PAYMENT REACHED!!!!")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payIntent.PaymentStates[0].PayAmount.String())
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
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payIntent.PaymentStates[0].PayAmount.String())
	}
}

func getAllPaymentIntents() []internal.Payment {
	return internal.PaymentIntents
}

func checkAllAddresses() {
	paymentIntents := getAllPaymentIntents()
	for _, s := range paymentIntents {
		switch s.Mode {
		case "main":
			checkBalance(clientMain, s)
		case "test":
			checkBalance(clientTest, s)
		default:
			log.Fatal("Mode not supported!")
		}
	}
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
