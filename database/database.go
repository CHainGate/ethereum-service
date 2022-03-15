package database

import (
	"ethereum-service/internal"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB *gorm.DB
)

func DbInit() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", internal.Opts.DbHost, internal.Opts.DbUser, internal.Opts.DbPassword, internal.Opts.DbName, internal.Opts.DbPort)
	connection, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("could not connect to the database")
	}

	DB = connection
	err = connection.AutoMigrate(&internal.PaymentStatus{})
	err = connection.AutoMigrate(&internal.Payment{})
	err = connection.AutoMigrate(&internal.Account{})

	if err != nil {
		return
	}
}
