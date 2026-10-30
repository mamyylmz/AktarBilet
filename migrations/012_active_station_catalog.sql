-- Preserve historical station references while allowing the API to expose only
-- stations in the current verified İzmir catalog.
ALTER TABLE jw_stations
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_jw_stations_active_id
    ON jw_stations (id)
    WHERE is_active = TRUE;
