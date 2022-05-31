package service

import (
	"ethereum-service/internal/config"
	"ethereum-service/internal/testutils"
	"math/big"
	"testing"

	"github.com/CHainGate/backend/pkg/enum"
	"github.com/google/uuid"
	"gopkg.in/h2non/gock.v1"
)

func TestSendState(t *testing.T) {
	config.ReadOpts()
	defer gock.Off() // Flush pending mocks after test execution
	gock.New("http://localhost:8000").
		Put("/api/internal/payment/webhook").
		MatchType("json").
		//JSON(map[string]string{"foo": "bar"}).
		Reply(200)

	accountID := uuid.New()
	paymentID := uuid.New()
	paymentState := testutils.CreatePaymentState(accountID, paymentID, enum.PartiallyPaid, big.NewInt(10))
	SendState(paymentID, paymentState, "")
	if gock.IsDone() != true {
		t.Fatalf("Request should have been sent, but there are open requests")
	}
}
