package utils

import (
	"math/big"
	"testing"
)

func TestGetETHFromWEI(t *testing.T) {
	shouldEthAmount := big.NewFloat(0.1)
	ethAmount := GetETHFromWEI(big.NewInt(100000000000000000))
	if ethAmount.Cmp(shouldEthAmount) != 0 {
		t.Fatalf(`The calculated ethAmount %v, should be: %v`, ethAmount, shouldEthAmount)
	}
}

func TestGetWEIFromETH(t *testing.T) {
	shouldWEIAmount := big.NewInt(100000000000000000)
	ethAmount := 0.1
	weiAmount := GetWEIFromETH(&ethAmount)
	if weiAmount.Cmp(shouldWEIAmount) != 0 {
		t.Fatalf(`The calculated ethAmount %v, should be: %v`, ethAmount, shouldWEIAmount)
	}
}