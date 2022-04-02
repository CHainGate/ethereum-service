package dbaccess

import (
	"ethereum-service/database"
	"ethereum-service/model"
	"gorm.io/gorm"
	"log"
	"math/big"
)

func GetFreeAccount() (*gorm.DB, *model.Account) {
	acc := model.Account{}
	result := database.DB.Where("used = ?", "false").First(&acc)
	return result, &acc
}

func CreateAccount(acc *model.Account) *model.Account {
	createAccountResult := database.DB.Create(&acc)
	if createAccountResult.Error != nil {
		log.Fatal(createAccountResult.Error)
	}
	return acc
}

func UpdateAccount(acc *model.Account) error {
	result := database.DB.Save(&acc)
	if result.Error != nil {
		log.Fatal(result.Error)
		return result.Error
	}
	return nil
}

func UpdatePaymentState(payment model.Payment, state string, balance *big.Int) {
	payment.UpdatePaymentState(state, balance)
	database.DB.Save(&payment)
}

func CreatePayment(payment *model.Payment, finalPaymentAmount *big.Int) (*model.Payment, error) {
	payment.AddNewPaymentState("waiting", big.NewInt(0), finalPaymentAmount)
	result := database.DB.Create(&payment)
	if result.Error != nil {
		log.Fatal(result.Error)
		return nil, result.Error
	}
	return payment, nil
}

func GetAllPaymentIntents() []model.Payment {
	var payments []model.Payment
	database.DB.
		Preload("Account").
		Preload("CurrentPaymentState").
		Joins("CurrentPaymentState").
		Where("\"CurrentPaymentState\".\"status_name\" IN ?", []string{"waiting", "partially_paid"}).
		Find(&payments)
	return payments
}
