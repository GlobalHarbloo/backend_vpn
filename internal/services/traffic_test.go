package services

import (
	"encoding/hex"
	"testing"
)

func TestParseTrafficResponse_BinaryLE(t *testing.T) {
	// hex from user: 00 00 06 04 00 00 00 00
	data, err := hex.DecodeString("0000060400000000")
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	v, err := parseTrafficResponse(data)
	if err != nil {
		t.Fatalf("parseTrafficResponse failed: %v", err)
	}
	// expected little-endian uint64: 4<<24 + 6<<16 = 67108864 + 393216 = 67502080
	var expected int64 = 67502080
	if v != expected {
		t.Fatalf("unexpected value: got %d want %d", v, expected)
	}
}

func TestParseTrafficResponse_JSON(t *testing.T) {
	json := []byte(`{"jsonrpc":"2.0","id":1,"result":{"stat":{"name":"user>>>x>>>traffic>>>uplink","value":12345}}}`)
	v, err := parseTrafficResponse(json)
	if err != nil {
		t.Fatalf("parseTrafficResponse json failed: %v", err)
	}
	if v != 12345 {
		t.Fatalf("unexpected json value: got %d want %d", v, 12345)
	}
}
