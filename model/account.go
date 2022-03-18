package model

import "github.com/google/uuid"

type Account struct {
	Id         uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	PrivateKey string    `gorm:"type:varchar"`
	Address    string    `gorm:"type:varchar"`
	Used       bool
	PaymentId  uuid.UUID
}
