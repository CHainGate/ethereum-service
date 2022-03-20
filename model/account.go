package model

import (
	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
	PrivateKey string `gorm:"type:varchar"`
	Address    string `gorm:"type:varchar"`
	Used       bool
	Payments   []Payment
	Remainder  *BigInt `gorm:"default:0"`
}
