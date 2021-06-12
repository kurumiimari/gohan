package shakedex

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"github.com/btcsuite/btcd/btcec"
	"github.com/kurumiimari/gohan/chain"
	"github.com/kurumiimari/gohan/txscript"
	"github.com/pkg/errors"
	"io"
	"math"
	"sort"
	"strings"
	"time"
)

const (
	V1DutchAuctionMagic = "SHAKEDEX_PROOF:1.0.0"
)

var (
	ErrMalFormedProofFile = errors.New("mal-formed proof file")
)

type DutchAuction struct {
	Name            string
	LockingOutpoint *chain.Outpoint
	PublicKey       []byte
	PaymentAddress  *chain.Address
	FeeAddress      *chain.Address
	Bids            []*DutchAuctionBid

	lockingScript []byte
	sorted        bool
}

type DutchAuctionBid struct {
	Value     uint64
	LockTime  uint32
	Fee       uint64
	Signature []byte
}

func (d *DutchAuction) VerifyAllBids(lockCoinValue uint64) bool {
	for i := range d.Bids {
		if !d.VerifyBid(lockCoinValue, i) {
			return false
		}
	}

	return true
}

func (d *DutchAuction) VerifyBid(lockCoinValue uint64, bidIdx int) bool {
	tx, err := d.TXTemplate(lockCoinValue, bidIdx)
	if err != nil {
		return false
	}

	err = txscript.EngineStandardVerify(
		tx,
		0,
		tx.Outputs[0].Address,
		lockCoinValue,
	)
	return err == nil
}

func (d *DutchAuction) TXTemplate(lockCoinValue uint64, bidIdx int) (*chain.Transaction, error) {
	script, err := d.LockingScript()
	if err != nil {
		return nil, err
	}

	bid := d.Bids[bidIdx]
	tx := new(chain.Transaction)
	tx.LockTime = bid.LockTime
	tx.Inputs = append(tx.Inputs, &chain.Input{
		Prevout:  d.LockingOutpoint,
		Sequence: chain.DefaultSequence,
	})
	tx.Outputs = append(tx.Outputs, &chain.Output{
		Value:   lockCoinValue,
		Address: chain.NewAddressFromScript(script),
		Covenant: &chain.Covenant{
			Type: chain.CovenantTransfer,
		},
	})
	if bid.Fee > 0 {
		tx.Outputs = append(tx.Outputs, &chain.Output{
			Value:    bid.Fee,
			Address:  d.FeeAddress,
			Covenant: chain.EmptyCovenant,
		})
	}
	tx.Outputs = append(tx.Outputs, &chain.Output{
		Value:    bid.Value,
		Address:  d.PaymentAddress,
		Covenant: chain.EmptyCovenant,
	})
	tx.Witnesses = []*chain.Witness{
		{
			Items: [][]byte{
				bid.Signature,
				script,
			},
		},
	}
	return tx, nil
}

func (d *DutchAuction) LockingScript() ([]byte, error) {
	if d.lockingScript != nil {
		return d.lockingScript, nil
	}

	script, err := txscript.NewHIP1LockingScript(d.PublicKey)
	if err != nil {
		return nil, err
	}
	d.lockingScript = script
	return d.lockingScript, nil
}

func (d *DutchAuction) BestBid(lockTime uint32) int {
	if !d.sorted {
		sort.Sort(DutchAuctionBidsSort(d.Bids))
		d.sorted = true
	}

	lastBid := uint64(math.MaxUint64)
	for i, bid := range d.Bids {
		// make sure bids are correctly ordered with
		// their lock times
		if bid.Value > lastBid {
			return -1
		}

		if bid.LockTime <= lockTime {
			return i
		}
	}

	return -1
}

func (d *DutchAuction) AsCanonical() *CanonicalAuction {
	var feeAddr *string
	if d.FeeAddress != nil {
		bech := d.FeeAddress.String()
		feeAddr = &bech
	}

	bids := make([]*CanonicalBid, len(d.Bids))
	for i := 0; i < len(bids); i++ {
		bids[i] = &CanonicalBid{
			Price:     d.Bids[i].Value,
			LockTime:  d.Bids[i].LockTime,
			Fee:       d.Bids[i].Fee,
			Signature: hex.EncodeToString(d.Bids[i].Signature),
		}
	}

	return &CanonicalAuction{
		Name:             d.Name,
		LockingTxHash:    d.LockingOutpoint.Hash.String(),
		LockingOutputIdx: d.LockingOutpoint.Index,
		PublicKey:        hex.EncodeToString(d.PublicKey),
		PaymentAddr:      d.PaymentAddress.String(),
		FeeAddr:          feeAddr,
		Data:             bids,
	}
}

type DutchAuctionBidsSort []*DutchAuctionBid

func (srt DutchAuctionBidsSort) Len() int {
	return len(srt)
}

func (srt DutchAuctionBidsSort) Less(i, j int) bool {
	return srt[i].LockTime > srt[j].LockTime
}

func (srt DutchAuctionBidsSort) Swap(i, j int) {
	srt[i], srt[j] = srt[j], srt[i]
}

func CreateDutchAuction(flzOutpoint *chain.Outpoint, flzValue uint64, name string, auctionStart int64, start, end uint64, feeRatePercent float64, nDecrements int,
	decDuration time.Duration, paymentAddr *chain.Address, feeAddr *chain.Address, privKey *btcec.PrivateKey) (*DutchAuction, error) {
	if start < end {
		return nil, errors.New("start bid must be more than end bid")
	}

	priceDecrement := (start - end) / uint64(nDecrements)
	auction := &DutchAuction{
		Name:            name,
		LockingOutpoint: flzOutpoint,
		PublicKey:       privKey.PubKey().SerializeCompressed(),
		PaymentAddress:  paymentAddr,
		Bids:            make([]*DutchAuctionBid, nDecrements+1),
		FeeAddress:      feeAddr,
	}

	var sigHashes *txscript.TxSigHashes

	for i := 0; i < nDecrements+1; i++ {
		value := start - (priceDecrement * uint64(i))
		fee := uint64(float64(value) * feeRatePercent)
		valueLessFee := value - fee

		auction.Bids[i] = &DutchAuctionBid{
			Value:    valueLessFee,
			LockTime: uint32(auctionStart + int64(i)*int64(decDuration/time.Second)),
			Fee:      fee,
		}

		tx, err := auction.TXTemplate(flzValue, i)
		if err != nil {
			return nil, err
		}

		if sigHashes == nil {
			sigHashes = txscript.NewTxSigHashes(tx)
		}

		script, err := auction.LockingScript()
		if err != nil {
			return nil, err
		}
		sig, _, err := txscript.WitnessSignature(
			tx,
			sigHashes,
			0,
			flzValue,
			script,
			txscript.SigHashAnyOneCanPay|txscript.SigHashSingleReverse,
			privKey,
		)
		if err != nil {
			return nil, err
		}

		auction.Bids[i].Signature = sig
	}

	return auction, nil
}

type CanonicalAuction struct {
	Name             string          `json:"name"`
	LockingTxHash    string          `json:"lockingTxHash"`
	LockingOutputIdx uint32          `json:"lockingOutputIdx"`
	PublicKey        string          `json:"publicKey"`
	PaymentAddr      string          `json:"paymentAddr"`
	FeeAddr          *string         `json:"feeAddr"`
	Data             []*CanonicalBid `json:"data"`
}

type CanonicalBid struct {
	Price     uint64 `json:"price"`
	LockTime  uint32 `json:"lockTime"`
	Fee       uint64 `json:"fee"`
	Signature string `json:"signature"`
}

func ReadDutchAuctionProof(r io.Reader) (*DutchAuction, error) {
	br := bufio.NewReader(r)
	header, err := br.ReadString('\n')
	if err != nil {
		return nil, errors.WithStack(ErrMalFormedProofFile)
	}

	if strings.TrimSpace(header) != V1DutchAuctionMagic {
		return nil, errors.Wrap(ErrMalFormedProofFile, "invalid proof header")
	}

	limR := io.LimitReader(br, 10*1024*1024)
	dec := json.NewDecoder(limR)
	auctionJ := new(CanonicalAuction)
	if err := dec.Decode(auctionJ); err != nil {
		return nil, errors.Wrap(err, "mal-formed proof file")
	}

	if len(auctionJ.Data) == 0 {
		return nil, errors.WithStack(ErrMalFormedProofFile)
	}

	hashBytes, err := hex.DecodeString(auctionJ.LockingTxHash)
	if err != nil {
		return nil, errors.WithStack(ErrMalFormedProofFile)
	}
	pubB, err := hex.DecodeString(auctionJ.PublicKey)
	if err != nil {
		return nil, errors.WithStack(ErrMalFormedProofFile)
	}
	payAddr, err := chain.NewAddressFromBech32(auctionJ.PaymentAddr)
	if err != nil {
		return nil, errors.WithStack(ErrMalFormedProofFile)
	}
	var feeAddr *chain.Address
	if auctionJ.FeeAddr != nil {
		feeAddr, err = chain.NewAddressFromBech32(*auctionJ.FeeAddr)
		if err != nil {
			return nil, errors.WithStack(ErrMalFormedProofFile)
		}
	}

	bids := make([]*DutchAuctionBid, len(auctionJ.Data))
	for i, datum := range auctionJ.Data {
		sig, err := hex.DecodeString(datum.Signature)
		if err != nil {
			return nil, errors.WithStack(ErrMalFormedProofFile)
		}

		bids[i] = &DutchAuctionBid{
			Value:     datum.Price,
			LockTime:  datum.LockTime,
			Fee:       datum.Fee,
			Signature: sig,
		}
	}

	return &DutchAuction{
		Name: auctionJ.Name,
		LockingOutpoint: &chain.Outpoint{
			Hash:  hashBytes,
			Index: auctionJ.LockingOutputIdx,
		},
		PublicKey:      pubB,
		PaymentAddress: payAddr,
		Bids:           bids,
		FeeAddress:     feeAddr,
	}, nil
}

func WriteDutchAuctionProof(auction *DutchAuction, w io.Writer) error {
	j := auction.AsCanonical()

	if _, err := w.Write([]byte(V1DutchAuctionMagic)); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(j); err != nil {
		return err
	}
	return nil
}
