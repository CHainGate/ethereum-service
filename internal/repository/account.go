package repository

import (
	"ethereum-service/model"
	"log"

	"github.com/CHainGate/backend/pkg/enum"

	"gorm.io/gorm"
)

type AccountRepository struct {
	DB *gorm.DB
}

func InitAccount(db *gorm.DB) {
	Account = &AccountRepository{DB: db}
}

var (
	Account model.IAccountRepository
)

func (r *AccountRepository) GetFreeAccount(mode enum.Mode) (*gorm.DB, *model.Account) {
	acc := model.Account{}
	result := r.DB.Where("used = ? AND mode = ?", "false", mode.String()).First(&acc)
	return result, &acc
}

func (r *AccountRepository) CreateAccount(acc *model.Account) *model.Account {
	createAccountResult := r.DB.Create(&acc)
	if createAccountResult.Error != nil {
		log.Fatal(createAccountResult.Error)
	}
	return acc
}

func (r *AccountRepository) UpdateAccount(acc *model.Account) error {
	result := r.DB.Save(&acc)
	if result.Error != nil {
		log.Fatal(result.Error)
		return result.Error
	}
	return nil
}
