package repository

import (
	"ethereum-service/internal/testutils"
	"math/big"
	"os"
	"testing"

	"github.com/CHainGate/backend/pkg/enum"
	"github.com/DATA-DOG/go-sqlmock"
)

func shutdown() {
}

func TestMain(m *testing.M) {
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestCreatePayment(t *testing.T) {
	mock, repo := NewPaymentMock()
	mock = testutils.SetupCreatePayment(mock)
	ep := testutils.GetEmptyPayment()
	repo.CreatePayment(&ep, big.NewInt(100000000000000))
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
	wp := testutils.GetWaitingPayment()
	wp.UpdatePaymentState(enum.PartiallyPaid, big.NewInt(10))
	repo.UpdatePaymentState(&wp)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func NewPaymentMock() (sqlmock.Sqlmock, *PaymentRepository) {
	mock, gormDb := testutils.NewMock()
	return mock, &PaymentRepository{DB: gormDb}
}
