-- Migration 009: Fix batch device nonce constraint
-- Drop the erroneous unique constraint on device_nonce in jw_device_requests.
-- Nonce uniqueness at request level is handled atomically by jw_device_nonces.
-- Multi-event batches share a single request nonce across multiple sequences.

ALTER TABLE jw_device_requests DROP CONSTRAINT IF EXISTS uq_jw_device_requests_nonce;

