package chain

type Coin struct {
	Version    uint8
	Height     int
	Value      uint64
	Address    *Address
	Covenant   *Covenant
	Prevout    *Outpoint
	Coinbase   bool
	Derivation Derivation
}

func (c *Coin) Equal(other *Coin) bool {
	return c.Version == other.Version &&
		c.Height == other.Height &&
		c.Value == other.Value &&
		c.Address.Equal(other.Address) &&
		c.Covenant.Equal(other.Covenant) &&
		c.Coinbase == other.Coinbase
}
