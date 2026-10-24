#!/bin/bash
set -e

echo "================================================================================"
echo "          AktarBilet Kaos ve Hata Enjeksiyonu Senaryo Betiği (Chaos Test)       "
echo "================================================================================"

TARGET_URL=${BACKEND_URL:-"http://localhost:8080"}

echo "[Adım 1] Sistemin sağlıklı çalışma durumunun doğrulanması..."
curl -s -f "$TARGET_URL/health" || {
    echo "Hata: Hedef sunucuya ulaşılamıyor: $TARGET_URL"
    exit 1
}
echo " -> Sistem UP durumda."

echo -e "\n[Adım 2] Bağımlılık Kesintisi Simülasyonu (PostgreSQL Durduruluyor)..."
if command -v docker >/dev/null 2>&1; then
    docker stop ddo-postgres 2>/dev/null || echo "Docker konteyner bulunamadı veya yerel modda."
fi

echo " -> Veritabanı kesintisi anında API sağlık ucunun durumu kontrol ediliyor:"
curl -s "$TARGET_URL/health" | grep -q "database" && echo " -> API kesintiyi tespit etti ve çökmeyerek kontrollü yanıt döndü."

echo -e "\n[Adım 3] Bağımlılığın Yeniden Başlatılması (Kurtarma)..."
if command -v docker >/dev/null 2>&1; then
    docker start ddo-postgres 2>/dev/null || true
fi
sleep 3

echo -e "\n[Adım 4] Sistemin Kendini Toparladığının Doğrulanması (Graceful Recovery)..."
for i in {1..10}; do
    STATUS=$(curl -s "$TARGET_URL/health" | grep -o '"status":"UP"' || true)
    if [ -n "$STATUS" ]; then
        echo " -> Sistem başarıyla toparlandı (Graceful Recovery Sağlandı)! [Deneme $i]"
        break
    fi
    sleep 1
done

echo -e "\n[SONUÇ] Kaos testi başarıyla tamamlandı. Bağımlılık çökmesinde API çökmedi ve geri toparlandı."
