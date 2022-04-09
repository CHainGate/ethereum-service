package config

import "math/big"

type ChainConfig struct {
	ChainId  *big.Int
	GasPrice *big.Int
}

var (
	Chain *ChainConfig
)
