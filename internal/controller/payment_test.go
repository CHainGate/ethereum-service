package controller

import (
	"ethereum-service/internal"
	"ethereum-service/internal/repository"
	"ethereum-service/model"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"testing"
)

var acc = internal.CreateAccount()

var payment = model.Payment{
	Account:       acc,
	Mode:          "Test",
	Base:          model.Base{ID: uuid.New()},
	AccountID:     acc.ID,
	PriceAmount:   100,
	PriceCurrency: "USD",
	UserWallet:    "0xec5Df5cc18616FaEcB103691F4377972abf0B613",
}

func NewMock() (sqlmock.Sqlmock, repository.IPaymentRepository) {
	db, mock, err := sqlmock.New()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	dialector := postgres.New(postgres.Config{
		Conn:       db,
		DriverName: "postgres",
	})

	gormDb, err := gorm.Open(dialector, &gorm.Config{})
	return mock, &repository.Repository{DB: gormDb}
}

func getPayment() repository.IPaymentRepository {
	mock, repo := NewMock()
	rows := sqlmock.NewRows([]string{"id", "user_wallet", "mode", "price_amount", "price_currency", "created_at"}).
		AddRow("0eeaac28-832d-4bc1-85f9-57faeaf96682", "0xcDd9C81f1855Bfd6a309A395b53f273d539ad7aa", "Test", "100", "USD", "2022-03-18 15:56:52.534444 +00:00")
	mock.ExpectQuery("CREATE").
		WithArgs(1).
		WillReturnRows(rows)
	return repo
}

func TestCreatePayment(t *testing.T) {
	// TODO: Test
}
