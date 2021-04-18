package chain

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/kurumiimari/gohan/bio"
	"github.com/pkg/errors"
	"github.com/miekg/dns"
	"io"
	"math"
	"net"
)

type RecordType uint8

func (r RecordType) String() string {
	return recordTypeNames[int(r)]
}

const (
	RecordTypeDS RecordType = iota
	RecordTypeNS
	RecordTypeGlue4
	RecordTypeGlue6
	RecordTypeSynth4
	RecordTypeSynth6
	RecordTypeTXT
)

var recordTypeNames = [8]string{
	"DS",
	"NS",
	"GLUE4",
	"GLUE6",
	"SYNTH4",
	"SYNTH6",
	"TXT",
}

type Record interface {
	Type() RecordType
}

type RecordWriterTo interface {
	WriteTo(w *RecordWriter) (int64, error)
}

type RecordReaderFrom interface {
	ReadFrom(r *RecordReader) (int64, error)
}

type RecordReader struct {
	b           []byte
	off         int
	compressMap map[string]int
}

func NewRecordReader(b []byte) *RecordReader {
	return &RecordReader{
		b:           b,
		compressMap: make(map[string]int),
	}
}

func (r *RecordReader) Read(b []byte) (int, error) {
	bl := len(r.b)
	if r.off == bl {
		return 0, io.EOF
	}

	br := copy(b, r.b[r.off:])
	r.off += br
	return br, nil
}

func (r *RecordReader) ReadName(s *string) (int64, error) {
	name, newOff, err := dns.UnpackDomainName(r.b, r.off)
	if err != nil {
		return 0, err
	}
	read := int64(newOff - r.off)
	*s = name
	r.off = newOff
	return read, nil
}

type RecordWriter struct {
	b           []byte
	off         int
	compressMap map[string]int
}

func NewRecordWriter() *RecordWriter {
	return &RecordWriter{
		b:           make([]byte, 1024, 1024),
		compressMap: make(map[string]int),
	}
}

func (w *RecordWriter) Write(b []byte) (int, error) {
	wl := len(w.b)
	bl := len(b)
	if w.off == wl || w.off+bl > wl {
		return 0, errors.New("oversize write")
	}

	copy(w.b[w.off:], b)
	w.off += bl
	return bl, nil
}

func (w *RecordWriter) WriteName(name string) (int64, error) {
	newOff, err := dns.PackDomainName(name, w.b, w.off, w.compressMap, true)
	if err != nil {
		return 0, err
	}
	written := int64(newOff - w.off)
	w.off = newOff
	return written, nil
}

func (w *RecordWriter) Bytes() []byte {
	return w.b[:w.off]
}

type DSRecord struct {
	KeyTag     uint16
	Algorithm  uint8
	DigestType uint8
	Digest     []byte
}

func (ds *DSRecord) Type() RecordType {
	return RecordTypeDS
}

func (ds *DSRecord) WriteTo(w io.Writer) (int64, error) {
	if len(ds.Digest) > math.MaxUint8 {
		return 0, errors.New("digest must be less than 256 bytes")
	}
	g := bio.NewGuardWriter(w)
	bio.WriteUint16BE(g, ds.KeyTag)
	bio.WriteByte(g, ds.Algorithm)
	bio.WriteByte(g, ds.DigestType)
	bio.WriteByte(g, uint8(len(ds.Digest)))
	bio.WriteRawBytes(g, ds.Digest)
	return g.N, g.Err
}

func (ds *DSRecord) ReadFrom(r io.Reader) (int64, error) {
	g := bio.NewGuardReader(r)
	keyTag, _ := bio.ReadUint16BE(g)
	alg, _ := bio.ReadByte(g)
	dt, _ := bio.ReadByte(g)
	dl, _ := bio.ReadByte(g)
	digest, _ := bio.ReadFixedBytes(g, int(dl))
	if g.Err != nil {
		return g.N, g.Err
	}
	ds.KeyTag = keyTag
	ds.Algorithm = alg
	ds.DigestType = dt
	ds.Digest = digest
	return g.N, nil
}

func (ds *DSRecord) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type       string `json:"type"`
		KeyTag     uint16 `json:"key_tag"`
		Algorithm  uint8  `json:"algorithm"`
		DigestType uint8  `json:"digest_type"`
		Digest     string `json:"digest"`
	}{
		Type:       ds.Type().String(),
		KeyTag:     ds.KeyTag,
		Algorithm:  ds.Algorithm,
		DigestType: ds.DigestType,
		Digest:     hex.EncodeToString(ds.Digest),
	})
}

func (ds *DSRecord) UnmarshalJSON(bytes []byte) error {
	tmp := struct {
		Type       string `json:"type"`
		KeyTag     uint16 `json:"key_tag"`
		Algorithm  uint8  `json:"algorithm"`
		DigestType uint8  `json:"digest_type"`
		Digest     string `json:"digest"`
	}{}
	if err := json.Unmarshal(bytes, &tmp); err != nil {
		return err
	}

	digest, err := hex.DecodeString(tmp.Digest)
	if err != nil {
		return err
	}

	ds.KeyTag = tmp.KeyTag
	ds.Algorithm = tmp.Algorithm
	ds.DigestType = tmp.DigestType
	ds.Digest = digest
	return nil
}

type NSRecord struct {
	NS string `json:"ns"`
}

func (ns *NSRecord) Type() RecordType {
	return RecordTypeNS
}

func (ns *NSRecord) WriteTo(w *RecordWriter) (int64, error) {
	return w.WriteName(ns.NS)
}

func (ns *NSRecord) ReadFrom(r *RecordReader) (int64, error) {
	return r.ReadName(&ns.NS)
}

func (ns *NSRecord) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		NS   string `json:"ns"`
	}{
		Type: ns.Type().String(),
		NS:   ns.NS,
	})
}

type glueRecord struct {
	NS      string
	Address net.IP
}

func (g *glueRecord) writeTo(w *RecordWriter, is6 bool) (int64, error) {
	total, err := w.WriteName(g.NS)
	if err != nil {
		return total, err
	}
	var addr net.IP
	if is6 {
		addr = g.Address.To16()
	} else {
		addr = g.Address.To4()
	}
	n, err := bio.WriteRawBytes(w, addr)
	return total + int64(n), err
}

func (g *glueRecord) readFrom(r *RecordReader, is6 bool) (int64, error) {
	total, err := r.ReadName(&g.NS)
	if err != nil {
		return total, err
	}
	toRead := 4
	if is6 {
		toRead = 16
	}
	addr, err := bio.ReadFixedBytes(r, toRead)
	return total + int64(len(addr)), err
}

func (g *glueRecord) UnmarshalJSON(bytes []byte) error {
	tmp := struct {
		NS      string `json:"ns"`
		Address string `json:"address"`
	}{}
	if err := json.Unmarshal(bytes, &tmp); err != nil {
		return err
	}

	address := net.ParseIP(tmp.Address)
	if address == nil {
		return errors.New("invalid address")
	}

	g.NS = tmp.NS
	g.Address = address
	return nil
}

type Glue4Record struct {
	*glueRecord
}

func (g *Glue4Record) Type() RecordType {
	return RecordTypeGlue4
}

func (g *Glue4Record) WriteTo(w *RecordWriter) (int64, error) {
	return g.writeTo(w, false)
}

func (g *Glue4Record) ReadFrom(r *RecordReader) (int64, error) {
	return g.readFrom(r, false)
}

func (g *Glue4Record) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type    string `json:"type"`
		NS      string `json:"ns"`
		Address string `json:"address"`
	}{
		Type:    g.Type().String(),
		NS:      g.NS,
		Address: g.Address.String(),
	})
}

type Glue6Record struct {
	*glueRecord
}

func (g *Glue6Record) Type() RecordType {
	return RecordTypeGlue6
}

func (g *Glue6Record) WriteTo(w *RecordWriter) (int64, error) {
	return g.writeTo(w, true)
}

func (g *Glue6Record) ReadFrom(r *RecordReader) (int64, error) {
	return g.readFrom(r, true)
}

func (g *Glue6Record) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type    string `json:"type"`
		NS      string `json:"ns"`
		Address string `json:"address"`
	}{
		Type:    g.Type().String(),
		NS:      g.NS,
		Address: g.Address.String(),
	})
}

type synthRecord struct {
	Address net.IP
}

func (s *synthRecord) writeTo(w io.Writer, is6 bool) (int64, error) {
	if is6 {
		n, err := bio.WriteRawBytes(w, s.Address.To4())
		return int64(n), err
	}

	n, err := bio.WriteRawBytes(w, s.Address.To16())
	return int64(n), err
}

func (s *synthRecord) readFrom(r io.Reader, is6 bool) (int64, error) {
	toRead := 4
	if is6 {
		toRead = 16
	}
	b, err := bio.ReadFixedBytes(r, toRead)
	s.Address = b
	return int64(len(b)), err
}

func (s *synthRecord) UnmarshalJSON(bytes []byte) error {
	tmp := struct {
		Type    string `json:"type"`
		Address string `json:"address"`
	}{}
	if err := json.Unmarshal(bytes, &tmp); err != nil {
		return err
	}

	address := net.ParseIP(tmp.Address)
	if tmp.Type == recordTypeNames[RecordTypeSynth4] {
		address = address.To4()
	} else {
		address = address.To16()
	}

	if address == nil {
		return errors.New("invalid address")
	}

	s.Address = address
	return nil
}

type Synth4Record struct {
	*synthRecord
}

func (s *Synth4Record) Type() RecordType {
	return RecordTypeSynth4
}

func (s *Synth4Record) WriteTo(w io.Writer) (int64, error) {
	return s.writeTo(w, false)
}

func (s *Synth4Record) ReadFrom(r io.Reader) (int64, error) {
	return s.readFrom(r, false)
}

func (s *Synth4Record) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type    string `json:"type"`
		Address string `json:"address"`
	}{
		Type:    s.Type().String(),
		Address: s.Address.To4().String(),
	})
}

type Synth6Record struct {
	*synthRecord
}

func (s *Synth6Record) Type() RecordType {
	return RecordTypeSynth6
}

func (s *Synth6Record) WriteTo(w io.Writer) (int64, error) {
	return s.writeTo(w, true)
}

func (s *Synth6Record) ReadFrom(r io.Reader) (int64, error) {
	return s.readFrom(r, true)
}

func (s *Synth6Record) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type    string `json:"type"`
		Address string `json:"address"`
	}{
		Type:    s.Type().String(),
		Address: s.Address.To16().String(),
	})
}

type TXTRecord struct {
	Entries []string
}

func (t *TXTRecord) Type() RecordType {
	return RecordTypeTXT
}

func (t *TXTRecord) WriteTo(w io.Writer) (int64, error) {
	if len(t.Entries) > math.MaxUint8 {
		return 0, errors.New("too many entries")
	}

	g := bio.NewGuardWriter(w)
	bio.WriteByte(g, uint8(len(t.Entries)))
	for _, entry := range t.Entries {
		if len(entry) > math.MaxUint8 {
			return g.N, errors.New("entry too large")
		}
		bio.WriteByte(g, uint8(len(entry)))
		bio.WriteRawBytes(g, []byte(entry))
	}
	return g.N, g.Err
}

func (t *TXTRecord) ReadFrom(r io.Reader) (int64, error) {
	g := bio.NewGuardReader(r)
	count, _ := bio.ReadByte(g)
	entries := make([]string, count)
	for i := 0; i < int(count); i++ {
		entryLen, _ := bio.ReadByte(g)
		entry, _ := bio.ReadFixedBytes(g, int(entryLen))
		entries[i] = string(entry)
	}
	t.Entries = entries
	return g.N, g.Err
}

func (t *TXTRecord) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string   `json:"type"`
		TXT  []string `json:"txt"`
	}{
		t.Type().String(),
		t.Entries,
	})
}

func (t *TXTRecord) UnmarshalJSON(bytes []byte) error {
	tmp := struct {
		Type string   `json:"type"`
		TXT  []string `json:"txt"`
	}{}
	if err := json.Unmarshal(bytes, &tmp); err != nil {
		return err
	}

	t.Entries = tmp.TXT
	return nil
}

type Resource struct {
	TTL     int      `json:"ttl"`
	Records []Record `json:"records"`
}

func (rs *Resource) WriteTo(w io.Writer) (int64, error) {
	wr := NewRecordWriter()
	if _, err := bio.WriteByte(wr, 0); err != nil {
		return 0, err
	}
	for _, record := range rs.Records {
		if _, err := bio.WriteByte(wr, uint8(record.Type())); err != nil {
			return 0, err
		}
		switch rt := record.(type) {
		case io.WriterTo:
			if _, err := rt.WriteTo(wr); err != nil {
				return 0, err
			}
		case RecordWriterTo:
			if _, err := rt.WriteTo(wr); err != nil {
				return 0, err
			}
		default:
			panic("record type doesn't implement interface?")
		}
	}

	n, err := w.Write(wr.Bytes())
	return int64(n), err
}

func (rs *Resource) ReadFrom(b []byte) error {
	rr := NewRecordReader(b)
	ver, err := bio.ReadByte(rr)
	if err != nil {
		return err
	}
	if ver != 0 {
		return errors.New("invalid serialization version")
	}

	for {
		recType, err := bio.ReadByte(rr)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		var rec Record
		switch RecordType(recType) {
		case RecordTypeDS:
			rec = new(DSRecord)
		case RecordTypeNS:
			rec = new(NSRecord)
		case RecordTypeGlue4:
			rec = new(Glue4Record)
		case RecordTypeGlue6:
			rec = new(Glue6Record)
		case RecordTypeSynth4:
			rec = new(Synth4Record)
		case RecordTypeSynth6:
			rec = new(Synth6Record)
		case RecordTypeTXT:
			rec = new(TXTRecord)
		default:
			return errors.New("unknown record type")
		}

		switch rt := rec.(type) {
		case io.ReaderFrom:
			if _, err := rt.ReadFrom(rr); err != nil {
				return err
			}
		case RecordReaderFrom:
			if _, err := rt.ReadFrom(rr); err != nil {
				return err
			}
		default:
			panic("record type doesn't implement interface?")
		}
		rs.Records = append(rs.Records, rec)
	}
}

func (rs *Resource) UnmarshalJSON(bytes []byte) error {
	tmp := struct {
		TTL     int               `json:"ttl"`
		Records []json.RawMessage `json:"records"`
	}{}

	if err := json.Unmarshal(bytes, &tmp); err != nil {
		return err
	}

	records := make([]Record, len(tmp.Records))
	for i := 0; i < len(tmp.Records); i++ {
		recB := tmp.Records[i]
		jts := struct {
			Type string `json:"type"`
		}{}
		if err := json.Unmarshal(recB, &jts); err != nil {
			return err
		}

		var rec Record

		switch jts.Type {
		case recordTypeNames[RecordTypeDS]:
			rec = &DSRecord{}
		case recordTypeNames[RecordTypeNS]:
			rec = &NSRecord{}
		case recordTypeNames[RecordTypeGlue4]:
			rec = &Glue4Record{glueRecord: &glueRecord{}}
		case recordTypeNames[RecordTypeGlue6]:
			rec = &Glue6Record{glueRecord: &glueRecord{}}
		case recordTypeNames[RecordTypeSynth4]:
			rec = &Synth4Record{synthRecord: &synthRecord{}}
		case recordTypeNames[RecordTypeSynth6]:
			rec = &Synth6Record{synthRecord: &synthRecord{}}
		case recordTypeNames[RecordTypeTXT]:
			rec = &TXTRecord{}
		default:
			return fmt.Errorf("unknown type %s", jts.Type)
		}

		if err := json.Unmarshal(recB, rec); err != nil {
			return err
		}

		records[i] = rec
	}

	rs.TTL = tmp.TTL
	rs.Records = records
	return nil
}
