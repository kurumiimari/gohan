package chain

import (
	"bytes"
	_ "embed"
	"github.com/kurumiimari/gohan/bio"
	"github.com/kurumiimari/gohan/gcrypto"
	"io"
)

const (
	MaxNameSize = 63
)

//go:embed names.db
var NamesDB []byte

var dbSize int
var dbValue uint64
var rootValue uint64
var topValue uint64

var nameCharset = []byte{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0,
	0, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 0, 0, 0, 0, 4,
	0, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 0, 0, 0, 0, 0,
}

var NamesBlacklist = map[string]bool{
	"example":   true,
	"invalid":   true,
	"local":     true,
	"localhost": true,
	"test":      true,
}

type ReservationInfo struct {
	Name   string
	Hash   []byte
	Target string
	Value  uint64
	Root   bool
}

func IsNameValid(name string) bool {
	if len(name) == 0 {
		return false
	}

	if len(name) > MaxNameSize {
		return false
	}

	for i := 0; i < len(name); i++ {
		chr := rune(name[i])
		if chr&0xff80 > 0 {
			return false
		}

		chType := nameCharset[chr]

		switch chType {
		case 0:
			return false
		case 1:
			break
		case 2:
			return false
		case 3:
			break
		case 4:
			if i == 0 || i == len(name)-1 {
				return false
			}
			break
		}
	}

	return !IsNameBlacklisted(name)
}

func IsNameBlacklisted(name string) bool {
	_, ok := NamesBlacklist[name]
	return ok
}

func HashName(name string) gcrypto.Hash {
	return gcrypto.SHA3256([]byte(name))
}

func HasRollout(network *Network, height int, name string) bool {
	start, _ := NameRolloutInfo(network, name)
	return height > start
}

func IsNameReserved(network *Network, height int, name string) bool {
	return IsNameHashReserved(network, height, HashName(name))
}

func IsNameHashReserved(network *Network, height int, hash []byte) bool {
	if !network.HasReserved {
		return false
	}

	if height >= network.ClaimPeriod {
		return false
	}

	return find(hash) != -1
}

func NameRolloutInfo(network *Network, name string) (int, int) {
	return NameHashRolloutInfo(network, HashName(name))
}

func NameHashRolloutInfo(network *Network, hash []byte) (int, int) {
	p := 256 % 52
	var week int
	for i := 0; i < len(hash); i++ {
		week = (p*week + int(hash[i])) % 52
	}

	height := week * network.RolloutInterval
	return network.AuctionStart + height, week
}

func ReservedNameInfo(name string) *ReservationInfo {
	return ReservedNameHashInfo(HashName(name))
}

func ReservedNameHashInfo(h []byte) *ReservationInfo {
	pos := find(h)
	if pos == -1 {
		return nil
	}
	l := int(NamesDB[pos])
	target := NamesDB[pos+1 : pos+1+l]
	flags := NamesDB[pos+1+l]
	index := NamesDB[pos+1+l+1]
	root := (flags & 1) != 0
	top100 := (flags & 2) != 0
	custom := (flags & 4) != 0
	zero := (flags & 8) != 0
	name := string(target[0:index])

	value := dbValue

	if root {
		value += rootValue
	}

	if top100 {
		value += topValue
	}

	if custom {
		value += read64(bytes.NewReader(NamesDB[pos+1+l+2:]))
	}

	if zero {
		value = 0
	}

	return &ReservationInfo{
		Name:   name,
		Hash:   h,
		Target: string(target),
		Value:  value,
		Root:   root,
	}
}

func find(h []byte) int {
	var start int
	end := dbSize - 1

	for start <= end {
		index := int(uint(start+end) >> uint(1))
		pos := 28 + index*36
		cmp := compare(h, pos)

		if cmp == 0 {
			out, _ := bio.ReadUint32LE(bytes.NewReader(NamesDB[pos+32:]))
			return int(out)
		}

		if cmp < 0 {
			start = index + 1
		} else {
			end = index - 1
		}
	}

	return -1
}

func compare(b []byte, off int) int {
	for i := 0; i < 32; i++ {
		x := NamesDB[off+i]
		y := b[i]

		if x < y {
			return -1
		}

		if x > y {
			return 1
		}
	}

	return 0
}

func read32(r io.ReadSeeker) uint32 {
	out, _ := bio.ReadUint32LE(r)
	return out
}

func read64(r io.ReadSeeker) uint64 {
	out, _ := bio.ReadUint64LE(r)
	return out
}

func init() {
	r := bytes.NewReader(NamesDB)
	dbSize = int(read32(r))
	dbValue = read64(r)
	rootValue = read64(r)
	topValue = read64(r)
}
