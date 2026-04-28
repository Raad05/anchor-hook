// Package decoder parses Solana program logs produced by the Stage-1 Anchor
// program and extracts UserAction events without any external dependencies.
//
// Anchor event encoding recap
// ─────────────────────────────────────────────────────────────────────────────
//  Program log: "Program data: <base64>"
//
//  Decoded bytes layout (Borsh):
//   [0..8)   – 8-byte event discriminator (SHA256("event:UserAction")[0:8])
//   [8..40)  – 32-byte Pubkey  (user)
//   [40..44) – u32 LE string length prefix
//   [44..44+n) – UTF-8 action_type bytes
//   [44+n..44+n+8) – u64 LE amount
//
// The discriminator for UserAction is: [78 49 255 143 219 92 187 207]
// (read from contracts/target/idl/contracts.json).
package decoder

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// userActionDiscriminator is the first 8 bytes produced by
// SHA256("event:UserAction"). It is baked in from the generated IDL so we
// don't need to hash at runtime.
var userActionDiscriminator = [8]byte{78, 49, 255, 143, 219, 92, 187, 207}

// programDataPrefix is the Anchor log line that carries the event payload.
const programDataPrefix = "Program data: "

// UserAction mirrors the on-chain Anchor event struct.
type UserAction struct {
	// User is the raw 32-byte Solana public key.
	User [32]byte
	// ActionType is a human-readable label (e.g. "transfer", "stake").
	ActionType string
	// Amount is a generic numeric quantity (lamports, tokens, …).
	Amount uint64
}

// UserBase58 returns the User public key encoded in base58.
// We implement a minimal base58 encoder here to stay dependency-free.
func (ua *UserAction) UserBase58() string {
	return base58Encode(ua.User[:])
}

// FindProgramData scans the slice of log lines returned by a Solana
// logsSubscribe notification and returns the raw base64 payload from the
// first "Program data: …" line it finds, plus a boolean indicating success.
func FindProgramData(logs []string) (string, bool) {
	for _, line := range logs {
		if strings.HasPrefix(line, programDataPrefix) {
			return strings.TrimPrefix(line, programDataPrefix), true
		}
	}
	return "", false
}

// DecodeUserAction decodes a base64-encoded Anchor event payload into a
// UserAction struct.  It returns an error if:
//   - the base64 is malformed
//   - the discriminator does not match UserAction
//   - the payload is too short to contain all fields
func DecodeUserAction(b64 string) (*UserAction, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	// Minimum size: 8 (disc) + 32 (pubkey) + 4 (str len) + 0 (str) + 8 (u64) = 52
	if len(raw) < 52 {
		return nil, errors.New("payload too short")
	}

	// ── Discriminator ────────────────────────────────────────────────────────
	var disc [8]byte
	copy(disc[:], raw[:8])
	if disc != userActionDiscriminator {
		return nil, fmt.Errorf("discriminator mismatch: got %v, want %v", disc, userActionDiscriminator)
	}

	offset := 8

	// ── Pubkey (32 bytes) ─────────────────────────────────────────────────────
	var ua UserAction
	copy(ua.User[:], raw[offset:offset+32])
	offset += 32

	// ── String length prefix (u32 LE) ─────────────────────────────────────────
	strLen := int(binary.LittleEndian.Uint32(raw[offset : offset+4]))
	offset += 4

	if len(raw) < offset+strLen+8 {
		return nil, fmt.Errorf("payload truncated: need %d bytes for string+amount, have %d", strLen+8, len(raw)-offset)
	}

	// ── action_type string ────────────────────────────────────────────────────
	ua.ActionType = string(raw[offset : offset+strLen])
	offset += strLen

	// ── amount (u64 LE) ───────────────────────────────────────────────────────
	ua.Amount = binary.LittleEndian.Uint64(raw[offset : offset+8])

	return &ua, nil
}

// ─── minimal base58 encoder (no deps) ────────────────────────────────────────

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func base58Encode(input []byte) string {
	// Count leading zero bytes → leading '1' characters.
	leadingZeros := 0
	for _, b := range input {
		if b != 0 {
			break
		}
		leadingZeros++
	}

	// Copy into a mutable slice so we can do in-place division.
	digits := make([]byte, len(input))
	copy(digits, input)

	result := make([]byte, 0, len(input)*136/100)
	for {
		allZero := true
		remainder := 0
		for i := range digits {
			val := remainder*256 + int(digits[i])
			digits[i] = byte(val / 58)
			remainder = val % 58
			if digits[i] != 0 {
				allZero = false
			}
		}
		result = append(result, base58Alphabet[remainder])
		if allZero {
			break
		}
	}

	// Add leading '1's.
	for i := 0; i < leadingZeros; i++ {
		result = append(result, '1')
	}

	// Reverse.
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return string(result)
}
