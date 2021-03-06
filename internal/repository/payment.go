package repository

import (
	"ethereum-service/model"
	"log"
	"math/big"

	"github.com/CHainGate/backend/pkg/enum"

	"gorm.io/gorm"
)

type PaymentRepository struct {
	DB *gorm.DB
}

func InitPayment(db *gorm.DB) {
	Payment = &PaymentRepository{DB: db}
}

var (
	Payment model.IPaymentRepository
)

func (r *PaymentRepository) UpdatePaymentState(payment *model.Payment) {
	r.DB.Save(&payment)
}

func (r *PaymentRepository) Create(payment *model.Payment, finalPaymentAmount *big.Int) (*model.Payment, error) {
	payment.AddNewPaymentState(enum.Waiting, big.NewInt(0), finalPaymentAmount)
	result := r.DB.Create(&payment)
	if result.Error != nil {
		log.Printf("Error by creating new Payment %v", result.Error)
		return nil, result.Error
	}
	return payment, nil
}

func (r *PaymentRepository) GetAllOpen() []model.Payment {
	var payments []model.Payment
	r.DB.
		Preload("Account").
		Preload("CurrentPaymentState").
		Preload("PaymentStates").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"state_id\" IN ?", []enum.State{enum.Waiting, enum.PartiallyPaid}).
		Find(&payments)
	return payments
}

func (r *PaymentRepository) GetOpenByMode(mode enum.Mode) []model.Payment {
	var payments []model.Payment
	r.DB.
		Preload("Account").
		Where("mode = ?", mode).
		Preload("CurrentPaymentState").
		Preload("PaymentStates").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"state_id\" IN ?", []enum.State{enum.Waiting, enum.PartiallyPaid}).
		Find(&payments)
	return payments
}

func (r *PaymentRepository) GetConfirming(mode enum.Mode) []model.Payment {
	var payments []model.Payment
	r.DB.
		Where("mode = ?", mode).
		Preload("Account").
		Preload("CurrentPaymentState").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"state_id\" IN ?", []enum.State{enum.Paid}).
		Find(&payments)
	return payments
}

func (r *PaymentRepository) GetFinishing(mode enum.Mode) []model.Payment {
	var payments []model.Payment
	r.DB.
		Where("mode = ?", mode).
		Preload("Account").
		Preload("CurrentPaymentState").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"state_id\" IN ?", []enum.State{enum.Forwarded}).
		Find(&payments)
	return payments
}
