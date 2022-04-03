package repository

import (
	"ethereum-service/model"
	"log"
	"math/big"

	"gorm.io/gorm"
)

type IPaymentRepository interface {
	UpdatePaymentState(payment model.Payment, state string, balance *big.Int)
	CreatePayment(payment *model.Payment, finalPaymentAmount *big.Int) (*model.Payment, error)
	GetAllPaymentIntents() []model.Payment
}

func InitPayment(db *gorm.DB) {
	Payment = &Repository{DB: db}
}

var (
	Payment IPaymentRepository
)

func (r *Repository) UpdatePaymentState(payment model.Payment, state string, balance *big.Int) {
	payment.UpdatePaymentState(state, balance)
	r.DB.Save(&payment)
}

func (r *Repository) CreatePayment(payment *model.Payment, finalPaymentAmount *big.Int) (*model.Payment, error) {
	payment.AddNewPaymentState("waiting", big.NewInt(0), finalPaymentAmount)
	result := r.DB.Create(&payment)
	if result.Error != nil {
		log.Fatal(result.Error)
		return nil, result.Error
	}
	return payment, nil
}

func (r *Repository) GetAllPaymentIntents() []model.Payment {
	var payments []model.Payment
	r.DB.
		Preload("Account").
		Preload("CurrentPaymentState").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"status_name\" IN ?", []string{"waiting", "partially_paid"}).
		Find(&payments)
	return payments
}
