package utils

import "math/big"

func GetETHFromWEI(amount *big.Int) *big.Float {
	return big.NewFloat(0).Quo(new(big.Float).SetInt(amount), big.NewFloat(1000000000000000000))
}

func GetETHFromWEIFloat(amount *big.Float) *big.Float {
	return big.NewFloat(0).Quo(amount, big.NewFloat(1000000000000000000))
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
