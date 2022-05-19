package model

import (
	"database/sql/driver"
	"fmt"
	"github.com/CHainGate/backend/pkg/enum"
	"math/big"
	"reflect"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type IPaymentRepository interface {
	UpdatePaymentState(payment *Payment)
	CreatePayment(payment *Payment, finalPaymentAmount *big.Int) (*Payment, error)
	GetAllPaymentIntents() []Payment
	GetAllUnfinishedPayments() []Payment
}

type Payment struct {
	Base
	Account        *Account
	AccountID      uuid.UUID `gorm:"type:uuid"`
	MerchantWallet string
	Mode           string
	//TODO: change to float64
	PriceAmount           float64 `gorm:"type:numeric(30,15);default:0"`
	PriceCurrency         string
	CurrentPaymentStateId *uuid.UUID     `gorm:"type:uuid"`
	CurrentPaymentState   PaymentState   `gorm:"foreignKey:CurrentPaymentStateId"`
	PaymentStates         []PaymentState `gorm:"<-:false"`
}

func (p *Payment) GetActiveAmount() *big.Int {
	return &p.CurrentPaymentState.PayAmount.Int
}

/*
	Updates a paymentstate to the payment and also sets the new as the current one.
	It reuses the paymentAmount from last status
*/
func (p *Payment) UpdatePaymentState(newState enum.State, balance *big.Int) PaymentState {
	if balance == nil {
		balance = &p.CurrentPaymentState.AmountReceived.Int
	}
	state := PaymentState{
		StatusName:     newState.String(),
		AccountID:      p.AccountID,
		AmountReceived: NewBigInt(balance),
		PayAmount:      p.CurrentPaymentState.PayAmount,
		PaymentID:      p.ID,
	}
	p.CurrentPaymentState = state
	p.PaymentStates = append(p.PaymentStates, state)

	return state
}

func (p *Payment) IsNewlyPartlyPaid(balance *big.Int) bool {
	return p.CurrentPaymentState.IsWaitingForPayment() && balance.Uint64() > 0 && balance.Cmp(&p.CurrentPaymentState.AmountReceived.Int) > 0
}

func (p *Payment) IsPaid(balance *big.Int) bool {
	return balance.Cmp(p.GetActiveAmount()) >= 0
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
	PayAmount      *BigInt   `gorm:"type:numeric(30);default:0"`
	AmountReceived *BigInt   `gorm:"type:numeric(30);default:0"`
	StatusName     string
	PaymentID      uuid.UUID `gorm:"type:uuid"`
}

func (ps *PaymentState) IsWaitingForPayment() bool {
	return ps.StatusName == "partially_paid" || ps.StatusName == "waiting"
}

func (ps *PaymentState) IsPaid() bool {
	return ps.StatusName == "paid"
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
