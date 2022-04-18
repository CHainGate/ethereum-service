package service

import (
	"context"
	"ethereum-service/backendClientApi"
	"ethereum-service/internal/config"
	"ethereum-service/model"
	"ethereum-service/utils"
	"fmt"
	"math/big"
	"os"

	"github.com/google/uuid"
)

func SendState(paymentId uuid.UUID, state model.PaymentState) {
	payAmount, payAmountAccuracy := utils.GetETHFromWEI(&state.PayAmount.Int).Float64()
	amountReceived, amountReceivedAccuracy := utils.GetETHFromWEI(&state.AmountReceived.Int).Float64()
	if payAmountAccuracy == big.Exact && amountReceivedAccuracy == big.Exact {
		paymentUpdateDto := *backendClientApi.NewPaymentUpdateDto(paymentId.String(), payAmount, "ETH", *backendClientApi.NewNullableFloat64(&amountReceived), state.StatusName) // PaymentUpdateDto |  (optional)

		configuration := backendClientApi.NewConfiguration()
		configuration.Servers[0].URL = config.Opts.BackendBaseUrl
		apiClient := backendClientApi.NewAPIClient(configuration)
		resp, err := apiClient.PaymentUpdateApi.UpdatePayment(context.Background()).PaymentUpdateDto(paymentUpdateDto).Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `PaymentUpdateApi.UpdatePayment``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
		} else {
			fmt.Printf("Update sent: Payment %s updated to: %s\n", paymentId.String(), state.StatusName)
		}
	} else {
		// TODO handle too big or too small numbers
	}
}
