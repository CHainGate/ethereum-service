package bc

import (
	"context"
	"errors"
	"ethereum-service/internal/config"
	"ethereum-service/model"
	"ethereum-service/utils"
	"fmt"
	"github.com/CHainGate/backend/pkg/enum"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"time"
)

var TxFailed = errors.New("tx failed")

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

/*
	Theoretically the best way to check it is, that you check the transaction receipt. Because the user can pay multiple times we would need to check multiple tx's.
    Because there is no limit and the user could spam with a lot of tx's and run out of API-calls to infura.
    Therefore, this method checks the block. If older blocks gets reverted this is also not valid anymore.
*/
func IsBlockConfirmed(client *ethclient.Client, blockHash common.Hash) (bool, error) {
	block, err := client.BlockByHash(context.Background(), blockHash)
	if err != nil {
		return false, err
	}
	if err == ethereum.NotFound {
		return false, nil
	}
	return block != nil, nil
}

func IsTxConfirmed(client *ethclient.Client, txHash common.Hash, blockNr *big.Int) (bool, error) {
	tx, err := client.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		return false, err
	}
	if tx.Status == 0 {
		return false, TxFailed
	}
	hasEnoughConfirmations := big.NewInt(0).Add(tx.BlockNumber, big.NewInt(config.Opts.OutgoingTxConfirmations)).Cmp(blockNr) <= 0
	return hasEnoughConfirmations, nil
}

/*
	Check safely is paid, because it checks the balance on the address. Makes an API-Call to Ethereum.
*/
func IsPaidOnChain(payment *model.Payment) (bool, *big.Int) {
	client := GetClientByMode(payment.Mode)
	balance, err := GetUserBalanceAt(client, common.HexToAddress(payment.Account.Address), &payment.Account.Remainder.Int)
	if err != nil {
		log.Printf("Error by getting balance %v", err)
	}
	return payment.IsPaid(balance), balance
}

func CheckIfExpired(payment *model.Payment) bool {
	return payment.CreatedAt.Add(15 * time.Minute).Before(time.Now())
}

func CheckIfAmountIsTooLow(mode enum.Mode, final *big.Int) error {
	fees := big.NewInt(0)
	client := GetClientByMode(mode)
	var gasPrice *big.Int
	var err error
	if config.Chain == nil {
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Println(err)
		}
	} else {
		gasPrice = config.Chain.GasPrice
	}

	if fees.Mul(gasPrice, big.NewInt(21000*2)).Cmp(final) > 0 {
		return fmt.Errorf("requested amount is too low. Fees are: %v", fees)
	}
	return nil
}

func GetClientByMode(mode enum.Mode) *ethclient.Client {
	var client *ethclient.Client
	switch mode {
	case enum.Main:
		client = config.ClientTest
	case enum.Test:
		client = config.ClientTest
	}
	return client
}

func Forward(client *ethclient.Client, payment *model.Payment) *types.Transaction {
	toAddress := common.HexToAddress(payment.MerchantWallet)
	var gasPrice *big.Int
	var err error
	if config.Chain == nil {
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Printf("Couldn't get suggested gasPrice %v", err)
		}
	} else {
		gasPrice = config.Chain.GasPrice
	}
	chainGateEarnings := utils.GetChaingateEarnings(&payment.CurrentPaymentState.PayAmount.Int)
	fees := big.NewInt(0).Mul(big.NewInt(21000), gasPrice)
	feesAndChangateEarnings := big.NewInt(0).Add(fees, chainGateEarnings)
	finalAmount := big.NewInt(0).Sub(payment.GetActiveAmount(), feesAndChangateEarnings)

	signedTx := makeTransaction(client, payment.Account, gasPrice, finalAmount, toAddress)

	payment.ForwardingTransactionHash = signedTx.Hash().String()
	return signedTx
}

func ForwardEarnings(client *ethclient.Client, account *model.Account, fees *big.Int, gasPrice *big.Int) *types.Transaction {
	finalAmount := big.NewInt(0).Sub(&account.Remainder.Int, fees)
	toAddress := common.HexToAddress(config.Opts.TargetWallet)
	return makeTransaction(client, account, gasPrice, finalAmount, toAddress)
}

func makeTransaction(client *ethclient.Client, account *model.Account, gasPrice *big.Int, finalAmount *big.Int, toAddress common.Address) *types.Transaction {
	gasTipCap, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Println(err)
		return nil
	}

	var chainID *big.Int
	if config.Chain == nil {
		chainID, err = client.NetworkID(context.Background())
		if err != nil {
			log.Println(err)
			return nil
		}
	} else {
		chainID = config.Chain.ChainId
	}

	gasLimit := uint64(21000)

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
		log.Println(err)
		return nil
	}
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
	if err != nil {
		log.Println(err)
		return nil
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err == core.ErrNonceTooLow {
		nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(account.Address))
		if err != nil {
			log.Printf("Couldn't get Nonce %v", err)
			return nil
		}
		account.Nonce = nonce
		makeTransaction(client, account, gasPrice, finalAmount, toAddress)
	}
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
	fmt.Printf("tx sent: %s\n", signedTx.Hash().Hex())
	account.Remainder = model.NewBigInt(finalBalanceOnChaingateWallet)
	account.Nonce = account.Nonce + 1
	return signedTx
}

/*
	returns true when the earning were forwarded and the corresponding transaction
*/
func CheckForwardEarnings(client *ethclient.Client, account *model.Account) (bool, *types.Transaction) {
	var err error
	var gasPrice *big.Int
	if config.Chain == nil {
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Println(err)
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

	tx := ForwardEarnings(client, account, fees, gasPrice)
	return true, tx
}
