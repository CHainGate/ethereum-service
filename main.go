package main

import (
	"context"
	"ethereum-service/database"
	"ethereum-service/internal/bc"
	"ethereum-service/internal/config"
	"ethereum-service/internal/controller"
	repository "ethereum-service/internal/repository"
	"ethereum-service/openApi"
	"ethereum-service/services"
	"github.com/CHainGate/backend/pkg/enum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
)

func main() {
	config.ReadOpts()
	database.DbInit()
	router := InitializeRouter()

	config.CreateMainClientConnection(config.Opts.Main)
	config.CreateTestClientConnection(config.Opts.Test)

	checkAllAddresses()

	go listenToEthChain(enum.Main)
	go listenToEthChain(enum.Test)
	log.Printf("listing on port %v", 9000)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(9000), router))
}

/*
   Recovery
*/
func checkAllAddresses() {
	payments := repository.Payment.GetAllOpen()
	for _, s := range payments {
		client := bc.GetClientByMode(s.Mode)
		go controller.CheckBalanceStartup(client, &s)
	}
}

func listenToEthChain(mode enum.Mode) {
	headers := make(chan *types.Header)
	client := bc.GetClientByMode(mode)
	sub, err := client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case err := <-sub.Err():
			log.Fatal("Error in websocket", err)
		case header := <-headers:
			block, err := client.BlockByHash(context.Background(), header.Hash())
			if err != nil {
				log.Printf("Error in getting BlockByHash %v", err)
			} else {
				payments := repository.Payment.GetOpenByMode(mode)
				hash := block.Hash()
				for _, p := range payments {
					for _, tx := range block.Transactions() {
						if tx.To() != nil && tx.To().Hex() == p.Account.Address {
							controller.CheckBalanceNotify(&p, tx.Value(), block.Number(), &hash)
						}
					}
					controller.CheckPayment(&p, block.Number(), &hash, nil)

				}

				controller.CheckConfirming(client, block.Number(), mode)
			}
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

	return router
}
