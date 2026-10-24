-- AktarBilet Views, Materialized Views and Indexes Migration
-- Designed for 1M events/hour high throughput and fast reporting

-- 1. Kritik Dizinler (EXPLAIN ANALYZE hedefleri)
-- Sorgu 1: Kartın son 90 dakikalık biniş geçmişi ve son durağının anında bulunması
CREATE INDEX IF NOT EXISTS idx_jw_events_card_tapped ON jw_validator_events (card_id, tapped_at DESC);

-- Sorgu 2: Kart bakiye defteri toplamı ve hareket sorgusu
CREATE INDEX IF NOT EXISTS idx_jw_ledger_card_created ON jw_ledger (card_id, created_at DESC);

-- Sorgu 3: Hat bazlı zaman aralıklı biniş ve yoğunluk filtreleme
CREATE INDEX IF NOT EXISTS idx_jw_events_line_tapped ON jw_validator_events (line_id, tapped_at);

-- Sorgu 4: Validatör sequence ve durum filtreleme
CREATE INDEX IF NOT EXISTS idx_jw_events_status ON jw_validator_events (status);

-- Sorgu 5: Anormallik tespitinde kart geçmişi
CREATE INDEX IF NOT EXISTS idx_jw_anomalies_card_time ON jw_anomalies (card_id, detected_at DESC);

-- 2. 1 Milyon Olay/Saat Yükü İçin Özet Sayaç Tablosu (Aggregate Table)
CREATE TABLE IF NOT EXISTS jw_hourly_line_stats (
    line_id VARCHAR(50) NOT NULL REFERENCES jw_lines(id),
    stat_hour TIMESTAMPTZ NOT NULL,
    total_taps BIGINT NOT NULL DEFAULT 0,
    transfer_taps BIGINT NOT NULL DEFAULT 0,
    total_revenue_kurus BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (line_id, stat_hour)
);

-- 3. Hat Bazlı Doluluk Isı Haritası Materialized View (Bonus 1)
CREATE MATERIALIZED VIEW IF NOT EXISTS jw_mv_line_heatmap AS
SELECT 
    e.line_id,
    l.name AS line_name,
    l.mode,
    l.operator,
    COUNT(e.id) AS tap_count,
    ROUND(COUNT(e.id)::NUMERIC / 1000.0, 2) AS occupancy_ratio, -- 0.0 - 1.0 skalası
    CASE 
        WHEN COUNT(e.id) > 1500 THEN 'HIGH'
        WHEN COUNT(e.id) > 500 THEN 'MEDIUM'
        ELSE 'LOW'
    END AS crowd_level,
    MAX(e.tapped_at) AS last_tap_time
FROM jw_validator_events e
JOIN jw_lines l ON e.line_id = l.id
WHERE e.tapped_at >= NOW() - INTERVAL '24 hours' AND e.status = 'PROCESSED'
GROUP BY e.line_id, l.name, l.mode, l.operator;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_heatmap_line ON jw_mv_line_heatmap (line_id);

-- 4. İşletmeci Gelir Paylaşımı Materialized View (F-5 / T-4)
CREATE MATERIALIZED VIEW IF NOT EXISTS jw_mv_operator_revenue AS
SELECT 
    l.operator,
    DATE(e.tapped_at) AS trip_date,
    COUNT(e.id) AS total_trips,
    SUM(e.fare_charged_kurus) AS gross_revenue_kurus,
    SUM(CASE WHEN e.is_transfer THEN 1 ELSE 0 END) AS transfer_trips
FROM jw_validator_events e
JOIN jw_lines l ON e.line_id = l.id
WHERE e.status = 'PROCESSED'
GROUP BY l.operator, DATE(e.tapped_at);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_op_revenue ON jw_mv_operator_revenue (operator, trip_date);
