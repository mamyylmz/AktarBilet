-- Migration 010: jw_settlement_ledger idempotency_key ve penalty reason güncelleme
-- Bu migration önceki migrasyonlara bağımlıdır.

-- 1. jw_settlement_ledger tablosuna idempotency_key kolonu ekle (storno tekrar koruması)
ALTER TABLE jw_settlement_ledger
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);

-- Mevcut kayıtlar için benzersiz idempotency_key üret (eğer NULL ise)
UPDATE jw_settlement_ledger
SET idempotency_key = 'legacy-' || id::text
WHERE idempotency_key IS NULL;

-- Opsiyonel benzersiz index (NULL değerlere izin verir; unique null-safe)
CREATE UNIQUE INDEX IF NOT EXISTS uq_jw_settlement_ledger_idem_key
    ON jw_settlement_ledger (idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- 2. jw_penalties tablosundaki reason CHECK constraint'i güncelle
-- 'baskasinin_karti' ve 'sahte_belge' ek değerlerini destekle
ALTER TABLE jw_penalties
    DROP CONSTRAINT IF EXISTS jw_penalties_reason_check;

ALTER TABLE jw_penalties
    ADD CONSTRAINT jw_penalties_reason_check
    CHECK (reason IN (
        'kartsiz_binis',
        'usulsuz_kart_kullanimi',
        'sahte_kart',
        'baskasinin_karti',
        'sahte_belge',
        'diger'
    ));

-- 3. jw_settlement_periods tablosuna reopen_at, reopen_by, last_action_reason kolonu ekle
ALTER TABLE jw_settlement_periods
    ADD COLUMN IF NOT EXISTS reopen_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reopen_by UUID REFERENCES jw_tf_users(id),
    ADD COLUMN IF NOT EXISTS last_action_reason TEXT;
