package utils

import (
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
)

func TestGetETHFromWEI(t *testing.T) {
	shouldEthAmount := decimal.New(1, -1)
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

func TestGetWEIFromETHRoundUp(t *testing.T) {
	shouldWEIAmount := big.NewInt(1)
	ethAmount := 0.00000000000000000000000001
	weiAmount := GetWEIFromETH(&ethAmount)
	if weiAmount.Cmp(shouldWEIAmount) != 0 {
		t.Fatalf(`The calculated ethAmount %v, should be: %v`, ethAmount, shouldWEIAmount)
	}
}
