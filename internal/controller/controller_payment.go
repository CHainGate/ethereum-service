package controller

import (
	"ethereum-service/internal"
	"ethereum-service/internal/repository"
	"ethereum-service/model"
	"ethereum-service/utils"
	"fmt"
	"github.com/google/uuid"
)

func CreatePayment(mode string, priceAmount float64, priceCurrency string, wallet string) (*model.Payment, *float64, error) {
	acc, err := GetAccount()

	if err != nil {
		return nil, nil, fmt.Errorf("unable to get free address")
	}

	payment := model.Payment{
		Mode:          mode,
		AccountID:     acc.ID,
		Account:       &acc,
		PriceAmount:   priceAmount,
		PriceCurrency: priceCurrency,
		UserWallet:    wallet,
	}

	payment.ID = uuid.New()

	val := internal.GetETHAmount(payment)
	final := utils.GetWEIFromETH(val)
	_, err = repository.Payment.CreatePayment(&payment, final)

	return &payment, val, nil
}
