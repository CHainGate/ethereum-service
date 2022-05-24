package repository

import (
	"ethereum-service/model"
	"github.com/CHainGate/backend/pkg/enum"
	"github.com/ethereum/go-ethereum/common"
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
		log.Printf("Error by creating new Payment %v", result.Error)
		return nil, result.Error
	}
	return payment, nil
}

func (r *PaymentRepository) GetAllPaymentIntents() []model.Payment {
	var payments []model.Payment
	r.DB.
		Preload("Account").
		Preload("CurrentPaymentState").
		Preload("PaymentStates").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"status_name\" IN ?", []string{"waiting", "partially_paid"}).
		Find(&payments)
	return payments
}

func (r *PaymentRepository) GetModePaymentIntents(mode enum.Mode) []model.Payment {
	var payments []model.Payment
	r.DB.
		Preload("Account").
		Where("mode = ?", mode.String()).
		Preload("CurrentPaymentState").
		Preload("PaymentStates").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"status_name\" IN ?", []string{"waiting", "partially_paid"}).
		Find(&payments)
	return payments
}

func (r *PaymentRepository) GetPaymentIntentByAddress(address *common.Address) (*gorm.DB, model.Payment) {
	var payment model.Payment
	result := r.DB.Debug().
		Preload("Account").
		Preload("CurrentPaymentState").
		Preload("PaymentStates").
		Joins("Account").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"status_name\" IN ? AND \"Account\".\"address\" = ?", []string{"waiting", "partially_paid"}, address.Hex()).
		First(&payment)
	return result, payment
}

func (r *PaymentRepository) GetAllUnfinishedPayments() []model.Payment {
	var payments []model.Payment
	r.DB.
		Preload("Account").
		Preload("CurrentPaymentState").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"status_name\" IN ?", []string{enum.Confirmed.String(), enum.Forwarded.String()}).
		Find(&payments)
	return payments
}

func (r *PaymentRepository) GetAllConfirmingPayments(mode enum.Mode) []model.Payment {
	var payments []model.Payment
	r.DB.Debug().
		Where("mode = ?", mode.String()).
		Preload("Account").
		Preload("CurrentPaymentState").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"status_name\" IN ?", []string{enum.Paid.String()}).
		Find(&payments)
	return payments
}
