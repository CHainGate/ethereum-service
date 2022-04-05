package model

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"gorm.io/gorm"
	"log"
	"math/big"
)

type Account struct {
	Base
	PrivateKey string `gorm:"type:varchar"`
	Address    string `gorm:"type:varchar"`
	Nonce      uint64 `gorm:"type:bigint;default:0"`
	Used       bool
	Payments   []Payment
	Remainder  *BigInt `gorm:"type:numeric(30);default:0"`
}

type IAccountRepository interface {
	GetFreeAccount() (*gorm.DB, *Account)
	CreateAccount(acc *Account) *Account
	UpdateAccount(acc *Account) error
}

func CreateAccount() *Account {
	account := Account{}
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
	account.Remainder = NewBigInt(big.NewInt(0))

	account.Used = true

	return &account
}
