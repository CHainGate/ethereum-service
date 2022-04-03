package model

type Account struct {
	Base
	PrivateKey string `gorm:"type:varchar"`
	Address    string `gorm:"type:varchar"`
	Nonce      uint64 `gorm:"type:bigint;default:0"`
	Used       bool
	Payments   []Payment
	Remainder  *BigInt `gorm:"type:numeric(30);default:0"`
}
