package utils

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/params"
	"github.com/shopspring/decimal"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

func GetETHFromWEI(amount *big.Int) decimal.Decimal {
	return decimal.NewFromBigInt(amount, 0).Div(decimal.NewFromFloat(params.Ether))
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
	if strings.HasPrefix(key, "0x") {
		key = key[2:]
	}
	return crypto.HexToECDSA(key)
}
