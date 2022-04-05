package database

import (
	"ethereum-service/internal/config"
	"ethereum-service/internal/repository"
	"ethereum-service/model"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB *gorm.DB
)

func DbInit() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.Opts.DBOpts.DbHost,
		config.Opts.DBOpts.DbUser,
		config.Opts.DBOpts.DbPassword,
		config.Opts.DBOpts.DbName,
		config.Opts.DBOpts.DbPort)
	connection, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		panic("could not connect to the database")
	}

	DB = connection
	err = connection.AutoMigrate(&model.Payment{})
	err = connection.AutoMigrate(&model.PaymentState{})
	err = connection.AutoMigrate(&model.Account{})

	repository.InitPayment(DB)
	repository.InitAccount(DB)

	if err != nil {
		return
	}
}
