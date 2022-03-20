package model

import (
	"database/sql/driver"
	"fmt"
	"gorm.io/gorm"
	"math/big"
	"sort"
)

type PaymentIntent struct {
	mode    int
	account Account
	amount  *BigInt
}

type Payment struct {
	gorm.Model
	Account    Account
	AccountID  uint
	UserWallet string
	Mode       string
	//TODO: change to float64
	PriceAmount   *BigInt `gorm:"type:text"`
	PriceCurrency string
	PaymentStates []PaymentStatus
}

func (p *Payment) GetActiveState() PaymentStatus {
	//TODO: is there a better solution?
	sort.Slice(p.PaymentStates, func(i, j int) bool {
		return p.PaymentStates[i].CreatedAt.After(p.PaymentStates[j].CreatedAt)
	})
	return p.PaymentStates[0]
}

func (p *Payment) GetActiveAmount() *big.Int {
	return &p.GetActiveState().PayAmount.Int
}

type PaymentStatus struct {
	gorm.Model
	AccountID      uint
	PayAmount      *BigInt `gorm:"type:bigint"`
	AmountReceived *BigInt `gorm:"type:bigint"`
	StatusName     string
	PaymentID      uint
}

type BigInt struct {
	big.Int
}

func NewBigIntFromInt(value int64) *BigInt {
	x := new(big.Int).SetInt64(value)
	return NewBigInt(x)
}

func NewBigInt(value *big.Int) *BigInt {
	return &BigInt{Int: *value}
}

func (bigInt *BigInt) Value() (driver.Value, error) {
	if bigInt == nil {
		return "null", nil
	}
	return bigInt.String(), nil
}

func (bigInt *BigInt) Scan(v interface{}) error {
	value, ok := v.(string)
	if !ok {
		return fmt.Errorf("type error, %v", v)
	}
	if string(value) == "null" {
		return nil
	}
	data, ok := new(big.Int).SetString(string(value), 10)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", value)
	}
	bigInt.Int = *data
	return nil
}
