package internal

import (
	"context"
	"ethereum-service/backendClientApi"
	"ethereum-service/model"
	"fmt"
	"math/big"
	"os"

	"github.com/google/uuid"
)

func SendState(paymentId uuid.UUID, state model.PaymentState) {
	payAmount, payAmountAccuracy := new(big.Float).SetInt(&state.PayAmount.Int).Float64()
	amountReceived, amountReceivedAccuracy := new(big.Float).SetInt(&state.AmountReceived.Int).Float64()
	if payAmountAccuracy == big.Exact && amountReceivedAccuracy == big.Exact {
		paymentUpdateDto := *backendClientApi.NewPaymentUpdateDto(paymentId.String(), payAmount, "ETH", *backendClientApi.NewNullableFloat64(&amountReceived), state.StatusName) // PaymentUpdateDto |  (optional)

		configuration := backendClientApi.NewConfiguration()
		apiClient := backendClientApi.NewAPIClient(configuration)
		resp, err := apiClient.PaymentUpdateApi.UpdatePayment(context.Background()).PaymentUpdateDto(paymentUpdateDto).Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error when calling `PaymentUpdateApi.UpdatePayment``: %v\n", err)
			fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", resp)
		} else {
			fmt.Printf("Updated sended: Payment %s with updated to: %s", paymentId.String(), state.StatusName)
		}
	} else {
		// TODO handle too big or too small numbers
	}
}
