package client

import "github.com/kurumiimari/gohan/gjson"

type GetBloomRes struct {
	Height        int              `json:"height"`
	AddressBloom  gjson.ByteString `json:"addressBloom"`
	OutpointBloom gjson.ByteString `json:"outpointBloom"`
}
