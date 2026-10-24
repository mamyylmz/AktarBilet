-- AktarBilet Initial Schema Migration
-- Standard: All tables must have "jw_" prefix.
-- Standard: User table must be named "jw_tf_users".
-- Standard: All currency values are stored as integers (kuruş, 1 TL = 100 kuruş). No float!

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Kullanıcılar ve Roller
CREATE TABLE IF NOT EXISTS jw_tf_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('yolcu', 'kontrolor', 'hat_yoneticisi', 'muhasebe', 'operasyon', 'admin')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Ulaşım Hatları (Otobüs, Tramvay, Vapur)
CREATE TABLE IF NOT EXISTS jw_lines (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    mode VARCHAR(50) NOT NULL CHECK (mode IN ('bus', 'tram', 'ferry')),
    operator VARCHAR(100) NOT NULL, -- 'belediye' veya 'ozel_halk'
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Duraklar / İskeleler
CREATE TABLE IF NOT EXISTS jw_stations (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    mode VARCHAR(50) NOT NULL CHECK (mode IN ('bus', 'tram', 'ferry')),
    latitude NUMERIC(10, 7) NOT NULL,
    longitude NUMERIC(10, 7) NOT NULL,
    zone_id INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Hat-Durak Sıralaması (Güzergah)
CREATE TABLE IF NOT EXISTS jw_line_stations (
    line_id VARCHAR(50) NOT NULL REFERENCES jw_lines(id) ON DELETE CASCADE,
    station_id VARCHAR(50) NOT NULL REFERENCES jw_stations(id) ON DELETE CASCADE,
    sequence_no INT NOT NULL,
    direction INT NOT NULL DEFAULT 1 CHECK (direction IN (1, 2)), -- 1: Gidiş, 2: Dönüş
    distance_from_start_meters INT NOT NULL DEFAULT 0,
    PRIMARY KEY (line_id, station_id, direction)
);

-- 5. Kart Tipleri ve Kartlar
CREATE TABLE IF NOT EXISTS jw_cards (
    id VARCHAR(64) PRIMARY KEY, -- Kart UID veya Barkod
    user_id UUID REFERENCES jw_tf_users(id) ON DELETE SET NULL,
    card_type VARCHAR(50) NOT NULL CHECK (card_type IN ('tam', 'ogrenci', 'yasli', 'ucretsiz')),
    balance_kurus BIGINT NOT NULL DEFAULT 0, -- Kuruş cinsinden
    is_blacklisted BOOLEAN NOT NULL DEFAULT FALSE,
    blacklist_reason TEXT,
    daily_taps_count INT NOT NULL DEFAULT 0,
    last_daily_reset_date DATE NOT NULL DEFAULT CURRENT_DATE,
    monthly_capping_spent_kurus BIGINT NOT NULL DEFAULT 0,
    last_monthly_reset_date DATE NOT NULL DEFAULT CURRENT_DATE,
    last_tap_at TIMESTAMPTZ,
    last_station_id VARCHAR(50) REFERENCES jw_stations(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. Kart Bakiye Defteri (Append-Only Ledger)
-- Bakiye = SUM(amount_kurus). Her hareket bu tabloya eklenir.
CREATE TABLE IF NOT EXISTS jw_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id VARCHAR(64) NOT NULL REFERENCES jw_cards(id) ON DELETE RESTRICT,
    entry_type VARCHAR(50) NOT NULL CHECK (entry_type IN ('dolum', 'binis', 'iade', 'ceza', 'storno_duzeltme')),
    amount_kurus BIGINT NOT NULL, -- Pozitif: dolum/iade, Negatif: biniş/ceza
    balance_after_kurus BIGINT NOT NULL,
    reference_id VARCHAR(255), -- Validatör event ID veya ödeme referansı
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 7. Tarifeler (Kuruş cinsinden tamsayı)
CREATE TABLE IF NOT EXISTS jw_tariffs (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    mode VARCHAR(50) NOT NULL,
    base_fare_kurus BIGINT NOT NULL,
    student_fare_kurus BIGINT NOT NULL,
    elderly_fare_kurus BIGINT NOT NULL,
    transfer1_discount_pct INT NOT NULL DEFAULT 50, -- %50 indirim
    transfer2_discount_pct INT NOT NULL DEFAULT 75, -- %75 indirim
    night_surcharge_pct INT NOT NULL DEFAULT 50,    -- %50 gece zammı (00:00 - 05:59)
    student_daily_limit INT NOT NULL DEFAULT 10,     -- Günde maks 10 indirimli biniş
    monthly_capping_limit_kurus BIGINT NOT NULL DEFAULT 150000, -- 1500 TL tavan
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 8. Validatörler
CREATE TABLE IF NOT EXISTS jw_validators (
    id VARCHAR(64) PRIMARY KEY,
    line_id VARCHAR(50) REFERENCES jw_lines(id),
    station_id VARCHAR(50) REFERENCES jw_stations(id),
    vehicle_plate VARCHAR(50),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_heartbeat_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 9. Validatör Olayları (Turnike Okumaları)
CREATE TABLE IF NOT EXISTS jw_validator_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    validator_id VARCHAR(64) NOT NULL REFERENCES jw_validators(id),
    sequence_no BIGINT NOT NULL,
    card_id VARCHAR(64) NOT NULL REFERENCES jw_cards(id),
    line_id VARCHAR(50) NOT NULL REFERENCES jw_lines(id),
    station_id VARCHAR(50) NOT NULL REFERENCES jw_stations(id),
    direction INT NOT NULL DEFAULT 1,
    tapped_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fare_charged_kurus BIGINT NOT NULL,
    is_transfer BOOLEAN NOT NULL DEFAULT FALSE,
    transfer_level INT NOT NULL DEFAULT 0, -- 0: İlk biniş, 1: 1. Aktarma, 2: 2. Aktarma
    status VARCHAR(50) NOT NULL DEFAULT 'PROCESSED' CHECK (status IN ('PROCESSED', 'REJECTED', 'NEGATIVE_TOLERANCE', 'ANOMALY_FLAGGED')),
    rejection_reason TEXT,
    CONSTRAINT uq_validator_sequence UNIQUE (validator_id, sequence_no)
);

-- 10. Gelir Paylaşımı (İşletmeci Dağılımı ve Dönem Kapanışı)
CREATE TABLE IF NOT EXISTS jw_settlement_periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_name VARCHAR(100) NOT NULL UNIQUE,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    is_closed BOOLEAN NOT NULL DEFAULT FALSE,
    closed_at TIMESTAMPTZ,
    closed_by UUID REFERENCES jw_tf_users(id),
    total_revenue_kurus BIGINT NOT NULL DEFAULT 0,
    belediye_share_kurus BIGINT NOT NULL DEFAULT 0,
    ozel_halk_share_kurus BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS jw_settlement_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_id UUID NOT NULL REFERENCES jw_settlement_periods(id),
    event_id UUID REFERENCES jw_validator_events(id),
    operator VARCHAR(100) NOT NULL,
    amount_kurus BIGINT NOT NULL,
    is_storno BOOLEAN NOT NULL DEFAULT FALSE,
    storno_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 11. Kontrol ve Ceza Kayıtları
CREATE TABLE IF NOT EXISTS jw_penalties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inspector_id UUID NOT NULL REFERENCES jw_tf_users(id),
    card_id VARCHAR(64) NOT NULL REFERENCES jw_cards(id),
    line_id VARCHAR(50) REFERENCES jw_lines(id),
    station_id VARCHAR(50) REFERENCES jw_stations(id),
    penalty_amount_kurus BIGINT NOT NULL,
    reason VARCHAR(100) NOT NULL CHECK (reason IN ('kartsiz_binis', 'usulsuz_kart_kullanimi', 'sahte_kart', 'diger')),
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT
);

-- 12. Güvenlik ve Anormallik Kayıtları (İmkansız Seyahat & Klon Kart)
CREATE TABLE IF NOT EXISTS jw_anomalies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id VARCHAR(64) NOT NULL REFERENCES jw_cards(id),
    event_id UUID REFERENCES jw_validator_events(id),
    anomaly_type VARCHAR(100) NOT NULL CHECK (anomaly_type IN ('IMPOSSIBLE_TRAVEL_SPEED', 'CLONE_CARD_DETECTED', 'CONCURRENT_TAP', 'BRUTE_FORCE_VALIDATOR')),
    description TEXT NOT NULL,
    calculated_speed_kmh NUMERIC(8, 2),
    time_delta_seconds INT,
    distance_meters INT,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 13. Append-Only Denetim Kaydı (Audit Log)
CREATE TABLE IF NOT EXISTS jw_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES jw_tf_users(id),
    action VARCHAR(100) NOT NULL,
    table_name VARCHAR(100) NOT NULL,
    record_id VARCHAR(255) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    ip_address VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
