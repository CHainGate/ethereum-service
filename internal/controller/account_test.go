package controller

import (
	"ethereum-service/internal/repository"
	"ethereum-service/internal/testutils"
	"testing"
)

func TestGetAccount(t *testing.T) {
	mock, gormDb := testutils.NewMock()
	repository.InitAccount(gormDb)
	mock = testutils.SetupGetFreeAccount(mock)
	mock = testutils.SetupUpdateAccount(mock)
	GetAccount()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
