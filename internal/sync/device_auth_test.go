package sync

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDeviceAuthManager_ValidSignatureAndReplayProtection(t *testing.T) {
	ctx := context.Background()
	valID := "VAL_500T_01"
	mgr := NewDeviceAuthManager(map[string]string{valID: "test-secret-key-12345-at-least-32-chars-long"}, nil)
	seq := int64(100)
	ts := time.Now().Unix()
	nonce := "random-nonce-abc-123"
	payloadHash := ComputePayloadHash([]byte(`{"card_id":"C1"}`))

	sig := mgr.GenerateSignature(valID, seq, ts, nonce, payloadHash)

	// 1. Valid signature succeeds
	err := mgr.VerifySignature(ctx, valID, seq, ts, nonce, payloadHash, sig)
	if err != nil {
		t.Fatalf("expected valid signature, got: %v", err)
	}

	// 2. Replay attack: same nonce -> must be rejected
	sigSameNonce := mgr.GenerateSignature(valID, seq+1, ts, nonce, payloadHash)
	err = mgr.VerifySignature(ctx, valID, seq+1, ts, nonce, payloadHash, sigSameNonce)
	if err != ErrReplayDetected {
		t.Fatalf("expected ErrReplayDetected for repeated nonce, got: %v", err)
	}

	// 3. Replay attack: lower or equal sequence number -> must be rejected
	newNonce := "random-nonce-def-456"
	sigOldSeq := mgr.GenerateSignature(valID, seq, ts, newNonce, payloadHash)
	err = mgr.VerifySignature(ctx, valID, seq, ts, newNonce, payloadHash, sigOldSeq)
	if err != ErrReplayDetected {
		t.Fatalf("expected ErrReplayDetected for non-monotonic sequence, got: %v", err)
	}

	// 4. Tampered payload hash -> must be rejected
	tamperedHash := ComputePayloadHash([]byte(`{"card_id":"C2"}`))
	err = mgr.VerifySignature(ctx, valID, seq+2, ts, "nonce-3", tamperedHash, sig)
	if err != ErrInvalidDeviceSignature {
		t.Fatalf("expected ErrInvalidDeviceSignature for tampered payload, got: %v", err)
	}
}

func TestDeviceAuthManager_TimestampOutOfBounds(t *testing.T) {
	ctx := context.Background()
	valID := "VAL_500T_01"
	mgr := NewDeviceAuthManager(map[string]string{valID: "test-secret-key-12345-at-least-32-chars-long"}, nil)
	seq := int64(1)
	oldTs := time.Now().Unix() - 1000 // > 15 mins (900s)
	nonce := "nonce-old"
	payloadHash := ComputePayloadHash([]byte(`{}`))
	sig := mgr.GenerateSignature(valID, seq, oldTs, nonce, payloadHash)

	err := mgr.VerifySignature(ctx, valID, seq, oldTs, nonce, payloadHash, sig)
	if err != ErrTimestampOutOfBounds {
		t.Fatalf("expected ErrTimestampOutOfBounds for old timestamp, got: %v", err)
	}
}

func TestDeviceAuthManager_RevocationAndFallback(t *testing.T) {
	ctx := context.Background()
	valID := "VAL_REVOKED_01"
	mgr := NewDeviceAuthManager(map[string]string{valID: "test-secret-key-12345-at-least-32-chars-long"}, nil)
	seq := int64(10)
	ts := time.Now().Unix()
	nonce := "nonce-rev"
	payloadHash := ComputePayloadHash([]byte(`{"test":1}`))
	sig := mgr.GenerateSignature(valID, seq, ts, nonce, payloadHash)

	// Revoke device in memory
	mgr.revokedIDs[valID] = true
	err := mgr.VerifySignature(ctx, valID, seq, ts, nonce, payloadHash, sig)
	if err != ErrDeviceRevoked {
		t.Fatalf("expected ErrDeviceRevoked, got: %v", err)
	}

	// Empty validator or signature
	if mgr.VerifySignature(ctx, "", seq, ts, nonce, payloadHash, sig) != ErrInvalidDeviceSignature {
		t.Fatalf("expected ErrInvalidDeviceSignature for empty validatorID")
	}
	if mgr.VerifySignature(ctx, valID, seq, ts, nonce, payloadHash, "") != ErrInvalidDeviceSignature {
		t.Fatalf("expected ErrInvalidDeviceSignature for empty signature")
	}
}

func TestDeviceAuthManager_VerifyBatchSignature(t *testing.T) {
	ctx := context.Background()
	valID := "VAL_500T_01"
	secret := "test-secret-key-12345-at-least-32-chars-long"
	mgr := NewDeviceAuthManager(map[string]string{valID: secret}, nil)
	seqNos := []int64{101, 102, 103}
	signingSeq := int64(103)
	ts := time.Now().Unix()
	nonce := "batch-nonce-001"
	payloadHash := ComputePayloadHash([]byte(`[{"id":1},{"id":2},{"id":3}]`))

	sig := mgr.GenerateSignature(valID, signingSeq, ts, nonce, payloadHash)

	// 1. Valid batch signature succeeds
	err := mgr.VerifyBatchSignature(ctx, valID, signingSeq, seqNos, ts, nonce, payloadHash, sig)
	if err != nil {
		t.Fatalf("expected batch signature to verify, got: %v", err)
	}

	// 2. Exact idempotent retry succeeds
	err = mgr.VerifyBatchSignature(ctx, valID, signingSeq, seqNos, ts, nonce, payloadHash, sig)
	if err != nil {
		t.Fatalf("expected idempotent batch retry to succeed, got: %v", err)
	}

	// 3. Replay with different payload returns ErrReplayDetected
	diffHash := ComputePayloadHash([]byte(`[{"id":9}]`))
	diffSig := mgr.GenerateSignature(valID, signingSeq, ts, nonce, diffHash)
	err = mgr.VerifyBatchSignature(ctx, valID, signingSeq, seqNos, ts, nonce, diffHash, diffSig)
	if err != ErrReplayDetected {
		t.Fatalf("expected ErrReplayDetected on reused nonce with different payload, got: %v", err)
	}

	// 4. Overlapping sequence in new batch returns ErrReplayDetected
	newNonce := "batch-nonce-002"
	newSig := mgr.GenerateSignature(valID, 104, ts, newNonce, payloadHash)
	err = mgr.VerifyBatchSignature(ctx, valID, 104, []int64{103, 104}, ts, newNonce, payloadHash, newSig)
	if err != ErrReplayDetected {
		t.Fatalf("expected ErrReplayDetected on overlapping sequence, got: %v", err)
	}

	// 5. Invalid signature returns ErrInvalidDeviceSignature
	err = mgr.VerifyBatchSignature(ctx, valID, signingSeq, seqNos, ts, nonce, payloadHash, "invalid-sig")
	if err != ErrInvalidDeviceSignature {
		t.Fatalf("expected ErrInvalidDeviceSignature, got: %v", err)
	}

	// 6. Revoked validator returns ErrDeviceRevoked
	mgr.revokedIDs[valID] = true
	err = mgr.VerifyBatchSignature(ctx, valID, signingSeq, seqNos, ts, nonce, payloadHash, sig)
	if err != ErrDeviceRevoked {
		t.Fatalf("expected ErrDeviceRevoked, got: %v", err)
	}
}

func TestLoadDeviceCredentials_TableDriven(t *testing.T) {
	// Empty path
	_, err := LoadDeviceCredentials("")
	if err == nil {
		t.Fatalf("expected error for empty path")
	}

	// Non-existent file
	_, err = LoadDeviceCredentials("/nonexistent/file/path.json")
	if err == nil {
		t.Fatalf("expected error for non-existent file")
	}

	// Invalid JSON
	tmpFile, err := os.CreateTemp("", "invalid-*.json")
	if err != nil {
		t.Fatalf("temp file error: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	_, _ = tmpFile.WriteString("not json")
	_ = tmpFile.Close()

	_, err = LoadDeviceCredentials(tmpFile.Name())
	if err == nil {
		t.Fatalf("expected error for invalid JSON")
	}

	// Valid credentials
	validFile, err := os.CreateTemp("", "valid-*.json")
	if err != nil {
		t.Fatalf("temp file error: %v", err)
	}
	defer os.Remove(validFile.Name())
	validJSON := `{
		"VAL_01": "0123456789abcdef0123456789abcdef0123456789abcdef",
		"VAL_02": "abcdef0123456789abcdef0123456789abcdef0123456789"
	}`
	_, _ = validFile.WriteString(validJSON)
	_ = validFile.Close()

	creds, err := LoadDeviceCredentials(validFile.Name())
	if err != nil {
		t.Fatalf("expected valid credentials to load, got: %v", err)
	}
	if len(creds) != 2 || creds["VAL_01"] == "" {
		t.Fatalf("credentials improperly loaded: %v", creds)
	}

	// Empty credentials object
	emptyObjFile, _ := os.CreateTemp("", "empty-obj-*.json")
	defer os.Remove(emptyObjFile.Name())
	_, _ = emptyObjFile.WriteString("{}")
	_ = emptyObjFile.Close()
	if _, err := LoadDeviceCredentials(emptyObjFile.Name()); err == nil {
		t.Fatalf("expected error on empty object")
	}

	// Secret too short (< 32 bytes)
	shortFile, _ := os.CreateTemp("", "short-*.json")
	defer os.Remove(shortFile.Name())
	_, _ = shortFile.WriteString(`{"VAL_01": "too-short"}`)
	_ = shortFile.Close()
	if _, err := LoadDeviceCredentials(shortFile.Name()); err == nil {
		t.Fatalf("expected error on secret < 32 bytes")
	}

	// Duplicate secret
	dupFile, _ := os.CreateTemp("", "dup-*.json")
	defer os.Remove(dupFile.Name())
	_, _ = dupFile.WriteString(`{
		"VAL_01": "0123456789abcdef0123456789abcdef0123456789abcdef",
		"VAL_02": "0123456789abcdef0123456789abcdef0123456789abcdef"
	}`)
	_ = dupFile.Close()
	if _, err := LoadDeviceCredentials(dupFile.Name()); err == nil {
		t.Fatalf("expected error on duplicate secret")
	}

	// Empty validator ID
	emptyValFile, _ := os.CreateTemp("", "empty-val-*.json")
	defer os.Remove(emptyValFile.Name())
	_, _ = emptyValFile.WriteString(`{"": "0123456789abcdef0123456789abcdef0123456789abcdef"}`)
	_ = emptyValFile.Close()
	if _, err := LoadDeviceCredentials(emptyValFile.Name()); err == nil {
		t.Fatalf("expected error on empty validator ID")
	}
}
