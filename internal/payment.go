package internal

import (
	"crypto/ecdsa"
	"errors"
	"ethereum-service/database"
	"ethereum-service/internal/config"
	"ethereum-service/model"
	"fmt"
	"gorm.io/gorm"
	"log"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	Payments []model.Payment
)

func GetAccount() (model.Account, error) {
	return getFreeAccount()
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

	account.Used = true
	result := database.DB.Create(&account) // pass pointer of data to Create
	if result.Error != nil {
		log.Fatal(result.Error)
	}

	return account
}

func getFreeAccount() (model.Account, error) {
	acc := model.Account{}
	result := database.DB.Where("used = ?", "false").First(&acc)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			acc = createAccount()
		} else {
			return model.Account{}, result.Error
		}
	} else {
		acc.Used = true
		result = database.DB.Save(&acc)
		if result.Error != nil {
			return model.Account{}, result.Error
		}
	}
	return acc, nil
}

func getAccountFromOptsPrivateKey() (model.Account, error) {
	if config.Opts.AccountPrivateKey != "" {
		pk, err := crypto.HexToECDSA(config.Opts.AccountPrivateKey)
		if err != nil {
			log.Fatal(err)
		}

		acc := model.Account{
			PrivateKey: config.Opts.AccountPrivateKey,
			Address:    crypto.PubkeyToAddress(pk.PublicKey).Hex(),
			Used:       true,
		}

		return acc, nil
	}
	return model.Account{}, fmt.Errorf("unable to get free address")
}

func GetPrivateKey(key string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(key)
}

func GetPrivateKeyString(key *ecdsa.PrivateKey) string {
	return hexutil.Encode(crypto.FromECDSA(key))
}
