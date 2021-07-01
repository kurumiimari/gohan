package chain

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"github.com/kurumiimari/gohan/bio"
	"github.com/kurumiimari/gohan/gcrypto"
	"github.com/pkg/errors"
	"io"
)

type CovenantType uint8

const (
	CovenantNone CovenantType = iota
	CovenantClaim
	CovenantOpen
	CovenantBid
	CovenantReveal
	CovenantRedeem
	CovenantRegister
	CovenantUpdate
	CovenantRenew
	CovenantTransfer
	CovenantFinalize
	CovenantRevoke
)

func (c CovenantType) String() string {
	switch c {
	case CovenantNone:
		return "NONE"
	case CovenantClaim:
		return "CLAIM"
	case CovenantOpen:
		return "OPEN"
	case CovenantBid:
		return "BID"
	case CovenantReveal:
		return "REVEAL"
	case CovenantRedeem:
		return "REDEEM"
	case CovenantRegister:
		return "REGISTER"
	case CovenantUpdate:
		return "UPDATE"
	case CovenantRenew:
		return "RENEW"
	case CovenantTransfer:
		return "TRANSFER"
	case CovenantFinalize:
		return "FINALIZE"
	case CovenantRevoke:
		return "REVOKE"
	default:
		panic("invalid covenant type")
	}
}

func NewCovenantTypeFromString(s string) CovenantType {
	switch s {
	case "NONE":
		return CovenantNone
	case "CLAIM":
		return CovenantClaim
	case "OPEN":
		return CovenantOpen
	case "BID":
		return CovenantBid
	case "REVEAL":
		return CovenantReveal
	case "REDEEM":
		return CovenantRedeem
	case "REGISTER":
		return CovenantRegister
	case "UPDATE":
		return CovenantUpdate
	case "RENEW":
		return CovenantRenew
	case "TRANSFER":
		return CovenantTransfer
	case "FINALIZE":
		return CovenantFinalize
	case "REVOKE":
		return CovenantRevoke
	default:
		panic("invalid covenant type")
	}
}

type Covenant struct {
	Type  CovenantType
	Items [][]byte
}

var EmptyCovenant = new(Covenant)

func (c *Covenant) Size() int {
	var itemSizes int
	for _, item := range c.Items {
		itemSizes += len(item)
		itemSizes += bio.SizeVarint(len(item))
	}
	return 1 + bio.SizeVarint(len(c.Items)) + itemSizes
}

func (c *Covenant) WriteTo(w io.Writer) (int64, error) {
	g := bio.NewGuardWriter(w)
	bio.WriteByte(w, byte(c.Type))
	bio.WriteVarint(w, uint64(len(c.Items)))
	for _, item := range c.Items {
		bio.WriteVarBytes(w, item)
	}
	return g.N, errors.Wrap(g.Err, "error writing covenant")
}

func (c *Covenant) ReadFrom(r io.Reader) (int64, error) {
	g := bio.NewGuardReader(r)
	cType, _ := bio.ReadByte(g)
	count, _ := bio.ReadVarint(g)

	var items [][]byte
	for i := 0; i < int(count); i++ {
		item, _ := bio.ReadVarBytes(g)
		items = append(items, item)
	}
	if g.Err != nil {
		return g.N, errors.Wrap(g.Err, "error reading covenant")
	}
	c.Type = CovenantType(cType)
	c.Items = items
	return g.N, nil
}

func (c *Covenant) MarshalJSON() ([]byte, error) {
	items := make([]string, 0)
	for _, item := range c.Items {
		items = append(items, hex.EncodeToString(item))
	}
	jsonCov := struct {
		Type   uint8    `json:"type"`
		Action string   `json:"action"`
		Items  []string `json:"items"`
	}{
		Type:   uint8(c.Type),
		Action: c.Type.String(),
		Items:  items,
	}

	return json.Marshal(jsonCov)
}

func (c *Covenant) UnmarshalJSON(bytes []byte) error {
	jsonCov := struct {
		Type   uint8    `json:"type"`
		Action string   `json:"action"`
		Items  []string `json:"items"`
	}{}

	if err := json.Unmarshal(bytes, &jsonCov); err != nil {
		return err
	}

	items, err := bio.DecodeHexArray(jsonCov.Items)
	if err != nil {
		return err
	}

	c.Type = CovenantType(jsonCov.Type)
	c.Items = items
	return nil
}

func (c *Covenant) Equal(other *Covenant) bool {
	if c.Type != other.Type {
		return false
	}

	if len(c.Items) != len(other.Items) {
		return false
	}
	for i := 0; i < len(c.Items); i++ {
		if !bytes.Equal(c.Items[i], other.Items[i]) {
			return false
		}
	}

	return true
}

func NewRegisterCovenant(name string, height int, renewalHash gcrypto.Hash, resource *Resource) *Covenant {
	buf := new(bytes.Buffer)
	if resource == nil {
		buf.WriteByte(0x00)
	} else {
		if _, err := resource.WriteTo(buf); err != nil {
			panic(err)
		}
	}

	return &Covenant{
		Type: CovenantRegister,
		Items: [][]byte{
			HashName(name),
			bio.Uint32LE(uint32(height)),
			buf.Bytes(),
			renewalHash,
		},
	}
}

func NewUpdateCovenant(name string, height int, resource *Resource) *Covenant {
	buf := new(bytes.Buffer)
	if resource == nil {
		buf.WriteByte(0x00)
	} else {
		if _, err := resource.WriteTo(buf); err != nil {
			panic(err)
		}
	}

	return &Covenant{
		Type: CovenantUpdate,
		Items: [][]byte{
			HashName(name),
			bio.Uint32LE(uint32(height)),
			buf.Bytes(),
		},
	}
}

func NewTransferCovenant(name string, height int, transferee *Address) *Covenant {
	return &Covenant{
		Type: CovenantTransfer,
		Items: [][]byte{
			HashName(name),
			bio.Uint32LE(uint32(height)),
			{transferee.Version},
			transferee.Hash,
		},
	}
}

func NewFinalizeCovenant(name string, weak bool, renewalHash gcrypto.Hash, height, claimed, renewals int) *Covenant {
	var flags uint8
	if weak {
		flags |= 1
	}

	return &Covenant{
		Type: CovenantFinalize,
		Items: [][]byte{
			HashName(name),
			bio.Uint32LE(uint32(height)),
			[]byte(name),
			{flags},
			bio.Uint32LE(uint32(claimed)),
			bio.Uint32LE(uint32(renewals)),
			renewalHash,
		},
	}
}
