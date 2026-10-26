-- Durable device-request reservations and optimization approval integrity.

CREATE TABLE IF NOT EXISTS jw_device_requests (
    validator_id VARCHAR(64) NOT NULL REFERENCES jw_validators(id) ON DELETE RESTRICT,
    sequence_no BIGINT NOT NULL CHECK (sequence_no > 0),
    request_hash CHAR(64) NOT NULL,
    device_nonce VARCHAR(128) NOT NULL,
    authenticated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (validator_id, sequence_no),
    CONSTRAINT uq_jw_device_requests_nonce UNIQUE (device_nonce)
);

ALTER TABLE jw_optimization_approvals
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);

CREATE UNIQUE INDEX IF NOT EXISTS uq_jw_optimization_approvals_idempotency
    ON jw_optimization_approvals (idempotency_key)
    WHERE idempotency_key IS NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ck_jw_optimization_approvals_headways'
    ) THEN
        ALTER TABLE jw_optimization_approvals
            ADD CONSTRAINT ck_jw_optimization_approvals_headways
            CHECK (current_headway_minutes > 0 AND target_headway_minutes > 0) NOT VALID;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ck_jw_optimization_approvals_capacity'
    ) THEN
        ALTER TABLE jw_optimization_approvals
            ADD CONSTRAINT ck_jw_optimization_approvals_capacity
            CHECK (fleet_capacity_per_vehicle > 0 AND fleet_capacity >= 0 AND proposed_vehicles > 0) NOT VALID;
    END IF;
END
$$;
