package service

import (
	"context"
	"ethereum-service/backendClientApi"
	"ethereum-service/internal/config"
	"ethereum-service/model"
	"fmt"
	"os"

	"github.com/google/uuid"
)

func SendState(paymentId uuid.UUID, state model.PaymentState) error {
	paymentUpdateDto := *backendClientApi.NewPaymentUpdateDto(paymentId.String(), state.PayAmount.String(), "ETH", state.AmountReceived.String(), state.StatusName) // PaymentUpdateDto |  (optional)

	configuration := backendClientApi.NewConfiguration()
	configuration.Servers[0].URL = config.Opts.BackendBaseUrl
	apiClient := backendClientApi.NewAPIClient(configuration)
	resp, err := apiClient.PaymentUpdateApi.UpdatePayment(context.Background()).PaymentUpdateDto(paymentUpdateDto).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `PaymentUpdateApi.UpdatePayment``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
		return err
	} else {
		fmt.Printf("Update sent: Payment %s updated to: %s\n", paymentId.String(), state.StatusName)
	}
	return nil
}
