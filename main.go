package main

import (
	"encoding/json"
	"ethereum-service/database"
	"ethereum-service/internal"
	"ethereum-service/internal/config"
	repository "ethereum-service/internal/repository"
	"ethereum-service/model"
	"ethereum-service/openApi"
	"ethereum-service/services"
	"fmt"
	"log"
	"math/big"
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

func checkAllAddresses() {
	paymentIntents := repository.Payment.GetAllPaymentIntents()
	for _, s := range paymentIntents {
		switch s.Mode {
		case "main":
			internal.CheckBalance(clientMain, &s)
		case "test":
			internal.CheckBalance(clientTest, &s)
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

	var payment model.Payment
	database.DB.Preload("CurrentPaymentState").First(&payment)
	payment.UpdatePaymentState("paid", big.NewInt(2))
	database.DB.Save(&payment)
	if err != nil {
		return
	}
	return
}
