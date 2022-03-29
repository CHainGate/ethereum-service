package config

import (
	"flag"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type OptsType struct {
	Main   string
	Test   string
	DBOpts DBOpts
}

type DBOpts struct {
	DbHost     string
	DbUser     string
	DbPassword string
	DbName     string
	DbPort     string
}

var (
	Opts *OptsType
)

func lookupEnv(key string, defaultValues ...string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	for _, v := range defaultValues {
		if v != "" {
			return v
		}
	}
	return ""
}

func ReadOpts() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Could not find env file [%v], using defaults", err)
	}

	o := &OptsType{}
	flag.StringVar(&o.Main, "MAIN", lookupEnv("MAIN"), "Mainnet")
	flag.StringVar(&o.Test, "TEST", lookupEnv("TEST"), "Testnet")
	flag.StringVar(&o.DBOpts.DbHost, "DB_HOST", lookupEnv("DB_HOST"), "Database Host")
	flag.StringVar(&o.DBOpts.DbUser, "DB_USER", lookupEnv("DB_USER"), "Database User")
	flag.StringVar(&o.DBOpts.DbPassword, "DB_PASSWORD", lookupEnv("DB_PASSWORD"), "Database Password")
	flag.StringVar(&o.DBOpts.DbName, "DB_NAME", lookupEnv("DB_NAME"), "Database Name")
	flag.StringVar(&o.DBOpts.DbPort, "DB_PORT", lookupEnv("DB_PORT"), "Database Port")
	Opts = o
}
