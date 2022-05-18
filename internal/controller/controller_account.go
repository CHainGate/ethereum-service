package controller

import (
	"errors"
	"ethereum-service/internal/repository"
	"ethereum-service/model"

	"github.com/CHainGate/backend/pkg/enum"

	"gorm.io/gorm"
)

func GetAccount(mode enum.Mode) (model.Account, error) {
	return getFreeAccount(mode)
}

func getFreeAccount(mode enum.Mode) (model.Account, error) {
	result, acc := repository.Account.GetFreeAccount(mode)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			acc = model.CreateAccount(mode)
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
