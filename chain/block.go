package chain

import (
	"bytes"
	"encoding/hex"
	"github.com/kurumiimari/gohan/bio"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
	"io"
)

const (
	HashLen       = 32
	ExtraNonceLen = 24
)

type Block struct {
	Nonce        uint32
	Time         uint64
	PrevHash     []byte
	TreeRoot     []byte
	ExtraNonce   []byte
	ReservedRoot []byte
	WitnessRoot  []byte
	MerkleRoot   []byte
	Version      uint32
	Bits         uint32
	Mask         []byte
	Transactions []*Transaction
}

func NewBlockFromBytes(b []byte) (*Block, error) {
	block := new(Block)
	if _, err := block.ReadFrom(bytes.NewReader(b)); err != nil {
		return nil, err
	}
	return block, nil
}

func (b *Block) Hash() []byte {
	leftData := new(bytes.Buffer)
	bio.WriteUint32LE(leftData, b.Nonce)
	bio.WriteUint64LE(leftData, b.Time)
	leftData.Write(b.padding(20))
	leftData.Write(b.PrevHash)
	leftData.Write(b.TreeRoot)
	leftData.Write(b.commitHash())
	left := leftData.Bytes()

	leftH, _ := blake2b.New512(nil)
	leftH.Write(left)

	rightH := sha3.New256()
	rightH.Write(left)
	rightH.Write(b.padding(8))

	outH, _ := blake2b.New256(nil)
	outH.Write(leftH.Sum(nil))
	outH.Write(b.padding(32))
	outH.Write(rightH.Sum(nil))
	return outH.Sum(nil)
}

func (b *Block) HashHex() string {
	return hex.EncodeToString(b.Hash())
}

func (b *Block) WriteTo(w io.Writer) (int64, error) {
	g := bio.NewGuardWriter(w)
	bio.WriteUint32LE(g, b.Nonce)
	bio.WriteUint64LE(g, b.Time)
	bio.WriteFixedBytes(g, b.PrevHash, HashLen)
	bio.WriteFixedBytes(g, b.TreeRoot, HashLen)
	bio.WriteFixedBytes(g, b.ExtraNonce, ExtraNonceLen)
	bio.WriteFixedBytes(g, b.ReservedRoot, HashLen)
	bio.WriteFixedBytes(g, b.WitnessRoot, HashLen)
	bio.WriteFixedBytes(g, b.MerkleRoot, HashLen)
	bio.WriteUint32LE(g, b.Version)
	bio.WriteUint32LE(g, b.Bits)
	bio.WriteFixedBytes(g, b.Mask, HashLen)
	bio.WriteVarint(g, uint64(len(b.Transactions)))
	for _, tx := range b.Transactions {
		tx.WriteTo(g)
	}
	return g.N, errors.Wrap(g.Err, "error writing block")
}

func (b *Block) ReadFrom(r io.Reader) (int64, error) {
	g := bio.NewGuardReader(r)
	nonce, _ := bio.ReadUint32LE(g)
	time, _ := bio.ReadUint64LE(g)
	prevHash, _ := bio.ReadFixedBytes(g, HashLen)
	treeRoot, _ := bio.ReadFixedBytes(g, HashLen)
	extraNonce, _ := bio.ReadFixedBytes(g, ExtraNonceLen)
	resRoot, _ := bio.ReadFixedBytes(g, HashLen)
	witRoot, _ := bio.ReadFixedBytes(g, HashLen)
	merkRoot, _ := bio.ReadFixedBytes(g, HashLen)
	ver, _ := bio.ReadUint32LE(g)
	bits, _ := bio.ReadUint32LE(g)
	mask, _ := bio.ReadFixedBytes(g, HashLen)
	txCount, _ := bio.ReadVarint(g)
	var txs []*Transaction
	for i := 0; i < int(txCount); i++ {
		tx := new(Transaction)
		tx.ReadFrom(g)
		txs = append(txs, tx)
	}
	if g.Err != nil {
		return g.N, errors.Wrap(g.Err, "error reading block")
	}
	b.Nonce = nonce
	b.Time = time
	b.PrevHash = prevHash
	b.TreeRoot = treeRoot
	b.ExtraNonce = extraNonce
	b.ReservedRoot = resRoot
	b.WitnessRoot = witRoot
	b.MerkleRoot = merkRoot
	b.Version = ver
	b.Bits = bits
	b.Mask = mask
	b.Transactions = txs
	return g.N, nil
}

func (b *Block) commitHash() []byte {
	h, _ := blake2b.New256(nil)
	h.Write(b.subHash())
	h.Write(b.maskHash())
	return h.Sum(nil)
}

func (b *Block) subHash() []byte {
	h, _ := blake2b.New256(nil)
	h.Write(b.ExtraNonce)
	h.Write(b.ReservedRoot)
	h.Write(b.WitnessRoot)
	h.Write(b.MerkleRoot)
	bio.WriteUint32LE(h, b.Version)
	bio.WriteUint32LE(h, b.Bits)
	return h.Sum(nil)
}

func (b *Block) maskHash() []byte {
	h, _ := blake2b.New256(nil)
	h.Write(b.PrevHash)
	h.Write(b.Mask)
	return h.Sum(nil)
}

func (b *Block) padding(size int) []byte {
	buf := make([]byte, size, size)
	for i := 0; i < len(buf); i++ {
		buf[i] = b.PrevHash[i%32] ^ b.TreeRoot[i%32]
	}
	return buf
}
