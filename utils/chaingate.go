package utils

import (
	"ethereum-service/internal/config"
	"fmt"
	"math/big"
)

func GetChaingateEarnings(payAmount *big.Int) *big.Int {
	chainGatePercent := new(big.Int)
	chainGatePercent, ok := chainGatePercent.SetString(config.Opts.ChaingateEarningsPercent, 10)
	if !ok {
		fmt.Println("Unable to calculate earnings. Don't subtract anything as earnings")
		return big.NewInt(0)
	}
	mul := big.NewInt(0).Mul(payAmount, chainGatePercent)
	final := mul.Div(mul, big.NewInt(100))
	return final
}
