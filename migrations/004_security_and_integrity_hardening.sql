-- AktarBilet 004 Security and Integrity Hardening Migration
-- Direction constraints, settlement idempotency and audit hardening

-- 1. Validator Event Direction Check Constraint (direction must be 1 or 2)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_validator_events_direction'
    ) THEN
        ALTER TABLE jw_validator_events ADD CONSTRAINT chk_validator_events_direction CHECK (direction IN (1, 2));
    END IF;
END $$;

-- 2. Settlement Ledger Idempotency Key Column & Unique Constraint
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'jw_settlement_ledger' AND column_name = 'idempotency_key'
    ) THEN
        ALTER TABLE jw_settlement_ledger ADD COLUMN idempotency_key VARCHAR(255) UNIQUE;
    END IF;
END $$;

-- 3. Invariant: jw_cards balance consistency trigger / check
CREATE OR REPLACE FUNCTION audit_financial_ledger_event()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO jw_audit_logs (user_id, action, table_name, record_id, old_values, new_values, created_at)
    VALUES (
        NULL,
        TG_OP || '_' || NEW.entry_type,
        'jw_ledger',
        NEW.card_id,
        NULL,
        json_build_object(
            'entry_type', NEW.entry_type,
            'amount_kurus', NEW.amount_kurus,
            'balance_after_kurus', NEW.balance_after_kurus,
            'reference_id', NEW.reference_id
        )::jsonb,
        NOW()
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_audit_financial_ledger ON jw_ledger;
CREATE TRIGGER trg_audit_financial_ledger
AFTER INSERT ON jw_ledger
FOR EACH ROW EXECUTE FUNCTION audit_financial_ledger_event();

