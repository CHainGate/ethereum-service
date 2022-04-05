package repository

import (
	"ethereum-service/internal/testutils"
	"github.com/CHainGate/backend/pkg/enum"
	"github.com/DATA-DOG/go-sqlmock"
	"math/big"
	"os"
	"testing"
)

func shutdown() {
}

func TestMain(m *testing.M) {
	testutils.Setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestCreatePayment(t *testing.T) {
	mock, repo := NewPaymentMock()
	mock = testutils.SetupCreatePayment(mock)
	repo.CreatePayment(&testutils.EmptyPayment, big.NewInt(1000000))
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetAllPaymentIntents(t *testing.T) {
	mock, repo := NewPaymentMock()
	mock = testutils.SetupAllPaymentIntents(mock)
	repo.GetAllPaymentIntents()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestUpdatePaymentState(t *testing.T) {
	mock, repo := NewPaymentMock()
	mock = testutils.SetupUpdatePaymentState(mock)
	repo.UpdatePaymentState(testutils.WaitingPayment, enum.StatePartiallyPaid.String(), big.NewInt(10))
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func NewPaymentMock() (sqlmock.Sqlmock, *PaymentRepository) {
	mock, gormDb := testutils.NewMock()
	return mock, &PaymentRepository{DB: gormDb}
}