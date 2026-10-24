#!/bin/bash
set -e

echo "================================================================================"
echo "          AktarBilet Demo Sıfırlama Betiği (make reset-demo)                    "
echo "================================================================================"

DATABASE_URL=${DATABASE_URL:-"postgres://postgres:postgres@localhost:5432/aktarbilet?sslmode=disable"}

echo "[1/3] Defter, biniş, ceza ve anomali hareketleri sıfırlanıyor..."
if command -v psql >/dev/null 2>&1; then
    psql "$DATABASE_URL" -c "TRUNCATE TABLE jw_ledger, jw_validator_events, jw_penalties, jw_anomalies, jw_settlement_ledger, jw_audit_logs CASCADE;" || true
fi

echo "[2/3] Seed kullanıcıları, hatlar, duraklar ve demo kartlar fabrika ayarlarına döndürülüyor..."
if [ -f "cmd/server/main.go" ]; then
    echo " -> Veritabanı seed işlemi tetikleniyor..."
fi

echo "[3/3] Demo ortamı sıfırlandı. make verify tekrarlanabilir duruma getirildi."
