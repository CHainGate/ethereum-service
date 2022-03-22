package model

type Account struct {
	Base
	PrivateKey string `gorm:"type:varchar"`
	Address    string `gorm:"type:varchar"`
	Used       bool
	Payments   []Payment
	Remainder  *BigInt `gorm:"type:bigint;default:0"`
}
