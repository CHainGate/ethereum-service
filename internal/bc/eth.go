package bc

import (
	"context"
	"crypto/ecdsa"
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
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

var TxFailed = errors.New("tx failed")

func GetPrivateKey(key string) (*ecdsa.PrivateKey, error) {
	if strings.HasPrefix(key, "0x") {
		key = key[2:]
	}
	return crypto.HexToECDSA(key)
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

/*
	Theoretically the best way to check it is, that you check the transaction receipt. Because the user can pay multiple times we would need to check multiple txs.
    Because there is no limit and the user could spam us with a lot of tx's, therefore we would run out of API-calls to infura.
    Therefore, it simply checks the balance after X blocks if it is still reached.
	It checks the block therefore if older blocks gets reverted it is also not valid anymore.
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

	key, err := GetPrivateKey(payment.Account.PrivateKey)
	if err != nil {
		log.Printf("Couldn't get privateKey %v", err)
	}
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), key)
	if err != nil {
		log.Printf("Couldn't get latests signer for ChainID %v", err)
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err == core.ErrNonceTooLow {
		nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(payment.Account.Address))
		if err != nil {
			log.Printf("Couldn't get Nonce %v", err)
			return nil
		}
		payment.Account.Nonce = nonce
		Forward(client, payment)
	}
	if err != nil {
		log.Printf("Couldn't send Transaction %v, %v", signedTx.Hash(), err)
		return nil
	}

	_, err = bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		log.Printf("Can't wait until transaction is mined %v", err)
	}

	finalBalanceOnChaingateWallet, err := GetBalanceAt(client, common.HexToAddress(payment.Account.Address))
	if err != nil {
		log.Printf("Unable to get Balance of chaingate wallet %v", err)
	}

	payment.Account.Remainder = model.NewBigInt(finalBalanceOnChaingateWallet)

	fmt.Printf("tx sent: %s\n", signedTx.Hash().Hex())
	payment.Account.Nonce = payment.Account.Nonce + 1
	payment.ForwardingTransactionHash = signedTx.Hash().String()
	return signedTx
}

func ForwardEarnings(client *ethclient.Client, account *model.Account, fees *big.Int, gasPrice *big.Int) *types.Transaction {
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

	key, err := GetPrivateKey(account.PrivateKey)
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
func CheckForwardEarnings(client *ethclient.Client, account *model.Account) (bool, *types.Transaction) {
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

	tx := ForwardEarnings(client, account, fees, gasPrice)
	return true, tx
}
