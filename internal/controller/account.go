package controller

import (
	"errors"
	"ethereum-service/internal/repository"
	"ethereum-service/model"
	"gorm.io/gorm"
)

func GetAccount() (model.Account, error) {
	return getFreeAccount()
}

func getFreeAccount() (model.Account, error) {
	result, acc := repository.Account.GetFreeAccount()
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			acc = model.CreateAccount()
			repository.Account.CreateAccount(acc)
		} else {
			return model.Account{}, result.Error
		}
	} else {
		acc.Used = true
		err := repository.Account.UpdateAccount(acc)
		if err != nil {
			return model.Account{}, result.Error
		}
	}
	return *acc, nil
}
