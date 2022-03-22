package model

import (
	"database/sql/driver"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math/big"
	"reflect"
	"time"
)

type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Payment struct {
	Base
	Account    Account
	AccountID  uuid.UUID `gorm:"type:uuid"`
	UserWallet string
	Mode       string
	//TODO: change to float64
	PriceAmount           *BigInt `gorm:"type:bigint;default:0"`
	PriceCurrency         string
	CurrentPaymentStateId *uuid.UUID     `gorm:"type:uuid"`
	CurrentPaymentState   PaymentState   `gorm:"foreignKey:CurrentPaymentStateId"`
	PaymentStates         []PaymentState `gorm:"<-:false"`
}

/*func (p *Payment) GetActiveState() PaymentState {
	//TODO: is there a better solution?
	sort.Slice(p.PaymentStates, func(i, j int) bool {
		return p.PaymentStates[i].CreatedAt.After(p.PaymentStates[j].CreatedAt)
	})
	return p.PaymentStates[0]
}*/

func (p *Payment) GetActiveAmount() *big.Int {
	return &p.CurrentPaymentState.PayAmount.Int
}

/*
	Updates a paymentstatus to the payment and also sets the new as the current one.
	It reuses the paymentAmount from last status
*/
func (p *Payment) UpdatePaymentState(newState string, balance *big.Int) PaymentState {
	state := PaymentState{
		StatusName:     newState,
		AccountID:      p.AccountID,
		AmountReceived: NewBigInt(balance),
		PayAmount:      p.CurrentPaymentState.PayAmount,
		PaymentID:      p.ID,
	}
	p.CurrentPaymentState = state
	p.PaymentStates = append(p.PaymentStates, state)
	return state
}

/*
	Adds a paymentstatus to the payment and also sets the new as the current one.
*/
func (p *Payment) AddNewPaymentState(newState string, balance *big.Int, payAmount *big.Int) PaymentState {
	state := PaymentState{
		StatusName:     newState,
		AccountID:      p.AccountID,
		AmountReceived: NewBigInt(balance),
		PayAmount:      NewBigInt(payAmount),
		PaymentID:      p.ID,
	}
	p.CurrentPaymentState = state
	p.PaymentStates = append(p.PaymentStates, state)
	return state
}

type PaymentState struct {
	Base
	AccountID      uuid.UUID `gorm:"type:uuid;"`
	PayAmount      *BigInt   `gorm:"type:bigint;default:0"`
	AmountReceived *BigInt   `gorm:"type:bigint;default:0"`
	StatusName     string
	PaymentID      uuid.UUID `gorm:"type:uuid"`
}

func (ps *PaymentState) IsPaid() bool {
	return ps.StatusName == "partially_paid" || ps.StatusName == "waiting"
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

func (bigInt *BigInt) Scan(val interface{}) error {
	if val == nil {
		return nil
	}
	var data string
	switch v := val.(type) {
	case []byte:
		data = string(v)
	case string:
		data = v
	case int64:
		*bigInt = *NewBigIntFromInt(v)
		return nil
	default:
		return fmt.Errorf("bigint: can't convert %s type to *big.Int", reflect.TypeOf(val).Kind())
	}
	bigI, ok := new(big.Int).SetString(data, 10)
	if !ok {
		return fmt.Errorf("not a valid big integer: %s", data)
	}
	bigInt.Int = *bigI
	return nil
}
