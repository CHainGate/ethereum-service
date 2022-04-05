package repository

import (
	"ethereum-service/internal/testutils"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
)

func TestCreateAccount(t *testing.T) {
	mock, repo := NewAccountMock()
	mock = testutils.SetupCreateAccount(mock)
	repo.CreateAccount(testutils.ChaingateAcc)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestUpdateAccount(t *testing.T) {
	mock, repo := NewAccountMock()
	mock = testutils.SetupUpdateAccount(mock)
	repo.UpdateAccount(testutils.ChaingateAcc)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetFreeAccount(t *testing.T) {
	mock, repo := NewAccountMock()
	mock = testutils.SetupGetFreeAccount(mock)
	repo.GetFreeAccount()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func NewAccountMock() (sqlmock.Sqlmock, *AccountRepository) {
	mock, gormDb := testutils.NewMock()
	return mock, &AccountRepository{DB: gormDb}
}
