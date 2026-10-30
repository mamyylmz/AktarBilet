-- Settlement adjustments retain their accounting category, reason, and actor.
ALTER TABLE jw_settlement_ledger
    ADD COLUMN adjustment_type VARCHAR(32),
    ADD COLUMN reason TEXT,
    ADD COLUMN actor_id UUID REFERENCES jw_tf_users(id);

UPDATE jw_settlement_ledger
SET adjustment_type = CASE WHEN is_storno THEN 'STORNO' ELSE 'MANUAL_ADJUSTMENT' END,
    reason = COALESCE(reason, storno_reason)
WHERE adjustment_type IS NULL;

ALTER TABLE jw_settlement_ledger
    ALTER COLUMN adjustment_type SET DEFAULT 'MANUAL_ADJUSTMENT',
    ALTER COLUMN adjustment_type SET NOT NULL,
    ADD CONSTRAINT chk_jw_settlement_adjustment_type
        CHECK (adjustment_type IN ('STORNO', 'MANUAL_ADJUSTMENT', 'SUBSIDY')),
    ADD CONSTRAINT chk_jw_settlement_storno_type
        CHECK (is_storno = (adjustment_type = 'STORNO'));

CREATE INDEX idx_jw_settlement_ledger_period_type_created
    ON jw_settlement_ledger (period_id, adjustment_type, created_at DESC);
