package main

import (
	"context"
	"ethereum-service/database"
	"ethereum-service/internal/config"
	"ethereum-service/internal/controller"
	repository "ethereum-service/internal/repository"
	"ethereum-service/openApi"
	"ethereum-service/services"
	"fmt"
	"github.com/CHainGate/backend/pkg/enum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"

	"github.com/robfig/cron/v3"
)

func main() {
	config.ReadOpts()
	database.DbInit()
	router := InitializeRouter()

	//createCronJob()

	config.CreateMainClientConnection(config.Opts.Main)
	config.CreateTestClientConnection(config.Opts.Test)

	go listenToEthChain(config.ClientMain, enum.Main)
	go listenToEthChain(config.ClientTest, enum.Test)
	log.Printf("listing on port %v", 9000)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(9000), router))
}

func createCronJob() {
	c := cron.New()
	_, err := c.AddFunc("@every 15s", cronHandler)
	if err != nil {
		fmt.Println("Unable to add cron trigger")
	}
	c.Start()
}

func cronHandler() {
	go checkAllAddresses()
}

func checkAllAddresses() {
	paymentIntents := repository.Payment.GetAllPaymentIntents()
	for _, s := range paymentIntents {
		switch s.Mode {
		case "main":
			go controller.CheckBalanceStartup(config.ClientMain, &s)
		case "test":
			go controller.CheckBalanceStartup(config.ClientTest, &s)
		default:
			log.Fatal("Mode not supported!")
		}
	}
}

func checkConfirming(client *ethclient.Client, currentBlockNr uint64, mode enum.Mode) {
	paymentIntents := repository.Payment.GetAllConfirmingPayments(mode)
	for _, p := range paymentIntents {
		// This could be maybe done via SQL query
		log.Printf("blocknr: %v", p.ReceivingBlockNr)
		log.Printf("currentBlockNr: %v", currentBlockNr)
		log.Printf("isittrue: %v", p.ReceivingBlockNr == 0 || currentBlockNr > p.ReceivingBlockNr+6)
		if p.ReceivingBlockNr == 0 || currentBlockNr >= p.ReceivingBlockNr+6 {
			p.ForwardingBlockNr = currentBlockNr
			go controller.HandleConfirming(client, &p)
		}
	}
}

func listenToEthChain(client *ethclient.Client, mode enum.Mode) {
	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case header := <-headers:
			block, err := client.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				log.Fatal(err)
			}

			payments := repository.Payment.GetModePaymentIntents(mode)
			for _, p := range payments {
				for _, tx := range block.Transactions() {
					if tx.To() != nil && tx.To().Hex() == p.Account.Address {
						controller.CheckBalanceNotify(&p, tx.Value(), block.Number().Uint64())
					}
				}

				controller.CheckIfExpired(&p, nil)
			}

			checkConfirming(client, block.Number().Uint64(), mode)
		}
	}
}

func getAllUnfinishedPayments() {
	unfinishedPayments := repository.Payment.GetAllUnfinishedPayments()
	for _, s := range unfinishedPayments {
		switch s.Mode {
		case "main":
			go controller.HandleUnfinished(config.ClientMain, &s)
		case "test":
			go controller.HandleUnfinished(config.ClientTest, &s)
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
