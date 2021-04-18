package chain

import (
	"encoding/json"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

type Derivation []uint32

func (d Derivation) MarshalJSON() ([]byte, error) {
	if len(d) == 0 {
		return json.Marshal(nil)
	}

	return json.Marshal(d.String())
}

func (d *Derivation) UnmarshalJSON(b []byte) error {
	var data string
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	if data == "" {
		return nil
	}

	newDeriv, err := ParseDerivation(data)
	if err != nil {
		return err
	}
	*d = newDeriv
	return nil
}

func (d Derivation) String() string {
	nodes := make([]string, len(d)+1)
	nodes[0] = "m"
	for i, der := range d {
		if IsHardenedNode(der) {
			nodes[i+1] = strconv.FormatUint(uint64(der-hdkeychain.HardenedKeyStart), 10)
			nodes[i+1] += "'"
		} else {
			nodes[i+1] = strconv.FormatUint(uint64(der), 10)
		}
	}
	return strings.Join(nodes, "/")
}

func ParseDerivation(in string) (Derivation, error) {
	nodes := strings.Split(in, "/")
	deriv := make(Derivation, len(nodes)-1)

	if nodes[0] != "m" {
		return nil, errors.New("path must start with m/")
	}

	if len(nodes) < 2 {
		return nil, errors.New("path must contain at least one component")
	}

	for i := 1; i < len(nodes); i++ {
		nodeStr := nodes[i]
		trimmed := strings.TrimSuffix(nodeStr, "'")
		node, err := strconv.ParseUint(trimmed, 10, 32)
		if err != nil {
			return nil, errors.Wrap(err, "invalid path node")
		}

		if strings.HasSuffix(nodeStr, "'") {
			deriv[i-1] = HardenNode(uint32(node))
		} else {
			deriv[i-1] = uint32(node)
		}
	}

	return deriv, nil
}

func IsHardenedNode(i uint32) bool {
	return i >= hdkeychain.HardenedKeyStart
}

func HardenNode(i uint32) uint32 {
	return i + hdkeychain.HardenedKeyStart
}

func DeriveExtendedKey(m ExtendedKey, children ...uint32) ExtendedKey {
	for _, child := range children {
		m = m.Child(child)
	}
	return m
}
