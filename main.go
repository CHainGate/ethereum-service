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
	"log"
	"math/big"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gorilla/mux"

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

func getChaingateEarnings(payment model.Payment, percent int64) *big.Int {
	chainGatePercent := big.NewInt(percent)
	chainGateEarnings := big.NewInt(0)
	mul := chainGateEarnings.Mul(&payment.CurrentPaymentState.PayAmount.Int, chainGatePercent)
	final := mul.Div(mul, big.NewInt(100))
	return final
}

func paid(client *ethclient.Client, payment model.Payment, balance *big.Int) {
	log.Printf("PAYMENT REACHED!!!!")
	log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payment.GetActiveAmount().String())

	nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(payment.Account.Address))
	if err != nil {
		log.Fatal(err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	gasTipCap, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	gasLimit := uint64(21000)

	chainGateEarnings := getChaingateEarnings(payment, 1)

	payment.CurrentPaymentState.PayAmount.Sub(&payment.CurrentPaymentState.PayAmount.Int, chainGateEarnings)
	payment.Account.Remainder.Add(&payment.Account.Remainder.Int, chainGateEarnings)

	toAddress := common.HexToAddress("0x916C2110E4b540c8D3Bb522d40a1E42Ec149aAE8")
	fees := big.NewInt(0).Mul(big.NewInt(21000), gasPrice)
	fmt.Printf("gasPrice: %s\n", gasPrice.String())

	fmt.Printf("Fees: %s\n", fees.String())
	finalAmount := big.NewInt(0).Sub(payment.GetActiveAmount(), fees)
	fmt.Printf("finalAmount: %s\n", finalAmount.String())

	// Transaction fees and Gas explained: https://docs.avax.network/learn/platform-overview/transaction-fees
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasFeeCap: gasPrice,  //gasPrice,     // maximum price per unit of gas that the transaction is willing to pay
		GasTipCap: gasTipCap, //tipCap,       // maximum amount above the baseFee of a block that the transaction is willing to pay to be included
		Gas:       gasLimit,
		To:        &toAddress,
		Value:     finalAmount,
	})

	key, err := internal.GetPrivateKey(payment.Account.PrivateKey)
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

	fmt.Printf("tx sent: %s", signedTx.Hash().Hex()) // tx sent: 0x77006fcb3938f648e2cc65bafd27dec30b9bfbe9df41f78498b9c8b7322a249e

	payment.Account.Used = false
	payment.UpdatePaymentState("paid", balance)
	database.DB.Save(&payment)
}

func partlyPaid(client *ethclient.Client, payment model.Payment, balance *big.Int) {
	if balance.Cmp(&payment.CurrentPaymentState.AmountReceived.Int) > 0 {
		payment.UpdatePaymentState("partially_paid", balance)
		database.DB.Save(&payment)
	}
}

func checkBalance(client *ethclient.Client, payment model.Payment) {
	balance, err := getBalanceAt(client, common.HexToAddress(payment.Account.Address))
	if err != nil {
		log.Fatal(err)
	}

	if balance.Cmp(payment.GetActiveAmount()) >= 0 {
		paid(client, payment, balance)
	} else if payment.CurrentPaymentState.IsWaitingForPayment() {
		partlyPaid(client, payment, balance)
		log.Printf("PAYMENT party paid")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payment.GetActiveAmount().String())
	} else {
		log.Printf("PAYMENT still not reached")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payment.GetActiveAmount().String())
	}
}

func getAllPaymentIntents() []model.Payment {
	var payments []model.Payment
	database.DB.
		Preload("Account").
		Preload("CurrentPaymentState").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"status_name\" IN ?", []string{"waiting", "partially_paid"}).
		Find(&payments)
	return payments
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

	var payment model.Payment
	database.DB.Preload("CurrentPaymentState").First(&payment)
	payment.UpdatePaymentState("paid", big.NewInt(2))
	database.DB.Save(&payment)
	if err != nil {
		return
	}
	return
}
