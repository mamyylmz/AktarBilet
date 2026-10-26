-- AktarBilet 005 Device Authentication & Optimization Approvals Migration

-- 1. Validator Device Credentials Table (PostgreSQL persistent revocation, sequence & secret storage)
CREATE TABLE IF NOT EXISTS jw_validator_credentials (
    validator_id VARCHAR(64) PRIMARY KEY REFERENCES jw_validators(id),
    secret_key VARCHAR(255) NOT NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    last_sequence_no BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Device Nonces Table (PostgreSQL persistent replay prevention across restarts)
CREATE TABLE IF NOT EXISTS jw_device_nonces (
    nonce VARCHAR(128) PRIMARY KEY,
    validator_id VARCHAR(64) NOT NULL REFERENCES jw_validators(id),
    used_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Optimization Approvals Table (Hat Yöneticisi audit trail)
CREATE TABLE IF NOT EXISTS jw_optimization_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    line_id VARCHAR(50) NOT NULL REFERENCES jw_lines(id),
    scenario_name VARCHAR(100) DEFAULT '',
    approved_by UUID NOT NULL REFERENCES jw_tf_users(id),
    current_headway_minutes INT NOT NULL DEFAULT 0,
    target_headway_minutes INT NOT NULL DEFAULT 0,
    fleet_capacity_per_vehicle INT DEFAULT 0,
    fleet_capacity INT DEFAULT 0,
    proposed_vehicles INT NOT NULL DEFAULT 0,
    vehicle_delta INT NOT NULL DEFAULT 0,
    delta_cost_kurus BIGINT NOT NULL DEFAULT 0,
    projected_net_margin_kurus BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'APPROVED',
    notes TEXT,
    approved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ensure backwards-compatibility for existing environments
ALTER TABLE jw_optimization_approvals ADD COLUMN IF NOT EXISTS fleet_capacity INT DEFAULT 0;
ALTER TABLE jw_optimization_approvals ADD COLUMN IF NOT EXISTS vehicle_delta INT DEFAULT 0;
ALTER TABLE jw_optimization_approvals ADD COLUMN IF NOT EXISTS projected_net_margin_kurus BIGINT DEFAULT 0;
ALTER TABLE jw_optimization_approvals ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'APPROVED';
ALTER TABLE jw_optimization_approvals ADD COLUMN IF NOT EXISTS notes TEXT;
ALTER TABLE jw_optimization_approvals ADD COLUMN IF NOT EXISTS approved_at TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE jw_optimization_approvals ALTER COLUMN scenario_name DROP NOT NULL;
