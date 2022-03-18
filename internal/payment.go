package internal

import (
	"crypto/ecdsa"
	"ethereum-service/database"
	"ethereum-service/internal/config"
	"ethereum-service/model"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
)

var (
	PaymentIntents []model.Payment
	accounts       []model.Account
)

func GetAccount() model.Account {
	acc, err := getFreeAccount()
	if err != nil {
		acc = createAccount()
	}
	return acc
}

func createAccount() model.Account {
	account := model.Account{}
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	account.PrivateKey = hexutil.Encode(crypto.FromECDSA(privateKey))
	fmt.Println("SAVE BUT DO NOT SHARE THIS (Private Key):", account.PrivateKey)

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	fmt.Println("Public Key:", hexutil.Encode(crypto.FromECDSAPub(publicKeyECDSA)))

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	fmt.Println("Address:", address)
	account.Address = address

	result := database.DB.Create(&account) // pass pointer of data to Create
	if result.Error != nil {
		log.Fatal(result.Error)
	}

	accounts = append(accounts, account)
	return account
}

func getFreeAccount() (model.Account, error) {
	if config.Opts.AccountPrivateKey != "" {
		pk, err := crypto.HexToECDSA(config.Opts.AccountPrivateKey)
		if err != nil {
			log.Fatal(err)
		}

		acc := model.Account{
			PrivateKey: config.Opts.AccountPrivateKey,
			Address:    crypto.PubkeyToAddress(pk.PublicKey).Hex(),
		}

		return acc, nil
	}
	return model.Account{}, fmt.Errorf("unable to get free address")
}

func getPrivateKey(key string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(key)
}

func getPrivateKeyString(key *ecdsa.PrivateKey) string {
	return hexutil.Encode(crypto.FromECDSA(key))
}
