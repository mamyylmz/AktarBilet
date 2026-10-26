package sync

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDeviceRevoked          = errors.New("validatör cihazı devre dışı bırakılmış veya iptal edilmiş")
	ErrInvalidDeviceSignature = errors.New("geçersiz validatör cihaz imzası veya yetkisiz cihaz")
	ErrReplayDetected         = errors.New("replay saldırısı tespit edildi: geçersiz veya mükerrer sıra numarası ya da nonce")
	ErrTimestampOutOfBounds   = errors.New("cihaz zaman damgası kabul edilebilir zaman penceresi (15 dk) dışında")
	ErrIdempotencyConflict    = errors.New("idempotency anahtarı çakışması: farklı parametrelerle daha önce kullanılmış")
)

type DeviceAuthManager struct {
	mu           sync.RWMutex
	deviceKeys   map[string][]byte
	pool         *pgxpool.Pool
	revokedIDs   map[string]bool
	lastSeqNos   map[string]int64
	seenNonces   map[string]time.Time
	requests     map[string]string
	maxClockSkew time.Duration
}

func NewDeviceAuthManager(deviceSecrets map[string]string, pool *pgxpool.Pool) *DeviceAuthManager {
	keys := make(map[string][]byte, len(deviceSecrets))
	for validatorID, secret := range deviceSecrets {
		if validatorID != "" && len(secret) >= 32 {
			keys[validatorID] = []byte(secret)
		}
	}
	return &DeviceAuthManager{
		deviceKeys:   keys,
		pool:         pool,
		revokedIDs:   make(map[string]bool),
		lastSeqNos:   make(map[string]int64),
		seenNonces:   make(map[string]time.Time),
		requests:     make(map[string]string),
		maxClockSkew: 15 * time.Minute,
	}
}

// ComputePayloadHash calculates SHA-256 of the raw request payload bytes
func ComputePayloadHash(body []byte) string {
	h := sha256.Sum256(body)
	return hex.EncodeToString(h[:])
}

// GenerateSignature produces an HMAC-SHA256 signature for a validator payload
func (m *DeviceAuthManager) GenerateSignature(validatorID string, seqNo int64, timestamp int64, nonce string, payloadHash string) string {
	deviceSecret, ok := m.deviceKeys[validatorID]
	if !ok {
		return ""
	}
	message := fmt.Sprintf("%s:%d:%d:%s:%s", validatorID, seqNo, timestamp, nonce, payloadHash)
	return computeHMAC(deviceSecret, message)
}

// VerifySignature validates device identity, sequence number, timestamp, and payload integrity.
// State mutations (nonce consumption and sequence tracking) occur ONLY after signature is verified.
func (m *DeviceAuthManager) VerifySignature(ctx context.Context, validatorID string, seqNo int64, timestamp int64, nonce string, payloadHash string, signature string) error {
	return m.verifyAndReserve(ctx, validatorID, seqNo, []int64{seqNo}, timestamp, nonce, payloadHash, signature)
}

// VerifyBatchSignature authenticates one signed batch and atomically reserves
// every sequence in it. Reservations, rather than a monotonic-only rule, allow
// legitimately out-of-order offline delivery while rejecting sequence reuse
// with a different signed payload.
func (m *DeviceAuthManager) VerifyBatchSignature(ctx context.Context, validatorID string, signingSeq int64, sequenceNos []int64, timestamp int64, nonce string, payloadHash string, signature string) error {
	return m.verifyAndReserve(ctx, validatorID, signingSeq, sequenceNos, timestamp, nonce, payloadHash, signature)
}

func (m *DeviceAuthManager) verifyAndReserve(ctx context.Context, validatorID string, signingSeq int64, sequenceNos []int64, timestamp int64, nonce string, payloadHash string, signature string) error {
	if validatorID == "" || signingSeq <= 0 || len(sequenceNos) == 0 || timestamp <= 0 || nonce == "" || payloadHash == "" || signature == "" {
		return ErrInvalidDeviceSignature
	}
	seenSequences := make(map[int64]struct{}, len(sequenceNos))
	for _, sequenceNo := range sequenceNos {
		if sequenceNo <= 0 {
			return ErrReplayDetected
		}
		if _, duplicate := seenSequences[sequenceNo]; duplicate {
			return ErrReplayDetected
		}
		seenSequences[sequenceNo] = struct{}{}
	}

	// 1. Timestamp Window Check (Clock Skew)
	now := time.Now().Unix()
	diff := now - timestamp
	if diff < 0 {
		diff = -diff
	}
	if time.Duration(diff)*time.Second > m.maxClockSkew {
		return ErrTimestampOutOfBounds
	}

	message := fmt.Sprintf("%s:%d:%d:%s:%s", validatorID, signingSeq, timestamp, nonce, payloadHash)

	if m.pool != nil {
		// PostgreSQL-backed verification (durably survives server restarts)
		var secretKey string
		var isRevoked bool

		err := m.pool.QueryRow(ctx, `
			SELECT secret_key, is_revoked
			FROM jw_validator_credentials
			WHERE validator_id = $1
		`, validatorID).Scan(&secretKey, &isRevoked)

		if err != nil {
			return ErrInvalidDeviceSignature
		}

		if isRevoked {
			return ErrDeviceRevoked
		}

		// Kriptografik imza kontrolü (yalnızca cihaza özel atanmış gizli anahtar ile)
		expectedSig := computeHMAC([]byte(secretKey), message)
		if !hmac.Equal([]byte(expectedSig), []byte(signature)) {
			return ErrInvalidDeviceSignature
		}

		// İmza geçerli. Şimdi atomik olarak PostgreSQL üzerinde sequence ve nonce kontrolü yap ve tüket.
		tx, err := m.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("transaction başlatılamadı: %w", err)
		}
		defer tx.Rollback(ctx)

		var lockedRevoked bool
		err = tx.QueryRow(ctx, `
			SELECT is_revoked FROM jw_validator_credentials
			WHERE validator_id = $1 FOR UPDATE
		`, validatorID).Scan(&lockedRevoked)
		if err != nil {
			return ErrInvalidDeviceSignature
		}
		if lockedRevoked {
			return ErrDeviceRevoked
		}

		allExisting := true
		for _, sequenceNo := range sequenceNos {
			var storedHash, storedNonce string
			err = tx.QueryRow(ctx, `
				SELECT request_hash, device_nonce FROM jw_device_requests
				WHERE validator_id = $1 AND sequence_no = $2
			`, validatorID, sequenceNo).Scan(&storedHash, &storedNonce)
			if err == nil {
				if !hmac.Equal([]byte(storedHash), []byte(payloadHash)) || !hmac.Equal([]byte(storedNonce), []byte(nonce)) {
					return ErrReplayDetected
				}
				continue
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("device request reservation cannot be read: %w", err)
			}
			allExisting = false
		}
		if allExisting {
			return nil
		}

		var nonceExists bool
		_ = tx.QueryRow(ctx, `SELECT true FROM jw_device_nonces WHERE nonce = $1`, nonce).Scan(&nonceExists)
		if nonceExists {
			return ErrReplayDetected
		}
		if _, err = tx.Exec(ctx, `
			INSERT INTO jw_device_nonces (nonce, validator_id, used_at)
			VALUES ($1, $2, NOW())
		`, nonce, validatorID); err != nil {
			return ErrReplayDetected
		}
		for _, sequenceNo := range sequenceNos {
			if _, err = tx.Exec(ctx, `
				INSERT INTO jw_device_requests (validator_id, sequence_no, request_hash, device_nonce)
				VALUES ($1, $2, $3, $4)
			`, validatorID, sequenceNo, payloadHash, nonce); err != nil {
				return ErrReplayDetected
			}
		}

		_, err = tx.Exec(ctx, `
			UPDATE jw_validator_credentials
			SET last_sequence_no = GREATEST(last_sequence_no, $1), updated_at = NOW()
			WHERE validator_id = $2
		`, signingSeq, validatorID)
		if err != nil {
			return fmt.Errorf("sequence güncellenemedi: %w", err)
		}

		return tx.Commit(ctx)
	}

	// In-memory fallback (unit testler veya pool'suz durumlar)
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.revokedIDs[validatorID] {
		return ErrDeviceRevoked
	}

	deviceSecret, ok := m.deviceKeys[validatorID]
	if !ok {
		return ErrInvalidDeviceSignature
	}
	expectedSig := computeHMAC(deviceSecret, message)
	if !hmac.Equal([]byte(expectedSig), []byte(signature)) {
		return ErrInvalidDeviceSignature
	}

	fingerprint := payloadHash + "\x00" + nonce
	allExisting := true
	for _, sequenceNo := range sequenceNos {
		requestKey := fmt.Sprintf("%s:%d", validatorID, sequenceNo)
		if existing, ok := m.requests[requestKey]; ok {
			if !hmac.Equal([]byte(existing), []byte(fingerprint)) {
				return ErrReplayDetected
			}
		} else {
			allExisting = false
		}
	}
	if allExisting {
		return nil
	}

	if _, seen := m.seenNonces[nonce]; seen {
		return ErrReplayDetected
	}
	for _, sequenceNo := range sequenceNos {
		requestKey := fmt.Sprintf("%s:%d", validatorID, sequenceNo)
		if _, exists := m.requests[requestKey]; exists {
			return ErrReplayDetected
		}
		m.requests[requestKey] = fingerprint
	}
	m.seenNonces[nonce] = time.Now()
	if signingSeq > m.lastSeqNos[validatorID] {
		m.lastSeqNos[validatorID] = signingSeq
	}
	return nil
}

func computeHMAC(key []byte, message string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}
