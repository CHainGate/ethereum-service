package controller

import (
	"context"
	"ethereum-service/internal/config"
	"ethereum-service/internal/repository"
	"ethereum-service/internal/service"
	"ethereum-service/model"
	"ethereum-service/utils"
	"fmt"
	"golang.org/x/exp/slices"
	"log"
	"math/big"
	"time"

	"github.com/CHainGate/backend/pkg/enum"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
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
		Mode:           mode.String(),
		AccountID:      acc.ID,
		Account:        &acc,
		PriceAmount:    priceAmount,
		PriceCurrency:  priceCurrency,
		MerchantWallet: wallet,
	}

	payment.ID = uuid.New()

	val := service.GetETHAmount(payment)
	final := utils.GetWEIFromETH(val)
	err = checkIfAmountIsTooLow(mode, final)
	if err != nil {
		return nil, nil, err
	}

	_, err = repository.Payment.CreatePayment(&payment, final)

	return &payment, final, nil
}

func checkIfAmountIsTooLow(mode enum.Mode, final *big.Int) error {
	fees := big.NewInt(0)
	var client *ethclient.Client
	switch mode {
	case enum.Main:
		client = config.ClientTest
	case enum.Test:
		client = config.ClientTest
	}
	var gasPrice *big.Int
	var err error
	if config.Chain == nil {
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		gasPrice = config.Chain.GasPrice
	}

	if fees.Mul(gasPrice, big.NewInt(21000*2)).Cmp(final) > 0 {
		return fmt.Errorf("requested amount is too low. Fees are: %v", fees)
	}
	return nil
}

func expire(payment *model.Payment, balance *big.Int) {
	payment.Account.Remainder = model.NewBigInt(balance)
	payment.Account.Used = false
	if repository.Account.UpdateAccount(payment.Account) != nil {
		log.Fatalf("Couldn't write wallet to database: %+v\n", &payment.Account)
	}
	if updateState(payment, nil, enum.Expired) != nil {
		return
	}
}

func CheckIfExpired(payment *model.Payment, balance *big.Int) {
	if balance == nil {
		balance = big.NewInt(0).Add(&payment.CurrentPaymentState.AmountReceived.Int, &payment.Account.Remainder.Int)
	}
	index := slices.IndexFunc(payment.PaymentStates, func(ps model.PaymentState) bool { return ps.StatusName == enum.Waiting.String() })
	if payment.PaymentStates[index].CreatedAt.Add(15 * time.Minute).Before(time.Now()) {
		expire(payment, balance)
	}
}

func CheckBalanceNotify(payment *model.Payment, txValue *big.Int, blockNr uint64) {
	balance := big.NewInt(0).Add(&payment.CurrentPaymentState.AmountReceived.Int, txValue)
	if payment.IsPaid(balance) {
		log.Printf("PAYMENT REACHED!!!!")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payment.GetActiveAmount().String())
		payment.ReceivingBlockNr = blockNr
		if updateState(payment, balance, enum.Paid) != nil {
			return
		}
	} else {
		updateState(payment, balance, enum.PartiallyPaid)
		log.Printf("PAYMENT partly paid")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payment.GetActiveAmount().String())
	}
}

func CheckBalanceStartup(client *ethclient.Client, payment *model.Payment) {
	balance, err := GetUserBalanceAt(client, common.HexToAddress(payment.Account.Address), &payment.Account.Remainder.Int)
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
		updateState(payment, balance, enum.PartiallyPaid)
		log.Printf("PAYMENT partly paid")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payment.GetActiveAmount().String())
	} else {
		log.Printf("PAYMENT still not reached Address: %s", payment.Account.Address)
		log.Printf("Current Payment: %s WEI, %s ETH", balance.String(), utils.GetETHFromWEI(balance).String())
		log.Printf("Expected Payment: %s WEI, %s ETH", payment.GetActiveAmount().String(), utils.GetETHFromWEI(payment.GetActiveAmount()).String())
		log.Printf("Please pay additional: %s WEI, %s ETH", big.NewInt(0).Sub(payment.GetActiveAmount(), balance).String(), utils.GetETHFromWEI(payment.GetActiveAmount()).Sub(utils.GetETHFromWEI(balance)).String())
		CheckIfExpired(payment, balance)
	}
}

func HandleUnfinished(client *ethclient.Client, payment *model.Payment) {

}

/*
	Theoretically the best way to check it is, that you check the transaction receipt. Because the user can pay multiple times we would need to check multiple txs.
    Because there is no limit and the user could spam us with a lot of txs and we would run out of API-calls to infura.
    Therefore, it simply checks the balance after 6 blocks if it is still reached.
*/
func IsConfirmed(client *ethclient.Client, p *model.Payment) (bool, error) {
	balance, err := GetBalanceAt(client, common.HexToAddress(p.Account.Address))
	if err != nil {
		return false, err
	}
	return p.IsPaid(balance), nil
}

func HandleConfirming(client *ethclient.Client, payment *model.Payment) *types.Transaction {
	isConfirmed, err := IsConfirmed(client, payment)
	if isConfirmed {
		if updateState(payment, nil, enum.Confirmed) != nil {
			return nil
		}
		tx := forward(client, payment)
		if updateState(payment, nil, enum.Forwarded) != nil {
			return nil
		}
		checkForwardEarnings(client, payment.Account)
		cleanUp(payment)
		return tx
	} else if err != nil {
		log.Printf("Error in getting balance. Acc Address: %v. Try again next confirming round", payment.Account.Address)
	} else {
		log.Printf("Balance not enough anymore. Potential reverted Tx. Checkout blockNr: %v, Acc Address: %v", payment.ReceivingBlockNr, payment.Account.Address)
		finalBalanceOnChaingateWallet, err := GetBalanceAt(client, common.HexToAddress(payment.Account.Address))
		if err != nil {
			log.Printf("Error in getting balance in final recovery. Acc Address: %v", payment.Account.Address)
			return nil
		}
		payment.Account.Remainder = model.NewBigInt(finalBalanceOnChaingateWallet)
		payment.Account.Used = false
		if repository.Account.UpdateAccount(payment.Account) != nil {
			log.Fatalf("Couldn't write wallet to database: %+v\n", &payment.Account)
		}
		updateState(payment, nil, enum.Failed)
	}
	return nil
}

func cleanUp(payment *model.Payment) {
	payment.Account.Used = false
	if repository.Account.UpdateAccount(payment.Account) != nil {
		log.Fatalf("Couldn't write wallet to database: %+v\n", &payment.Account)
	}
	updateState(payment, nil, enum.Finished)
}

func updateState(payment *model.Payment, balance *big.Int, state enum.State) error {
	newState := payment.UpdatePaymentState(state, balance)
	err := service.SendState(payment.ID, newState)
	if err != nil {
		return nil
	}
	repository.Payment.UpdatePaymentState(payment)
	return err
}

/*
	Subtracts the remainder, because this is the CHainGateEarnings
*/
func GetUserBalanceAt(client *ethclient.Client, address common.Address, remainder *big.Int) (*big.Int, error) {
	realBalance, err := GetBalanceAt(client, address)
	if err != nil {
		return nil, err
	}
	return realBalance.Sub(realBalance, remainder), nil
}

// GetBalanceAt
/*
   Never use this Method to check if the user has paid enough, because it doesn't factor in the *Remainder*
*/
func GetBalanceAt(client *ethclient.Client, address common.Address) (*big.Int, error) {
	return client.BalanceAt(context.Background(), address, nil)
}

func forward(client *ethclient.Client, payment *model.Payment) *types.Transaction {
	toAddress := common.HexToAddress(payment.MerchantWallet)
	gasTipCap, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Printf("Couldn't get suggested gasTipCap %v", err)
	}

	var gasPrice *big.Int
	if config.Chain == nil {
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Printf("Couldn't get suggested gasPrice %v", err)
		}
	} else {
		gasPrice = config.Chain.GasPrice
	}

	var chainID *big.Int
	if config.Chain == nil {
		chainID, err = client.NetworkID(context.Background())
		if err != nil {
			log.Printf("Couldn't get networkID %v", err)
		}
	} else {
		chainID = config.Chain.ChainId
	}

	chainGateEarnings := utils.GetChaingateEarnings(&payment.CurrentPaymentState.PayAmount.Int)

	fees := big.NewInt(0).Mul(big.NewInt(21000), gasPrice)
	feesAndChangateEarnings := big.NewInt(0).Add(fees, chainGateEarnings)

	fmt.Printf("gasPrice: %s\n", gasPrice.String())
	fmt.Printf("gasTipCap: %s\n", gasTipCap.String())
	fmt.Printf("chainGateEarnings: %s\n", chainGateEarnings.String())
	fmt.Printf("Fees: %s\n", fees.String())
	finalAmount := big.NewInt(0).Sub(payment.GetActiveAmount(), feesAndChangateEarnings)
	fmt.Printf("finalAmount: %s\n", finalAmount.String())

	gasLimit := uint64(21000)

	// Transaction fees and Gas explained: https://docs.avax.network/learn/platform-overview/transaction-fees
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     payment.Account.Nonce,
		GasFeeCap: gasPrice,  //gasPrice,     // maximum price per unit of gas that the transaction is willing to pay
		GasTipCap: gasTipCap, //tipCap,       // maximum amount above the baseFee of a block that the transaction is willing to pay to be included
		Gas:       gasLimit,
		To:        &toAddress,
		Value:     finalAmount,
	})

	key, err := utils.GetPrivateKey(payment.Account.PrivateKey)
	if err != nil {
		log.Printf("Couldn't get privateKey %v", err)
	}
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
	if err != nil {
		log.Printf("Couldn't get latests signer for ChainID %v", err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}

	_, err = bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		log.Fatalf("Can't wait until transaction is mined %v", err)
	}

	finalBalanceOnChaingateWallet, err := GetBalanceAt(client, common.HexToAddress(payment.Account.Address))
	if err != nil {
		log.Fatalf("Unable to get Balance of chaingate wallet %v", err)
	}

	payment.Account.Remainder = model.NewBigInt(finalBalanceOnChaingateWallet)

	fmt.Printf("tx sent: %s\n", signedTx.Hash().Hex())
	payment.Account.Nonce = payment.Account.Nonce + 1
	payment.ForwardingTransactionHash = signedTx.Hash().String()
	return signedTx
}

func forwardEarnings(client *ethclient.Client, account *model.Account, fees *big.Int, gasPrice *big.Int) *types.Transaction {
	toAddress := common.HexToAddress(config.Opts.TargetWallet)
	gasTipCap, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatal(err)
		return nil
	}

	var chainID *big.Int
	if config.Chain == nil {
		chainID, err = client.NetworkID(context.Background())
		if err != nil {
			log.Fatal(err)
			return nil
		}
	} else {
		chainID = config.Chain.ChainId
	}

	gasLimit := uint64(21000)
	finalAmount := big.NewInt(0).Sub(&account.Remainder.Int, fees)

	// Transaction fees and Gas explained: https://docs.avax.network/learn/platform-overview/transaction-fees
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     account.Nonce,
		GasFeeCap: gasPrice,  //gasPrice,     // maximum price per unit of gas that the transaction is willing to pay
		GasTipCap: gasTipCap, //tipCap,       // maximum amount above the baseFee of a block that the transaction is willing to pay to be included
		Gas:       gasLimit,
		To:        &toAddress,
		Value:     finalAmount,
	})

	key, err := utils.GetPrivateKey(account.PrivateKey)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Printf("Unable to send Transaction %v", err)
		return nil
	}

	_, err = bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		log.Printf("Can't wait until transaction is mined %v", err)
		return nil
	}

	finalBalanceOnChaingateWallet, err := GetBalanceAt(client, common.HexToAddress(account.Address))
	if err != nil {
		log.Printf("Unable to get Balance of chaingate wallet %v", err)
		return nil
	}
	account.Remainder = model.NewBigInt(finalBalanceOnChaingateWallet)

	fmt.Printf("tx sent: %s\n", signedTx.Hash().Hex())
	account.Nonce = account.Nonce + 1
	return signedTx
}

/*
	returns true when the earning were forwarded and the corresponding transaction
*/
func checkForwardEarnings(client *ethclient.Client, account *model.Account) (bool, *types.Transaction) {
	var err error
	var gasPrice *big.Int
	if config.Chain == nil {
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	} else {
		gasPrice = config.Chain.GasPrice
	}

	fees := big.NewInt(0).Mul(big.NewInt(21000), gasPrice)

	factor := new(big.Int)
	factor, ok := factor.SetString(config.Opts.FeeFactor, 10)
	if !ok {
		fmt.Println("SetString: error")
		return false, nil
	}

	earningsForwardThreshold := big.NewInt(0).Mul(fees, factor)
	if earningsForwardThreshold.Cmp(&account.Remainder.Int) > 0 {
		return false, nil
	}

	tx := forwardEarnings(client, account, fees, gasPrice)
	return true, tx
}
