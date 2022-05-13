/*
 * OpenAPI blockchain services
 *
 * This is the OpenAPI definition of the blockchain services.
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package services

import (
	"context"
	"ethereum-service/internal/controller"
	"ethereum-service/openApi"
	"fmt"
	"github.com/CHainGate/backend/pkg/enum"
	"net/http"
)

// PaymentApiService is a service that implements the logic for the PaymentApiServicer
// This service should implement the business logic for every endpoint for the PaymentApi API.
// Include any external packages or services that will be required by this service.
type PaymentApiService struct {
}

// NewPaymentApiService creates a default api service
func NewPaymentApiService() openApi.PaymentApiServicer {
	return &PaymentApiService{}
}

// CreatePayment - create new payment
func (s *PaymentApiService) CreatePayment(ctx context.Context, paymentRequest openApi.PaymentRequest) (openApi.ImplResponse, error) {
	mode, ok := enum.ParseStringToModeEnum(paymentRequest.Mode)
	if !ok {
		return openApi.Response(http.StatusInternalServerError, nil), fmt.Errorf("unable to parse mode")
	}
	payment, finalPayAmount, err := controller.CreatePayment(mode, paymentRequest.PriceAmount, paymentRequest.PriceCurrency, paymentRequest.Wallet)
	if err != nil {
		return openApi.Response(http.StatusInternalServerError, nil), fmt.Errorf("unable to get free address")
	}

	paymentResponse := openApi.PaymentResponse{
		PaymentId:     payment.ID.String(),
		PriceAmount:   payment.PriceAmount,
		PriceCurrency: payment.PriceCurrency,
		PayAddress:    payment.Account.Address,
		PayAmount:     finalPayAmount.String(),
		PayCurrency:   "ETH",
		PaymentState:  payment.CurrentPaymentState.StatusName,
	}
	return openApi.Response(http.StatusCreated, paymentResponse), nil
}
