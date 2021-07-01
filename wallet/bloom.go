package wallet

import (
	"bytes"
	"github.com/bits-and-blooms/bloom/v3"
	"github.com/kurumiimari/gohan/chain"
	"github.com/pkg/errors"
)

// https://hur.st/bloomfilter/?n=1M&p=1.0E-7&m=&k=

const (
	AddrBloomM = 3354775
	AddrBloomK = 23

	OutpointBloomM = 3354775
	OutpointBloomK = 23
)

type OutpointBloom struct {
	filter *bloom.BloomFilter
}

func NewOutpointBloomFromOutpoints(o []*chain.Outpoint) *OutpointBloom {
	filter := bloom.New(OutpointBloomM, OutpointBloomK)
	buf := new(bytes.Buffer)
	for _, el := range o {
		buf.Reset()
		if _, err := el.WriteTo(buf); err != nil {
			panic(err)
		}
		filter.Add(buf.Bytes())
	}
	return &OutpointBloom{
		filter: filter,
	}
}

func NewOutpointBloomFromBytes(buf []byte) (*OutpointBloom, error) {
	r := bytes.NewReader(buf)
	filter := new(bloom.BloomFilter)
	if _, err := filter.ReadFrom(r); err != nil {
		return nil, errors.WithStack(err)
	}
	return &OutpointBloom{
		filter: filter,
	}, nil
}

func MustOutpointBloomFromBytes(buf []byte) *OutpointBloom {
	b, err := NewOutpointBloomFromBytes(buf)
	if err != nil {
		panic(err)
	}
	return b
}

func (o *OutpointBloom) Add(op *chain.Outpoint) {
	buf := new(bytes.Buffer)
	if _, err := op.WriteTo(buf); err != nil {
		panic(err)
	}
	o.filter.Add(buf.Bytes())
}

func (o *OutpointBloom) Test(op *chain.Outpoint) bool {
	buf := new(bytes.Buffer)
	if _, err := op.WriteTo(buf); err != nil {
		panic(err)
	}
	return o.filter.Test(buf.Bytes())
}

func (o *OutpointBloom) Bytes() []byte {
	buf := new(bytes.Buffer)
	if _, err := o.filter.WriteTo(buf); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func (o *OutpointBloom) Copy() *OutpointBloom {
	return &OutpointBloom{filter: o.filter.Copy()}
}
