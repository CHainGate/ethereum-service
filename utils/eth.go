package utils

import (
	"crypto/ecdsa"
	"errors"
	"ethereum-service/internal/config"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

var BlockFailed = errors.New("tx failed")

func GetETHFromWEI(amount *big.Int) *big.Float {
	f := new(big.Float)
	f.SetPrec(236) //  IEEE 754 octuple-precision binary floating-point format: binary256
	f.SetMode(big.ToNearestEven)
	fWei := new(big.Float)
	fWei.SetPrec(236) //  IEEE 754 octuple-precision binary floating-point format: binary256
	fWei.SetMode(big.ToNearestEven)
	return f.Quo(fWei.SetInt(amount), big.NewFloat(params.Ether))
}

func GetWEIFromETH(val *float64) *big.Int {
	bigval := new(big.Float)
	bigval.SetFloat64(*val)
	balance := big.NewFloat(0).Mul(bigval, big.NewFloat(1000000000000000000))
	final, accur := balance.Int(nil)
	if accur == big.Below {
		final.Add(final, big.NewInt(1))
	}
	return final
}

func GetPrivateKey(key string) (*ecdsa.PrivateKey, error) {
	decryptedKey, err := Decrypt([]byte(config.Opts.PrivateKeySecret), key)
	if err != nil {
		log.Println("Unable to decrypt Private Key!", err)
	}
	if strings.HasPrefix(decryptedKey, "0x") {
		decryptedKey = decryptedKey[2:]
	}
	return crypto.HexToECDSA(decryptedKey)
}
