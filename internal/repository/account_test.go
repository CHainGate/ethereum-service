package repository

import (
	"ethereum-service/internal/testutils"
	"testing"

	"github.com/CHainGate/backend/pkg/enum"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateAccount(t *testing.T) {
	mock, repo := NewAccountMock()
	mock = testutils.SetupCreateAccount(mock)
	ca := testutils.GetChaingateAcc()
	repo.Create(&ca)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestUpdateAccount(t *testing.T) {
	mock, repo := NewAccountMock()
	mock = testutils.SetupUpdateAccount(mock)
	ca := testutils.GetChaingateAcc()
	repo.Update(&ca)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetFreeAccount(t *testing.T) {
	mock, repo := NewAccountMock()
	mock = testutils.SetupGetFreeAccount(mock)
	repo.GetFree(enum.Main)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func NewAccountMock() (sqlmock.Sqlmock, *AccountRepository) {
	mock, gormDb := testutils.NewMock()
	return mock, &AccountRepository{DB: gormDb}
}
