package utils

import "math/big"

func GetETHFromWEI(amount *big.Int) *big.Float {
	return big.NewFloat(0).Quo(new(big.Float).SetInt(amount), big.NewFloat(1000000000000000000))
}