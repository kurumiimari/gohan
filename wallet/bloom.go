package wallet

import (
	"bytes"
	"github.com/kurumiimari/gohan/chain"
	"github.com/pkg/errors"
	"github.com/willf/bloom"
)

// https://hur.st/bloomfilter/?n=1M&p=1.0E-7&m=&k=

const (
	AddrBloomM = 3354775
	AddrBloomK = 23

	OutpointBloomM = 3354775
	OutpointBloomK = 23
)

type AddressBloom struct {
	filter *bloom.BloomFilter
}

func NewAddrBloomFromAddrs(addresses []*chain.Address) *AddressBloom {
	filter := bloom.New(AddrBloomM, AddrBloomK)
	buf := new(bytes.Buffer)
	for _, el := range addresses {
		buf.Reset()
		if _, err := el.WriteTo(buf); err != nil {
			panic(err)
		}
		filter.Add(buf.Bytes())
	}

	return &AddressBloom{
		filter: filter,
	}
}

func AddressBloomFromBytes(buf []byte) (*AddressBloom, error) {
	r := bytes.NewReader(buf)
	filter := new(bloom.BloomFilter)
	if _, err := filter.ReadFrom(r); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling bloom filter")
	}
	return &AddressBloom{
		filter: filter,
	}, nil
}

func MustAddressBloomFromBytes(buf []byte) *AddressBloom {
	b, err := AddressBloomFromBytes(buf)
	if err != nil {
		panic(err)
	}
	return b
}

func (a *AddressBloom) Add(address *chain.Address) {
	buf := new(bytes.Buffer)
	if _, err := address.WriteTo(buf); err != nil {
		panic(err)
	}
	a.filter.Add(buf.Bytes())
}

func (a *AddressBloom) Test(address *chain.Address) bool {
	buf := new(bytes.Buffer)
	if _, err := address.WriteTo(buf); err != nil {
		panic(err)
	}
	return a.filter.Test(buf.Bytes())
}

func (a *AddressBloom) Bytes() []byte {
	buf := new(bytes.Buffer)
	if _, err := a.filter.WriteTo(buf); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func (a *AddressBloom) Copy() *AddressBloom {
	return &AddressBloom{filter: a.filter.Copy()}
}

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
		return nil, errors.Wrap(err, "error unmarshaling bloom filter")
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
