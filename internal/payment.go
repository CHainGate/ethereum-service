package internal

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"ethereum-service/internal/dbaccess"
	"ethereum-service/model"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"strings"

	"gorm.io/gorm"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

func GetAccount() (model.Account, error) {
	return getFreeAccount()
}

func createAccount() *model.Account {
	account := model.Account{}
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	account.PrivateKey = hexutil.Encode(crypto.FromECDSA(privateKey))

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	fmt.Println("New account with address created. Address: ", address)
	account.Address = address
	account.Remainder = model.NewBigInt(big.NewInt(0))

	account.Used = true

	return &account
}

func getFreeAccount() (model.Account, error) {
	result, acc := dbaccess.GetFreeAccount()
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			acc = createAccount()
			dbaccess.CreateAccount(acc)
		} else {
			return model.Account{}, result.Error
		}
	} else {
		acc.Used = true
		err := dbaccess.UpdateAccount(acc)
		if err != nil {
			return model.Account{}, result.Error
		}
	}
	return *acc, nil
}

func getETHFromWEI(amount *big.Int) *big.Float {
	return big.NewFloat(0).Quo(new(big.Float).SetInt(amount), big.NewFloat(1000000000000000000))
}

func CheckBalance(client *ethclient.Client, payment model.Payment) {
	balance, err := GetUserBalanceAt(client, common.HexToAddress(payment.Account.Address), &payment.Account.Remainder.Int)
	if err != nil {
		log.Fatal(err)
	}

	if payment.IsPaid(balance) {
		log.Printf("PAYMENT REACHED!!!!")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payment.GetActiveAmount().String())
		forward(client, &payment, nil, nil)
		err = dbaccess.UpdateAccount(&payment.Account)
		if err != nil {
			log.Fatalf("Couldn't write wallet to database: %+v\n", &payment.Account)
		}
		dbaccess.UpdatePaymentState(payment, "paid", balance)
	} else if payment.IsNewlyPartlyPaid(balance) {
		dbaccess.UpdatePaymentState(payment, "partially_paid", balance)
		log.Printf("PAYMENT partly paid")
		log.Printf("Current Payment: %s \n Expected Payment: %s", balance.String(), payment.GetActiveAmount().String())
	} else {
		log.Printf("PAYMENT still not reached Address: %s", payment.Account.Address)
		log.Printf("Current Payment: %s WEI, %s ETH", balance.String(), getETHFromWEI(balance).String())
		log.Printf("Expected Payment: %s WEI, %s ETH", payment.GetActiveAmount().String(), getETHFromWEI(payment.GetActiveAmount()).String())
		log.Printf("Please pay additional: %s WEI, %s ETH", big.NewInt(0).Sub(payment.GetActiveAmount(), balance).String(), big.NewFloat(0).Sub(getETHFromWEI(payment.GetActiveAmount()), getETHFromWEI(balance)).String())
	}
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

func forward(client *ethclient.Client, payment *model.Payment, chainID *big.Int, gasPrice *big.Int) *types.Transaction {
	nonce, err := client.PendingNonceAt(context.Background(), common.HexToAddress(payment.Account.Address))
	if err != nil {
		log.Fatal(err)
	}

	if gasPrice == nil {
		gasPrice, err = client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	}

	gasTipCap, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	if chainID == nil {
		chainID, err = client.NetworkID(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	}

	gasLimit := uint64(21000)

	chainGateEarnings := getChaingateEarnings(*payment, 1)
	payment.Account.Remainder.Add(&payment.Account.Remainder.Int, chainGateEarnings)

	fees := big.NewInt(0).Mul(big.NewInt(21000), gasPrice)
	feesAndChangateEarnings := big.NewInt(0).Add(fees, chainGateEarnings)

	fmt.Printf("gasPrice: %s\n", gasPrice.String())
	fmt.Printf("chainGateEarnings: %s\n", chainGateEarnings.String())
	fmt.Printf("Fees: %s\n", fees.String())
	finalAmount := big.NewInt(0).Sub(payment.GetActiveAmount(), feesAndChangateEarnings)
	fmt.Printf("finalAmount: %s\n", finalAmount.String())

	toAddress := common.HexToAddress(payment.UserWallet)
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

	key, err := GetPrivateKey(payment.Account.PrivateKey)
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

	fmt.Printf("tx sent: %s\n", signedTx.Hash().Hex())
	payment.Account.Used = false
	payment.Account.Nonce = payment.Account.Nonce + 1
	return signedTx
}

func getChaingateEarnings(payment model.Payment, percent int64) *big.Int {
	chainGatePercent := big.NewInt(percent)
	chainGateEarnings := big.NewInt(0)
	mul := chainGateEarnings.Mul(&payment.CurrentPaymentState.PayAmount.Int, chainGatePercent)
	final := mul.Div(mul, big.NewInt(100))
	return final
}

func GetPrivateKey(key string) (*ecdsa.PrivateKey, error) {
	if strings.HasPrefix(key, "0x") {
		key = key[2:]
	}
	return crypto.HexToECDSA(key)
}

func GetPrivateKeyString(key *ecdsa.PrivateKey) string {
	return hexutil.Encode(crypto.FromECDSA(key))
}
