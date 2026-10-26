-- Migration 008: Multi-Role Support and User-Role Mapping Table
-- Invariant Check: Table name starts with mandatory prefix "jw_"

CREATE TABLE IF NOT EXISTS jw_user_roles (
    user_id UUID NOT NULL REFERENCES jw_tf_users(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role)
);

-- Seed current roles from jw_tf_users
INSERT INTO jw_user_roles (user_id, role)
SELECT id, role FROM jw_tf_users
ON CONFLICT (user_id, role) DO NOTHING;

-- Multi-role grant: Seed canary user seed.sedir@ddo-case.invalid holds both admin and yolcu roles
INSERT INTO jw_user_roles (user_id, role)
SELECT id, 'yolcu' FROM jw_tf_users WHERE email = 'seed.sedir@ddo-case.invalid'
ON CONFLICT (user_id, role) DO NOTHING;

INSERT INTO jw_user_roles (user_id, role)
SELECT id, 'admin' FROM jw_tf_users WHERE email = 'seed.sedir@ddo-case.invalid'
ON CONFLICT (user_id, role) DO NOTHING;

