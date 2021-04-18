package gohan

import (
	"github.com/kurumiimari/gohan/chain"
)

type config struct {
	Network *chain.Network
	Prefix  string
}

var Config = new(config)
