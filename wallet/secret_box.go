package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"github.com/pkg/errors"
	"golang.org/x/crypto/argon2"
)

const (
	Argon2IDAESGCM256DecryptorType = "argon2id-aes-gcm-256"
)

type SecretBox interface {
	Decrypt(password string) ([]byte, error)
}

type Argon2AESGCM256SecretBox struct {
	Time   uint32 `json:"time"`
	Memory uint32 `json:"memory"`
	Salt   []byte `json:"salt"`
	KeyLen uint32 `json:"key_len"`
	Nonce  []byte `json:"nonce"`
	Tag    []byte `json:"tag"`
	CT     []byte `json:"ct"`
	Type   string `json:"type"`
}

func NewArgon2AESGCM256SecretBox(pt []byte, password string) (SecretBox, error) {
	c := &Argon2AESGCM256SecretBox{
		Time:   1,
		Memory: 64 * 1024,
		Salt:   RandBytes(32),
		KeyLen: 32,
		Nonce:  RandBytes(12),
		Tag:    RandBytes(32),
		Type:   Argon2IDAESGCM256DecryptorType,
	}

	key := argon2.IDKey(
		[]byte(password),
		c.Salt,
		c.Time,
		c.Memory,
		4,
		c.KeyLen,
	)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize block cipher")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create GCM cipher")
	}

	c.CT = gcm.Seal(nil, c.Nonce, pt, c.Tag)
	return c, nil
}

func (c *Argon2AESGCM256SecretBox) Decrypt(password string) ([]byte, error) {
	key := argon2.IDKey(
		[]byte(password),
		c.Salt,
		c.Time,
		c.Memory,
		4,
		c.KeyLen,
	)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize block cipher")
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create GCM cipher")
	}

	return gcm.Open(nil, c.Nonce, c.CT, c.Tag)
}

func EncryptDefault(pt []byte, password string) (SecretBox, error) {
	return NewArgon2AESGCM256SecretBox(pt, password)
}

func UnmarshalSecretBox(in []byte) (SecretBox, error) {
	tmp := struct {
		Type string `json:"type"`
	}{}

	if err := json.Unmarshal(in, &tmp); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling decryptor type")
	}

	var dec SecretBox
	switch tmp.Type {
	case Argon2IDAESGCM256DecryptorType:
		dec = &Argon2AESGCM256SecretBox{}
	default:
		return nil, errors.New("unknown decryptor type")
	}

	if err := json.Unmarshal(in, dec); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling decryptor")
	}

	return dec, nil
}
