package repository

import (
	"ethereum-service/internal/config"
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
	repo.Create(&ep, big.NewInt(100000000000000))
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetAllPayments(t *testing.T) {
	config.ReadOpts()
	mock, repo := NewPaymentMock()
	mock = testutils.SetupAllPayments(mock)
	repo.GetAllOpen()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetOpenByMode(t *testing.T) {
	config.ReadOpts()
	mock, repo := NewPaymentMock()
	mock = testutils.SetupAllPayments(mock, enum.Main)
	repo.GetOpenByMode(enum.Main)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetConfirming(t *testing.T) {
	mock, repo := NewPaymentMock()
	mock = testutils.SetupModePayments(mock, enum.Main, enum.Paid)
	repo.GetConfirming(enum.Main)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetFinishing(t *testing.T) {
	mock, repo := NewPaymentMock()
	mock = testutils.SetupModePayments(mock, enum.Main, enum.Forwarded)
	repo.GetFinishing(enum.Main)
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
