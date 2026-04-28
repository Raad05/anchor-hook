package decoder

import (
	"encoding/base64"
	"encoding/binary"
	"strings"
	"testing"
)

// buildPayload constructs a valid Borsh-encoded UserAction payload with
// the correct discriminator, suitable for passing to DecodeUserAction.
func buildPayload(user [32]byte, actionType string, amount uint64) string {
	buf := make([]byte, 0, 8+32+4+len(actionType)+8)

	// Discriminator
	buf = append(buf, userActionDiscriminator[:]...)

	// Pubkey
	buf = append(buf, user[:]...)

	// action_type string (u32 LE length prefix + bytes)
	strLen := make([]byte, 4)
	binary.LittleEndian.PutUint32(strLen, uint32(len(actionType)))
	buf = append(buf, strLen...)
	buf = append(buf, []byte(actionType)...)

	// amount u64 LE
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, amount)
	buf = append(buf, amountBytes...)

	return base64.StdEncoding.EncodeToString(buf)
}

// ── DecodeUserAction tests ────────────────────────────────────────────────────

func TestDecodeUserAction_HappyPath(t *testing.T) {
	var user [32]byte
	user[0] = 0xAB
	user[31] = 0xCD

	b64 := buildPayload(user, "transfer", 999_000_000)

	ua, err := DecodeUserAction(b64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ua.User != user {
		t.Errorf("user: got %v, want %v", ua.User, user)
	}
	if ua.ActionType != "transfer" {
		t.Errorf("actionType: got %q, want %q", ua.ActionType, "transfer")
	}
	if ua.Amount != 999_000_000 {
		t.Errorf("amount: got %d, want %d", ua.Amount, 999_000_000)
	}
}

func TestDecodeUserAction_ZeroAmount(t *testing.T) {
	b64 := buildPayload([32]byte{}, "stake", 0)
	ua, err := DecodeUserAction(b64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ua.Amount != 0 {
		t.Errorf("amount: got %d, want 0", ua.Amount)
	}
}

func TestDecodeUserAction_WrongDiscriminator(t *testing.T) {
	raw := make([]byte, 52)
	// Leave discriminator as all zeros (wrong).
	b64 := base64.StdEncoding.EncodeToString(raw)

	_, err := DecodeUserAction(b64)
	if err == nil {
		t.Fatal("expected error for wrong discriminator, got nil")
	}
	if !strings.Contains(err.Error(), "discriminator mismatch") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestDecodeUserAction_TooShort(t *testing.T) {
	raw := []byte{1, 2, 3}
	b64 := base64.StdEncoding.EncodeToString(raw)

	_, err := DecodeUserAction(b64)
	if err == nil {
		t.Fatal("expected error for short payload, got nil")
	}
	if !strings.Contains(err.Error(), "too short") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestDecodeUserAction_InvalidBase64(t *testing.T) {
	_, err := DecodeUserAction("!!!not-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

func TestDecodeUserAction_TruncatedString(t *testing.T) {
	// Build a payload where the string length prefix claims 100 bytes but the
	// buffer only has 8 bytes after the prefix.
	buf := make([]byte, 8+32+4+8)
	copy(buf[:8], userActionDiscriminator[:])
	// Pubkey stays zero.
	binary.LittleEndian.PutUint32(buf[40:], 100) // claims 100 bytes
	// No string bytes follow.
	b64 := base64.StdEncoding.EncodeToString(buf)

	_, err := DecodeUserAction(b64)
	if err == nil {
		t.Fatal("expected error for truncated string, got nil")
	}
}

// ── FindProgramData tests ─────────────────────────────────────────────────────

func TestFindProgramData_Found(t *testing.T) {
	logs := []string{
		"Program DMSM65 invoke [1]",
		"Program data: abc123==",
		"Program DMSM65 success",
	}
	payload, ok := FindProgramData(logs)
	if !ok {
		t.Fatal("expected to find program data")
	}
	if payload != "abc123==" {
		t.Errorf("payload: got %q, want %q", payload, "abc123==")
	}
}

func TestFindProgramData_NotFound(t *testing.T) {
	logs := []string{
		"Program DMSM65 invoke [1]",
		"Program DMSM65 success",
	}
	_, ok := FindProgramData(logs)
	if ok {
		t.Fatal("expected not to find program data")
	}
}

func TestFindProgramData_EmptyLogs(t *testing.T) {
	_, ok := FindProgramData(nil)
	if ok {
		t.Fatal("expected not to find program data in empty log slice")
	}
}

// ── base58Encode tests ────────────────────────────────────────────────────────

func TestUserBase58_KnownKey(t *testing.T) {
	// All-zeros pubkey should encode to a string of '1' characters.
	ua := &UserAction{}
	result := ua.UserBase58()
	for _, c := range result {
		if c != '1' {
			t.Errorf("expected all '1's for zero pubkey, got %q", result)
			break
		}
	}
}
