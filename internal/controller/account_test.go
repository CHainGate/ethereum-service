package controller

import (
	"ethereum-service/internal/config"
	"ethereum-service/internal/repository"
	"ethereum-service/internal/testutils"
	"testing"

	"github.com/CHainGate/backend/pkg/enum"
)

func TestGetAccount(t *testing.T) {
	config.ReadOpts()
	mock, gormDb := testutils.NewMock()
	repository.InitAccount(gormDb)
	mock = testutils.SetupGetFreeAccount(mock)
	mock = testutils.SetupUpdateAccount(mock, 0)
	GetAccount(enum.Main)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
