package utils

import (
	"math/big"
	"testing"
)

func TestGetETHFromWEI(t *testing.T) {
	f := new(big.Float)
	f.SetPrec(236) //  IEEE 754 octuple-precision binary floating-point format: binary256
	f.SetMode(big.ToNearestEven)
	shouldEthAmount := f.Quo(big.NewFloat(1), big.NewFloat(10))
	ethAmount := GetETHFromWEI(big.NewInt(100000000000000000))
	if ethAmount.Text('f', 18) != shouldEthAmount.Text('f', 18) {
		t.Fatalf(`The calculated ethAmount %s, should be: %s`, ethAmount.Text('f', 18), shouldEthAmount.Text('f', 18))
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
