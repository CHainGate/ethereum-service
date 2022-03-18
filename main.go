package main

import (
	"context"
	"encoding/json"
	"ethereum-service/database"
	"ethereum-service/internal"
	"ethereum-service/internal/config"
	"ethereum-service/model"
	"ethereum-service/openApi"
	"ethereum-service/services"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"math/big"
	"net/http"
	"sort"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/robfig/cron/v3"
)

var (
	clientMain *ethclient.Client
	clientTest *ethclient.Client
)

func main() {
	config.ReadOpts()
	database.DbInit()
	router := InitializeRouter()

	createCronJob()
	createMainClientConnection(config.Opts.Main)
	createTestClientConnection(config.Opts.Test)

	log.Printf("listing on port %v", 9000)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(9000), router))
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

func checkBalance(client *ethclient.Client, payIntent model.Payment) {
	balance, err := getBalanceAt(client, common.HexToAddress(payIntent.Account.Address))
	if err != nil {
		log.Fatal(err)
	}

	//TODO: is there a better solution?
	sort.Slice(payIntent.PaymentStates, func(i, j int) bool {
		return payIntent.PaymentStates[i].CreatedAt.After(payIntent.PaymentStates[j].CreatedAt)
	})

	if balance.Cmp(payIntent.PaymentStates[0].PayAmount.BigInt()) >= 0 {
		log.Printf("PAYMENT REACHED!!!!")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payIntent.PaymentStates[0].PayAmount.BigInt().String())
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
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payIntent.PaymentStates[0].PayAmount.BigInt().String())
	}
}

func getAllPaymentIntents() []model.Payment {
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

func getTest() string {
	return "Test:"
}

func InitializeRouter() *mux.Router {
	PaymentApiService := services.NewPaymentApiService()
	PaymentApiController := openApi.NewPaymentApiController(PaymentApiService)

	router := openApi.NewRouter(PaymentApiController)

	// https://ribice.medium.com/serve-swaggerui-within-your-golang-application-5486748a5ed4
	sh := http.StripPrefix("/api/swaggerui/", http.FileServer(http.Dir("./swaggerui/")))
	router.PathPrefix("/api/swaggerui/").Handler(sh)

	router.HandleFunc("/test", test).Methods("GET", "OPTIONS")
	//router.HandleFunc("/payment-intent", paymentIntent).Methods("POST", "OPTIONS")
	return router
}

func test(w http.ResponseWriter, r *http.Request) {
	fmt.Println(getTest())
	err := json.NewEncoder(w).Encode(getTest())
	if err != nil {
		return
	}
	return
}
