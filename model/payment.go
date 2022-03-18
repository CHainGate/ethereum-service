package model

import (
	"database/sql/driver"
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"reflect"
	"time"
)

type PaymentIntent struct {
	mode    int
	account Account
	amount  *GormBigInt
}

type Payment struct {
	Id         uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	Account    Account
	UserWallet string
	Mode       string
	//TODO: change to float64
	PriceAmount   GormBigInt `gorm:"type:text"`
	PriceCurrency string
	CreatedAt     time.Time
	updatedAt     time.Time
	PaymentStates []PaymentStatus
}

type PaymentStatus struct {
	Id             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	PaymentId      uuid.UUID
	PayAmount      GormBigInt `gorm:"type:bigint"`
	AmountReceived GormBigInt `gorm:"type:bigint"`
	StatusName     string
	CreatedAt      time.Time
}

type GormBigInt big.Int

func (bi GormBigInt) BigInt() *big.Int {
	bigI := big.Int(bi)
	return &bigI
}

func (bi GormBigInt) Int64() int64 {
	bigI := big.Int(bi)
	return bigI.Int64()
}

func (bi GormBigInt) Bytes() []byte {
	bigI := big.Int(bi)
	return bigI.Bytes()
}

func (bi *GormBigInt) Scan(val interface{}) error {
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
		bigI := new(big.Int).SetInt64(v)
		*bi = GormBigInt(*bigI)
		return nil
	default:
		return fmt.Errorf("bigint: can't convert %s type to *big.Int", reflect.TypeOf(val).Kind())
	}

	bigI, ok := new(big.Int).SetString(data, 10)
	if !ok {
		return fmt.Errorf("bigint can't convert %s to *big.Int", data)
	}
	*bi = GormBigInt(*bigI)
	return nil
}

func (bi GormBigInt) Value() (driver.Value, error) {
	bigI := big.Int(bi)
	return bigI.String(), nil
}
