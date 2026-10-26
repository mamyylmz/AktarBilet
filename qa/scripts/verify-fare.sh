#!/bin/bash
set -u

echo "================================================================================"
echo "          AktarBilet T-1 Ücret Motoru Bağımsız Otomasyon Doğrulaması            "
echo "          (Bölüm 6 Görev 1 — T1-01 .. T1-12 Uçtan Uca Kalite Kapısı)            "
echo "================================================================================"
echo "Tarih: $(date '+%Y-%m-%d %H:%M:%S') | Çalışma Dizini: $(pwd)"
echo ""

FAIL_COUNT=0
TOTAL_TESTS=12

# Sonuçları saklayacağımız dizi
declare -A RESULTS
declare -A NOTES

run_test() {
    local id="$1"
    local name="$2"
    local cmd="$3"
    
    echo -n "[$id] $name çalıştırılıyor... "
    local out
    out=$(eval "$cmd" 2>&1)
    local status=$?
    
    if [ $status -eq 0 ]; then
        echo -e "\033[32mPASS\033[0m"
        RESULTS["$id"]="PASS"
        NOTES["$id"]="Tüm kural ve kısıtlar doğrulandı."
    else
        echo -e "\033[31mFAIL\033[0m"
        RESULTS["$id"]="FAIL"
        FAIL_COUNT=$((FAIL_COUNT + 1))
        # Hata özetini al
        local first_err
        first_err=$(echo "$out" | grep -E "FAIL|Error|expect\(" | head -n 1 | tr -d '\r\n')
        NOTES["$id"]="${first_err:-İhlal tespit edildi}"
    fi
}

echo ">>> AŞAMA 1: Go Bağımsız Oracle & Sınır Testleri (Unit & Boundary)"
run_test "T1-01" "İlk biniş ve kart türleri" "go test -v ./qa/go/... -run TestT1_01"
run_test "T1-02" "90 dk aktarma penceresi sınırları" "go test -v ./qa/go/... -run TestT1_02"
run_test "T1-03" "En fazla iki indirimli aktarma" "go test -v ./qa/go/... -run TestT1_03"
run_test "T1-04" "Aynı hatta ters yön kuralı" "go test -v ./qa/go/... -run TestT1_04"
run_test "T1-05" "Gece tarifesi ve Timezone tutarlılığı" "go test -v ./qa/go/... -run TestT1_05"
run_test "T1-06" "Öğrenci günlük limiti" "go test -v ./qa/go/... -run TestT1_06"
run_test "T1-07" "Aylık abonman tavanı (Fare Capping)" "go test -v ./qa/go/... -run TestT1_07"
run_test "T1-09" "Çoklu kural etkileşimleri" "go test -v ./qa/go/... -run TestT1_09"
run_test "T1-10" "Integer kuruş para hassasiyeti" "go test -v ./qa/go/... -run TestT1_10"

echo ""
echo ">>> AŞAMA 2: Canlı PostgreSQL 16 & API Entegrasyon Testleri"
export QA_FRONTEND_URL=${QA_FRONTEND_URL:-"http://127.0.0.1:13000"}
export QA_API_URL=${QA_API_URL:-"http://127.0.0.1:18080"}
export QA_PROJECT_NAME=${QA_PROJECT_NAME:-"aktarbilet-qa"}

run_test "T1-08" "Sırasız 3 olay uzlaşması (PostgreSQL)" "cd qa && npx playwright test tests/t1_live_system.spec.js -g 'T1-08'"
run_test "T1-11" "Defter bütünlüğü & Eşzamanlı binişler" "cd qa && npx playwright test tests/t1_live_system.spec.js -g 'Concurrency'"
run_test "T1-12" "Mükerrer ve Idempotency koruması" "cd qa && npx playwright test tests/t1_live_system.spec.js -g 'T1-12'"

echo ""
echo "================================================================================"
echo "                      T-1 ÜCRET MOTORU DOĞRULAMA TABLOSU                        "
echo "================================================================================"
printf "%-7s | %-44s | %-8s | %s\n" "KOD" "GÖREV / ÖZELLİK" "DURUM" "NOT / BULGU"
echo "--------------------------------------------------------------------------------"

for id in T1-01 T1-02 T1-03 T1-04 T1-05 T1-06 T1-07 T1-08 T1-09 T1-10 T1-11 T1-12; do
    status="${RESULTS[$id]:-NOT_RUN}"
    note="${NOTES[$id]:-}"
    if [ "$status" == "PASS" ]; then
        status_color="\033[32mPASS\033[0m"
    else
        status_color="\033[31mFAIL\033[0m"
    fi
    printf "%-7s | %-44s | %-17b | %s\n" "$id" "$(eval echo \$name_$id 2>/dev/null || echo "$id Kontrolü")" "$status_color" "$note"
done
echo "================================================================================"

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "\n\033[32m[TEBRİKLER] T-1 Ücret Motoru tüm zorlu gereksinimleri karşılamaktadır (%100 PASS)!\033[0m"
    exit 0
else
    echo -e "\n\033[31m[DİKKAT] T-1 Ücret Motorunda $FAIL_COUNT adet kritik hata tespit edildi!\033[0m"
    echo "Detaylı kök neden raporu: qa/reports/T1_INITIAL_FAILURES.md"
    exit 1
fi
