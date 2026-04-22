package hash

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestJSONMD5Hash_ReturnsHexEncoded(t *testing.T) {
	// Hex encoding of an MD5 digest is always 32 characters.
	h, err := JSONMD5Hash("hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(h) != 32 {
		t.Fatalf("expected 32-char hex hash, got %d chars: %q", len(h), h)
	}
	if _, err := hex.DecodeString(h); err != nil {
		t.Fatalf("hash is not valid hex: %v", err)
	}
}

func TestJSONMD5Hash_Deterministic(t *testing.T) {
	a, _ := JSONMD5Hash(map[string]string{"k": "v"})
	b, _ := JSONMD5Hash(map[string]string{"k": "v"})
	if a != b {
		t.Fatalf("expected deterministic hash, got %q vs %q", a, b)
	}
}

func TestDeterministicUUID_UsesRawBytes(t *testing.T) {
	// Regression for a bug where DeterministicUUID fed the *hex-encoded*
	// JSONMD5Hash string into uuid.FromBytes, producing UUIDs whose bytes
	// were the ASCII codes of hex digits (e.g. 30663964-3638-3061-... where
	// 0x30='0', 0x66='f', 0x39='9', 0x64='d'). The correct behavior is to
	// use the raw 16-byte md5 digest as the UUID bytes.

	seed := "test-seed"
	got, err := DeterministicUUID(seed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := json.Marshal(seed)
	sum := md5.Sum(data)
	// The UUID bytes must equal the raw md5 bytes.
	for i := 0; i < 16; i++ {
		if got[i] != sum[i] {
			t.Fatalf("UUID byte %d: want %#x, got %#x", i, sum[i], got[i])
		}
	}

	// Also assert the UUID is NOT the ASCII-hex-encoded variant of the hex
	// representation of the md5, which is what the old buggy code produced.
	hexStr := hex.EncodeToString(sum[:])
	var bogus [16]byte
	copy(bogus[:], hexStr[:16])
	if got == bogus {
		t.Fatal("DeterministicUUID regressed: still uses ASCII hex bytes as UUID bytes")
	}
}

func TestDeterministicUUID_Deterministic(t *testing.T) {
	a, _ := DeterministicUUID("same-seed")
	b, _ := DeterministicUUID("same-seed")
	if a != b {
		t.Fatalf("expected deterministic UUID, got %q vs %q", a, b)
	}
}

func TestDeterministicUUID_DifferentSeedsProduceDifferentUUIDs(t *testing.T) {
	a, _ := DeterministicUUID("seed-one")
	b, _ := DeterministicUUID("seed-two")
	if a == b {
		t.Fatalf("expected distinct UUIDs for distinct seeds, got %q twice", a)
	}
}

const passthroughUUIDStr = "550e8400-e29b-41d4-a716-446655440000"

func TestDeterministicUUID_PassesThroughValidUUIDString(t *testing.T) {
	got, err := DeterministicUUID(passthroughUUIDStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.String() != passthroughUUIDStr {
		t.Fatalf("expected passthrough %q, got %q", passthroughUUIDStr, got.String())
	}
}

func TestDeterministicUUID_PassesThroughUUIDValue(t *testing.T) {
	in := uuid.MustParse(passthroughUUIDStr)
	got, err := DeterministicUUID(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != in {
		t.Fatalf("expected passthrough %q, got %q", in, got)
	}
}

func TestDeterministicUUID_PassesThroughUUIDPointer(t *testing.T) {
	in := uuid.MustParse(passthroughUUIDStr)
	got, err := DeterministicUUID(&in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != in {
		t.Fatalf("expected passthrough %q, got %q", in, got)
	}
}

func TestDeterministicUUID_PassesThroughUUIDBytes(t *testing.T) {
	in := uuid.MustParse(passthroughUUIDStr)

	gotArr, err := DeterministicUUID([16]byte(in))
	if err != nil {
		t.Fatalf("unexpected error ([16]byte): %v", err)
	}
	if gotArr != in {
		t.Fatalf("[16]byte passthrough: want %q, got %q", in, gotArr)
	}

	gotSlice, err := DeterministicUUID(in[:])
	if err != nil {
		t.Fatalf("unexpected error ([]byte): %v", err)
	}
	if gotSlice != in {
		t.Fatalf("[]byte passthrough: want %q, got %q", in, gotSlice)
	}
}

func TestDeterministicUUID_PassesThroughNilUUID(t *testing.T) {
	const nilStr = "00000000-0000-0000-0000-000000000000"

	cases := []struct {
		name string
		in   any
	}{
		{"uuid.Nil value", uuid.Nil},
		{"nil string", nilStr},
		{"zero [16]byte", [16]byte{}},
	}
	for _, tc := range cases {
		got, err := DeterministicUUID(tc.in)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if got != uuid.Nil {
			t.Fatalf("%s: expected uuid.Nil passthrough, got %q", tc.name, got)
		}
	}
}

func TestDeterministicUUID_SingleElementSliceIsHashed(t *testing.T) {
	in := []string{passthroughUUIDStr}
	got, err := DeterministicUUID(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.String() == passthroughUUIDStr {
		t.Fatal("single-element slice must be treated as a composite and hashed, not unwrapped")
	}
	if got == uuid.Nil {
		t.Fatal("hashed composite should not be uuid.Nil")
	}
}

func TestDeterministicUUID_NonUUIDStringStillHashes(t *testing.T) {
	got, err := DeterministicUUID("test-seed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := json.Marshal("test-seed")
	sum := md5.Sum(data)
	if !bytes.Equal(got[:], sum[:]) {
		t.Fatalf("non-UUID strings must still hash; want %x, got %x", sum, got[:])
	}
}

func TestDeterministicUUID_ShortByteSliceStillHashes(t *testing.T) {
	in := []byte{1, 2, 3}
	got, err := DeterministicUUID(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := json.Marshal(in)
	sum := md5.Sum(data)
	if !bytes.Equal(got[:], sum[:]) {
		t.Fatalf("len != 16 []byte must hash via JSON-MD5; want %x, got %x", sum, got[:])
	}
}
