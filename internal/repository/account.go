package repository

import (
	"ethereum-service/model"
	"log"

	"gorm.io/gorm"
)

type Repository struct {
	DB *gorm.DB
}

type IAccountRepository interface {
	GetFreeAccount() (*gorm.DB, *model.Account)
	CreateAccount(acc *model.Account) *model.Account
	UpdateAccount(acc *model.Account) error
}

func InitAccount(db *gorm.DB) {
	Payment = &Repository{DB: db}
}

var (
	Account IAccountRepository
)

func (r *Repository) GetFreeAccount() (*gorm.DB, *model.Account) {
	acc := model.Account{}
	result := r.DB.Where("used = ?", "false").First(&acc)
	return result, &acc
}

func (r *Repository) CreateAccount(acc *model.Account) *model.Account {
	createAccountResult := r.DB.Create(&acc)
	if createAccountResult.Error != nil {
		log.Fatal(createAccountResult.Error)
	}
	return acc
}

func (r *Repository) UpdateAccount(acc *model.Account) error {
	result := r.DB.Save(&acc)
	if result.Error != nil {
		log.Fatal(result.Error)
		return result.Error
	}
	return nil
}
