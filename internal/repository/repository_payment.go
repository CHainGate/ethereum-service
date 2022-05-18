package repository

import (
	"ethereum-service/model"
	"github.com/CHainGate/backend/pkg/enum"
	"log"
	"math/big"

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

func (r *PaymentRepository) CreatePayment(payment *model.Payment, finalPaymentAmount *big.Int) (*model.Payment, error) {
	payment.AddNewPaymentState("waiting", big.NewInt(0), finalPaymentAmount)
	result := r.DB.Create(&payment)
	if result.Error != nil {
		log.Fatal(result.Error)
		return nil, result.Error
	}
	return payment, nil
}

func (r *PaymentRepository) GetAllPaymentIntents() []model.Payment {
	var payments []model.Payment
	r.DB.
		Preload("Account").
		Preload("CurrentPaymentState").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"status_name\" IN ?", []string{"waiting", "partially_paid"}).
		Find(&payments)
	return payments
}

func (r *PaymentRepository) GetAllUnfinishedPayments() []model.Payment {
	var payments []model.Payment
	r.DB.
		Preload("Account").
		Preload("CurrentPaymentState").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"status_name\" IN ?", []string{enum.Paid.String(), enum.Forwarded.String()}).
		Find(&payments)
	return payments
}
