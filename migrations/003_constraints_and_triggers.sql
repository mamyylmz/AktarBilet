-- AktarBilet Constraints and Append-Only Triggers Migration
-- Ensures financial data integrity, ledger immutability and valid range constraints

-- 1. Append-Only Trigger Function
CREATE OR REPLACE FUNCTION prevent_ledger_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'jw_ledger is append-only: UPDATE and DELETE are prohibited' USING ERRCODE = '55000';
END;
$$ LANGUAGE plpgsql;

-- 2. Trigger on jw_ledger
DROP TRIGGER IF EXISTS trg_prevent_ledger_mutation ON jw_ledger;
CREATE TRIGGER trg_prevent_ledger_mutation
BEFORE UPDATE OR DELETE ON jw_ledger
FOR EACH ROW EXECUTE FUNCTION prevent_ledger_mutation();

-- 3. Trigger on jw_audit_logs
DROP TRIGGER IF EXISTS trg_prevent_audit_mutation ON jw_audit_logs;
CREATE TRIGGER trg_prevent_audit_mutation
BEFORE UPDATE OR DELETE ON jw_audit_logs
FOR EACH ROW EXECUTE FUNCTION prevent_ledger_mutation();

-- 4. Unique Index on Reference ID in jw_ledger (for non-null reference IDs)
CREATE UNIQUE INDEX IF NOT EXISTS idx_jw_ledger_reference_id ON jw_ledger (reference_id) WHERE reference_id IS NOT NULL;

-- 5. Value and Range Constraints
DO $$ 
BEGIN
    -- Tariffs constraints
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_tariffs_base_fare') THEN
        ALTER TABLE jw_tariffs ADD CONSTRAINT chk_tariffs_base_fare CHECK (base_fare_kurus >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_tariffs_student_fare') THEN
        ALTER TABLE jw_tariffs ADD CONSTRAINT chk_tariffs_student_fare CHECK (student_fare_kurus >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_tariffs_elderly_fare') THEN
        ALTER TABLE jw_tariffs ADD CONSTRAINT chk_tariffs_elderly_fare CHECK (elderly_fare_kurus >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_tariffs_transfer1') THEN
        ALTER TABLE jw_tariffs ADD CONSTRAINT chk_tariffs_transfer1 CHECK (transfer1_discount_pct >= 0 AND transfer1_discount_pct <= 100);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_tariffs_transfer2') THEN
        ALTER TABLE jw_tariffs ADD CONSTRAINT chk_tariffs_transfer2 CHECK (transfer2_discount_pct >= 0 AND transfer2_discount_pct <= 100);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_tariffs_night') THEN
        ALTER TABLE jw_tariffs ADD CONSTRAINT chk_tariffs_night CHECK (night_surcharge_pct >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_tariffs_student_daily') THEN
        ALTER TABLE jw_tariffs ADD CONSTRAINT chk_tariffs_student_daily CHECK (student_daily_limit >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_tariffs_monthly_capping') THEN
        ALTER TABLE jw_tariffs ADD CONSTRAINT chk_tariffs_monthly_capping CHECK (monthly_capping_limit_kurus >= 0);
    END IF;

    -- Penalties constraint
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_penalties_amount') THEN
        ALTER TABLE jw_penalties ADD CONSTRAINT chk_penalties_amount CHECK (penalty_amount_kurus > 0);
    END IF;

    -- Cards constraints
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_cards_daily_taps') THEN
        ALTER TABLE jw_cards ADD CONSTRAINT chk_cards_daily_taps CHECK (daily_taps_count >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_cards_monthly_capping') THEN
        ALTER TABLE jw_cards ADD CONSTRAINT chk_cards_monthly_capping CHECK (monthly_capping_spent_kurus >= 0);
    END IF;
END $$;
