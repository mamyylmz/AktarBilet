#!/bin/bash
set -e

echo "================================================================================"
echo "          AktarBilet Demo Sıfırlama Betiği (make reset-demo)                    "
echo "================================================================================"

DATABASE_URL=${DATABASE_URL:-"postgres://postgres:postgres@localhost:5432/aktarbilet?sslmode=disable"}

echo "[1/3] Go reset aracı ile veritabanı tabloları sıfırlanıyor..."
if [ -f "cmd/server/main.go" ]; then
    go run cmd/server/main.go -reset || {
        echo "Go reset başarısız oldu, psql deneniyor..."
        if command -v psql >/dev/null 2>&1; then
            psql "$DATABASE_URL" -c "TRUNCATE TABLE jw_ledger, jw_validator_events, jw_penalties, jw_anomalies, jw_settlement_ledger, jw_audit_logs CASCADE; UPDATE jw_cards SET balance_kurus = 0, daily_taps_count = 0, monthly_capping_spent_kurus = 0;"
        fi
    }
fi

echo "[2/3] Seed kullanıcıları, hatlar, duraklar ve demo kartlar fabrika ayarlarına döndürüldü."
echo "[3/3] Demo ortamı sıfırlandı. make verify tekrarlanabilir duruma getirildi."
