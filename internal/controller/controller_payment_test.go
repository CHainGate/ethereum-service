package controller

import (
	"ethereum-service/internal/repository"
	"ethereum-service/internal/testutils"
	"ethereum-service/model"
	"ethereum-service/utils"
	"github.com/CHainGate/backend/pkg/enum"
	"gopkg.in/h2non/gock.v1"
	"testing"
)

func TestCreatePayment(t *testing.T) {
	expectedPayAmountFloat := 0.0001
	expectedPayAmountBigInt := utils.GetWEIFromETH(&expectedPayAmountFloat)
	mock, gormDb := testutils.NewMock()
	defer gock.Off() // Flush pending mocks after test execution
	gock.New("http://localhost:8001").
		Get("/api/price-conversion").
		Reply(200).
		JSON(map[string]float64{"Price": expectedPayAmountFloat})
	repository.InitAccount(gormDb)
	repository.InitPayment(gormDb)
	mock = testutils.SetupGetFreeAccount(mock)
	mock = testutils.SetupUpdateAccount(mock)
	mock = testutils.SetupCreatePaymentWithoutIdCheck(mock)
	p, _, _ := CreatePayment("Test", 100.0, "USD", model.CreateAccount().Address)
	if p.CurrentPaymentState.StatusName != enum.StateWaiting.String() {
		t.Fatalf("Payment is in the wrong state. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.StatusName, enum.StateWaiting.String())
	}
	if p.CurrentPaymentState.PayAmount.Int.Cmp(expectedPayAmountBigInt) != 0 {
		t.Fatalf("Payment has the wrong amount. Payment is \"%v\", but should be \"%v\"", p.CurrentPaymentState.PayAmount.Int.String(), expectedPayAmountBigInt.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
