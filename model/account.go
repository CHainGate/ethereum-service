package model

import (
	"crypto/ecdsa"
	"ethereum-service/internal/config"
	"ethereum-service/utils"
	"fmt"
	"log"
	"math/big"

	"github.com/CHainGate/backend/pkg/enum"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"gorm.io/gorm"
)

type Account struct {
	Base
	PrivateKey string `gorm:"type:varchar"`
	Address    string `gorm:"type:varchar"`
	Nonce      uint64 `gorm:"type:bigint;default:0"`
	Used       bool
	Payments   []Payment
	Remainder  *BigInt `gorm:"type:numeric(30);default:0"`
	Mode       enum.Mode
}

type IAccountRepository interface {
	GetFree(mode enum.Mode) (*gorm.DB, *Account)
	Create(acc *Account) *Account
	Update(acc *Account) error
}

func CreateAccount(mode enum.Mode) *Account {
	account := Account{}
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	encryptedPrivateKey, err := utils.Encrypt([]byte(config.Opts.PrivateKeySecret), hexutil.Encode(crypto.FromECDSA(privateKey)))
	if err != nil {
		log.Fatal("Unable to encrypt Private Key!", err)
	}
	account.PrivateKey = encryptedPrivateKey

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	fmt.Println("New account with address created. Address: ", address)
	account.Address = address
	account.Remainder = NewBigInt(big.NewInt(0))

	account.Used = true
	account.Mode = mode

	return &account
}
