package utils

import (
	"ethereum-service/internal/config"
	"math/big"
	"testing"
)

func TestGetChaingateEarnings(t *testing.T) {
	config.ReadOpts()
	final := GetChaingateEarnings(big.NewInt(100))
	shouldBe := big.NewInt(1)
	if final.Cmp(shouldBe) != 0 {
		t.Fatalf(`The calculated ethAmount %v, should be: %v`, final, shouldBe)
	}
}
