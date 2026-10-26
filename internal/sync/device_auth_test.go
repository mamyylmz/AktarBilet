package sync

import (
	"context"
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
