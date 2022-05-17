package main

import (
	"ethereum-service/database"
	"ethereum-service/internal/config"
	"ethereum-service/internal/controller"
	repository "ethereum-service/internal/repository"
	"ethereum-service/openApi"
	"ethereum-service/services"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

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

func cronHandler() {
	go checkAllAddresses()
	go getAllUnfinishedPayments()
}

func checkAllAddresses() {
	paymentIntents := repository.Payment.GetAllPaymentIntents()
	for _, s := range paymentIntents {
		switch s.Mode {
		case "main":
			go controller.CheckBalance(clientMain, &s)
		case "test":
			go controller.CheckBalance(clientTest, &s)
		default:
			log.Fatal("Mode not supported!")
		}
	}
}

func getAllUnfinishedPayments() {
	unfinishedPayments := repository.Payment.GetAllUnfinishedPayments()
	for _, s := range unfinishedPayments {
		switch s.Mode {
		case "main":
			go controller.HandleUnfinished(clientMain, &s)
		case "test":
			go controller.HandleUnfinished(clientTest, &s)
		default:
			log.Fatal("Mode not supported!")
		}
	}
}

func InitializeRouter() *mux.Router {
	PaymentApiService := services.NewPaymentApiService()
	PaymentApiController := openApi.NewPaymentApiController(PaymentApiService)

	router := openApi.NewRouter(PaymentApiController)

	// https://ribice.medium.com/serve-swaggerui-within-your-golang-application-5486748a5ed4
	sh := http.StripPrefix("/api/swaggerui/", http.FileServer(http.Dir("./swaggerui/")))
	router.PathPrefix("/api/swaggerui/").Handler(sh)

	//router.HandleFunc("/payment-intent", paymentIntent).Methods("POST", "OPTIONS")
	return router
}
