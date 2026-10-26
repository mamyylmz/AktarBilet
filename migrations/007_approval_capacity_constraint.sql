-- Correct the approval constraint to match the persisted approval DTO.
-- Migration 006 may already be applied, so this correction is forward-only.

ALTER TABLE jw_optimization_approvals
    DROP CONSTRAINT IF EXISTS ck_jw_optimization_approvals_capacity;

ALTER TABLE jw_optimization_approvals
    ADD CONSTRAINT ck_jw_optimization_approvals_capacity
    CHECK (fleet_capacity > 0) NOT VALID;
