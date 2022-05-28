package main

import (
	"context"
	"ethereum-service/database"
	"ethereum-service/internal/config"
	"ethereum-service/internal/controller"
	repository "ethereum-service/internal/repository"
	"ethereum-service/openApi"
	"ethereum-service/services"
	"github.com/CHainGate/backend/pkg/enum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
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

	go listenToEthChain(config.ClientMain, enum.Main)
	go listenToEthChain(config.ClientTest, enum.Test)
	log.Printf("listing on port %v", 9000)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(9000), router))
}

/*
   Recovery
*/
func checkAllAddresses() {
	payments := repository.Payment.GetAll()
	for _, s := range payments {
		switch s.Mode {
		case enum.Main:
			go controller.CheckBalanceStartup(config.ClientMain, &s)
		case enum.Test:
			go controller.CheckBalanceStartup(config.ClientTest, &s)
		default:
			log.Fatal("Mode not supported!")
		}
	}
}

func checkConfirming(client *ethclient.Client, currentBlockNr uint64, mode enum.Mode) {
	payments := repository.Payment.GetAllConfirming(mode)
	for _, p := range payments {
		// This could be maybe done via SQL query
		log.Printf("blocknr: %v", p.LastReceivingBlockNr)
		log.Printf("currentBlockNr: %v", currentBlockNr)
		log.Printf("isittrue: %v", p.LastReceivingBlockNr == 0 || currentBlockNr > p.LastReceivingBlockNr+6)
		if p.LastReceivingBlockNr == 0 || currentBlockNr >= p.LastReceivingBlockNr+6 {
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

			payments := repository.Payment.GetByMode(mode)
			for _, p := range payments {
				for _, tx := range block.Transactions() {
					if tx.To() != nil && tx.To().Hex() == p.Account.Address {
						controller.CheckBalanceNotify(&p, tx.Value(), block.Number().Uint64(), block.Hash())
					}
				}

				controller.CheckIfExpired(&p, nil)
			}

			checkConfirming(client, block.Number().Uint64(), mode)
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
