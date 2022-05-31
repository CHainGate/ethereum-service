package config

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type OptsType struct {
	Main                       string
	Test                       string
	ChaingateEarningsPercent   string
	TargetWallet               string
	FeeFactor                  string
	IncomingBlockConfirmations int64
	OutgoingTxConfirmations    int64
	PrivateKeySecret           string
	ProxyBaseUrl               string
	BackendBaseUrl             string
	DBOpts                     DBOpts
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

func lookupInt64Env(key string, defaultValue int64) int64 {
	s := lookupEnv(key)
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return defaultValue
	}
	return v
}

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
	if Opts == nil {
		err := godotenv.Load()
		if err != nil {
			log.Printf("Could not find env file [%v], using defaults", err)
		}

		o := &OptsType{}
		flag.StringVar(&o.Main, "MAIN", lookupEnv("MAIN", "https://mainnet.infura.io/v3/"), "Mainnet")
		flag.StringVar(&o.Test, "TEST", lookupEnv("TEST", "https://rinkeby.infura.io/v3/"), "Testnet")
		flag.StringVar(&o.ChaingateEarningsPercent, "CHAINGATE_EARNINGS", lookupEnv("CHAINGATE_EARNINGS", "1"), "Percent how much percent CHainGate takes from the transaction")
		flag.StringVar(&o.TargetWallet, "TARGET_WALLET", lookupEnv("TARGET_WALLET", "0xb794f5ea0ba39494ce839613fffba74279579268"), "Target wallet address to send the earned eth's")
		flag.StringVar(&o.FeeFactor, "FEE_FACTOR", lookupEnv("FEE_FACTOR", "100"), "How many times the earnings should be higher than the fees to forward the earnings")
		flag.Int64Var(&o.IncomingBlockConfirmations, "INCOMING_BLOCK_CONFIRMATIONS", lookupInt64Env("INCOMING_BLOCK_CONFIRMATIONS", 12), "How many confirmations should be waited until the block will be counted as confirmed")
		flag.Int64Var(&o.OutgoingTxConfirmations, "OUTGOING_TX_CONFIRMATIONS", lookupInt64Env("OUTGOING_TX_CONFIRMATIONS", 3), "How many confirmations should be waited until the tx of the payment will be counted as finished")
		flag.StringVar(&o.PrivateKeySecret, "PRIVATE_KEY_SECRET", lookupEnv("PRIVATE_KEY_SECRET", "secret16byte1234"), "Secret for decrypting private keys")
		flag.StringVar(&o.DBOpts.DbHost, "DB_HOST", lookupEnv("DB_HOST"), "Database Host")
		flag.StringVar(&o.DBOpts.DbUser, "DB_USER", lookupEnv("DB_USER"), "Database User")
		flag.StringVar(&o.DBOpts.DbPassword, "DB_PASSWORD", lookupEnv("DB_PASSWORD"), "Database Password")
		flag.StringVar(&o.DBOpts.DbName, "DB_NAME", lookupEnv("DB_NAME"), "Database Name")
		flag.StringVar(&o.DBOpts.DbPort, "DB_PORT", lookupEnv("DB_PORT"), "Database Port")
		flag.StringVar(&o.ProxyBaseUrl, "PROXY_BASE_URL", lookupEnv("PROXY_BASE_URL", "http://localhost:8001/api"), "Proxy base url")
		flag.StringVar(&o.BackendBaseUrl, "BACKEND_BASE_URL", lookupEnv("BACKEND_BASE_URL", "http://localhost:8000/api/internal"), "Backend base url")
		Opts = o
	}
}
