package internal

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"log"
	"math/big"
	"os"
	"time"
)

type OptsType struct {
	AccountPrivateKey string
	DbHost            string
	DbUser            string
	DbPassword        string
	DbName            string
	DbPort            string
}

var (
	PaymentIntents []Payment
	accounts       []Account
	Opts           *OptsType
)

type PaymentIntent struct {
	mode    int
	account Account
	amount  *big.Int
}

type Account struct {
	Id         uuid.UUID         `gorm:"type:uuid;default:uuid_generate_v4()"`
	PublicKey  *ecdsa.PublicKey  `gorm:"type:varchar"`
	PrivateKey *ecdsa.PrivateKey `gorm:"type:varchar"`
	Address    common.Address    `gorm:"type:varchar"`
	Used       bool
	PaymentId  uuid.UUID
}

type Payment struct {
	Id         uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	Account    Account
	UserWallet string
	Mode       string
	//TODO: change to float64
	PriceAmount   big.Int `gorm:"type:text"`
	PriceCurrency string
	CreatedAt     time.Time
	updatedAt     time.Time
	PaymentStates []PaymentStatus
}

type PaymentStatus struct {
	Id             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4()"`
	PaymentId      uuid.UUID
	PayAmount      big.Int `gorm:"type:bigint"`
	AmountReceived big.Int `gorm:"type:bigint"`
	StatusName     string
	CreatedAt      time.Time
}

func GetAccount() Account {
	acc, err := getFreeAccount()
	if err != nil {
		acc = createAccount()
	}
	return acc
}

func createAccount() Account {
	account := Account{}
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	account.PrivateKey = privateKey
	privateKeyBytes := crypto.FromECDSA(account.PrivateKey)
	fmt.Println("SAVE BUT DO NOT SHARE THIS (Private Key):", hexutil.Encode(privateKeyBytes))

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	account.PublicKey = publicKeyECDSA
	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	fmt.Println("Public Key:", hexutil.Encode(publicKeyBytes))

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	fmt.Println("Address:", address)
	account.Address = common.HexToAddress(address)
	accounts = append(accounts, account)
	return account
}

func getFreeAccount() (Account, error) {
	if Opts.AccountPrivateKey != "" {
		pk, err := crypto.HexToECDSA(Opts.AccountPrivateKey)
		if err != nil {
			log.Fatal(err)
		}

		acc := Account{
			PrivateKey: pk,
			PublicKey:  &pk.PublicKey,
			Address:    common.HexToAddress(crypto.PubkeyToAddress(pk.PublicKey).Hex()),
		}

		return acc, nil
	}
	return Account{}, fmt.Errorf("unable to get free address")
}

func CreateInternalOps() {
	o := &OptsType{}
	flag.StringVar(&o.AccountPrivateKey, "ACCOUNT_PRIVATE_KEY", lookupEnv("ACCOUNT_PRIVATE_KEY"), "Account private Key")
	flag.StringVar(&o.DbHost, "DB_HOST", lookupEnv("DB_HOST"), "Database Host")
	flag.StringVar(&o.DbUser, "DB_USER", lookupEnv("DB_USER"), "Database User")
	flag.StringVar(&o.DbPassword, "DB_PASSWORD", lookupEnv("DB_PASSWORD"), "Database Password")
	flag.StringVar(&o.DbName, "DB_NAME", lookupEnv("DB_NAME"), "Database Name")
	flag.StringVar(&o.DbPort, "DB_PORT", lookupEnv("DB_PORT"), "Database Port")
	Opts = o
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
