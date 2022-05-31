package controller

import (
	"ethereum-service/internal/config"
	"ethereum-service/internal/repository"
	"ethereum-service/internal/service"
	"ethereum-service/model"
	"ethereum-service/utils"
	"fmt"
	"github.com/CHainGate/backend/pkg/enum"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
)

func CreatePayment(mode enum.Mode, priceAmount float64, priceCurrency string, wallet string) (*model.Payment, *big.Int, error) {
	acc, err := GetAccount(mode)

	if err != nil {
		return nil, nil, fmt.Errorf("unable to get free address")
	}

	payment := model.Payment{
		Mode:           mode,
		AccountID:      acc.ID,
		Account:        &acc,
		PriceAmount:    priceAmount,
		PriceCurrency:  priceCurrency,
		MerchantWallet: wallet,
	}

	payment.ID = uuid.New()

	val := service.GetETHAmount(payment)
	final := utils.GetWEIFromETH(val)
	err = utils.CheckIfAmountIsTooLow(mode, final)
	if err != nil {
		return nil, nil, err
	}

	_, err = repository.Payment.Create(&payment, final)

	return &payment, final, nil
}

// CheckPayment
/*
  Checks if payment is expired. If it is expired it first checks the balance to make sure it isn't paid.
  Care for internal transactions.
*/
func CheckPayment(payment *model.Payment, blockNr *big.Int, txHash common.Hash) {
	if utils.CheckIfExpired(payment, nil) {
		client := utils.GetClientByMode(payment.Mode)
		balance, err := utils.GetUserBalanceAt(client, common.HexToAddress(payment.Account.Address), &payment.Account.Remainder.Int)
		if err != nil {
			log.Printf("Error by getting balance %v", err)
		}
		if payment.IsPaid(balance) {
			Pay(payment, nil, blockNr, txHash)
		}
		Expire(payment, nil)
	}
}

func CheckBalanceNotify(payment *model.Payment, txValue *big.Int, blockNr *big.Int, blockHash common.Hash) {
	balance := big.NewInt(0).Add(&payment.CurrentPaymentState.AmountReceived.Int, txValue)
	if payment.IsPaid(balance) {
		var paid bool
		paid, balance = utils.IsPaidOnChain(payment)
		if paid {
			Pay(payment, balance, blockNr, blockHash)
		} else {
			Fail(payment, balance)
		}
	} else {
		updateState(payment, balance, enum.PartiallyPaid)
		log.Printf("PAYMENT partly paid")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payment.GetActiveAmount().String())
	}
}

func CheckIncomingBlocks(client *ethclient.Client, currentBlockNr *big.Int, mode enum.Mode) {
	payments := repository.Payment.GetConfirming(mode)
	for _, p := range payments {
		hasBlockEnoughConfirmations := big.NewInt(0).Add(&p.LastReceivingBlockNr.Int, big.NewInt(config.Opts.IncomingBlockConfirmations)).Cmp(currentBlockNr) <= 0
		if p.LastReceivingBlockNr.Cmp(big.NewInt(0)) == 0 || hasBlockEnoughConfirmations {
			p.ForwardingBlockNr = model.NewBigInt(currentBlockNr)
			go HandleConfirming(client, &p)
		}
	}
}

func CheckOutgoingTx(client *ethclient.Client, currentBlockNr *big.Int, mode enum.Mode) {
	payments := repository.Payment.GetFinishing(mode)
	for _, p := range payments {
		txHash := common.HexToHash(p.ForwardingTransactionHash)
		isConfirmed, err := utils.IsTxConfirmed(client, common.HexToHash(p.ForwardingTransactionHash), currentBlockNr)
		if isConfirmed {
			finish(&p)
		} else if err == utils.TxFailed {
			log.Printf("Potential reverted Tx. Checkout blockNr: %v, Acc Address: %v", txHash, p.Account.Address)
			finalBalanceOnChaingateWallet, err := utils.GetBalanceAt(client, common.HexToAddress(p.Account.Address))
			if err != nil {
				log.Printf("Error in getting balance in final recovery. Acc Address: %v", p.Account.Address)
			}
			Fail(&p, finalBalanceOnChaingateWallet)
		} else if err != nil {
			log.Printf("Error in confirming tx. Acc Address: %v. Try again next confirming round", p.Account.Address)
		}
	}
}

func CheckConfirming(client *ethclient.Client, currentBlockNr *big.Int, mode enum.Mode) {
	go CheckIncomingBlocks(client, currentBlockNr, mode)
	go CheckOutgoingTx(client, currentBlockNr, mode)
}

func HandleConfirming(client *ethclient.Client, payment *model.Payment) *types.Transaction {
	var isConfirmed bool
	var err error
	// When no Tx hash is set do no confirming. This can happen when the service does a recovery and only check the open balances
	if payment.LastReceivingBlockHash == "" {
		isConfirmed = true
	} else {
		isConfirmed, err = utils.IsBlockConfirmed(client, common.HexToHash(payment.LastReceivingBlockHash))
	}

	if isConfirmed {
		return confirm(client, payment)
	} else if err != nil {
		log.Printf("Error in getting balance. Acc Address: %v. Try again next confirming round", payment.Account.Address)
	} else {
		log.Printf("Block doesn't exist anymore. Potential reverted Tx. Checkout blockNr: %v, Acc Address: %v", payment.LastReceivingBlockNr, payment.Account.Address)
		finalBalanceOnChaingateWallet, err := utils.GetBalanceAt(client, common.HexToAddress(payment.Account.Address))
		if err != nil {
			log.Printf("Error in getting balance in final recovery. Acc Address: %v", payment.Account.Address)
			return nil
		}
		Fail(payment, finalBalanceOnChaingateWallet)
	}
	return nil
}

func Pay(payment *model.Payment, balance *big.Int, blockNr *big.Int, blockHash common.Hash) bool {
	log.Printf("PAYMENT REACHED!!!!")
	log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payment.GetActiveAmount().String())
	payment.LastReceivingBlockNr = model.NewBigInt(blockNr)
	payment.LastReceivingBlockHash = blockHash.String()
	if updateState(payment, balance, enum.Paid) != nil {
		return true
	}
	return false
}

func Expire(payment *model.Payment, balance *big.Int) {
	payment.Account.Remainder = model.NewBigInt(balance)
	payment.Account.Used = false
	if repository.Account.Update(payment.Account) != nil {
		log.Fatalf("Couldn't write wallet to database: %+v\n", &payment.Account)
	}
	if updateState(payment, nil, enum.Expired) != nil {
		return
	}
}

func Fail(payment *model.Payment, balance *big.Int) {
	payment.Account.Remainder = model.NewBigInt(balance)
	payment.Account.Used = false
	if repository.Account.Update(payment.Account) != nil {
		log.Fatalf("Couldn't write wallet to database: %+v\n", &payment.Account)
	}
	if updateState(payment, nil, enum.Failed) != nil {
		return
	}
}

func confirm(client *ethclient.Client, payment *model.Payment) *types.Transaction {
	if updateState(payment, nil, enum.Confirmed) != nil {
		return nil
	}
	tx := utils.Forward(client, payment)
	if tx == nil {
		return nil
	}
	if updateState(payment, nil, enum.Forwarded) != nil {
		return nil
	}
	utils.CheckForwardEarnings(client, payment.Account)
	return tx
}

func finish(payment *model.Payment) {
	payment.Account.Used = false
	if repository.Account.Update(payment.Account) != nil {
		log.Fatalf("Couldn't write wallet to database: %+v\n", &payment.Account)
	}
	updateState(payment, nil, enum.Finished)
}

func CheckBalanceStartup(client *ethclient.Client, payment *model.Payment) {
	balance, err := utils.GetUserBalanceAt(client, common.HexToAddress(payment.Account.Address), &payment.Account.Remainder.Int)
	if err != nil {
		log.Printf("Error by getting balance %v", err)
	}

	if payment.IsPaid(balance) {
		log.Printf("PAYMENT REACHED!!!!")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payment.GetActiveAmount().String())
		if updateState(payment, balance, enum.Paid) != nil {
			return
		}
	} else if payment.IsNewlyPartlyPaid(balance) {
		log.Printf("PAYMENT partly paid")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payment.GetActiveAmount().String())
		updateState(payment, balance, enum.PartiallyPaid)
	} else {
		log.Printf("PAYMENT still not reached Address: %s", payment.Account.Address)
		log.Printf("Current Payment: %s WEI, %s ETH", balance.String(), utils.GetETHFromWEI(balance).String())
		log.Printf("Expected Payment: %s WEI, %s ETH", payment.GetActiveAmount().String(), utils.GetETHFromWEI(payment.GetActiveAmount()).String())
		log.Printf("Please pay additional: %s WEI, %s ETH", big.NewInt(0).Sub(payment.GetActiveAmount(), balance).String(), utils.GetETHFromWEI(payment.GetActiveAmount()).Sub(utils.GetETHFromWEI(balance)).String())
		if utils.CheckIfExpired(payment, balance) {
			Expire(payment, balance)
		}
	}
}

func updateState(payment *model.Payment, balance *big.Int, state enum.State) error {
	newState := payment.UpdatePaymentState(state, balance)
	err := service.SendState(payment.ID, newState, payment.ForwardingTransactionHash)
	if err != nil {
		return nil
	}
	repository.Payment.UpdatePaymentState(payment)
	return err
}
