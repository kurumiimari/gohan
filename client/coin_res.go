package client

type CoinRes struct {
	Version  int    `json:"version"`
	Height   int    `json:"height"`
	Value    uint64 `json:"value"`
	Address  string `json:"address"`
	Covenant struct {
		Type   int      `json:"type"`
		Action string   `json:"action"`
		Items  []string `json:"items"`
	} `json:"covenant"`
	Coinbase bool   `json:"coinbase"`
	Hash     string `json:"hash"`
	Index    int    `json:"index"`
}
