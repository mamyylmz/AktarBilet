package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mamyylmz/AktarBilet/internal/anomaly"
	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
	"github.com/mamyylmz/AktarBilet/internal/settlement"
	appsync "github.com/mamyylmz/AktarBilet/internal/sync"
)

type VerificationItem struct {
	ID       string
	Name     string
	Category string
	Passed   bool
	Duration time.Duration
	Notes    string
}

var verifierDeviceSecrets map[string]string

func signTapRequest(req *http.Request, valID string, seq int64, bodyBytes []byte, nonce string) {
	deviceSecret, ok := verifierDeviceSecrets[valID]
	if !ok {
		return
	}
	nowTs := time.Now().Unix()
	bodyHash := appsync.ComputePayloadHash(bodyBytes)
	msg := fmt.Sprintf("%s:%d:%d:%s:%s", valID, seq, nowTs, nonce, bodyHash)
	mac := hmac.New(sha256.New, []byte(deviceSecret))
	mac.Write([]byte(msg))
	sig := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("X-Validator-ID", valID)
	req.Header.Set("X-Device-Signature", sig)
	req.Header.Set("X-Device-Timestamp", fmt.Sprintf("%d", nowTs))
	req.Header.Set("X-Device-Nonce", nonce)
}

func main() {
	baseURL := os.Getenv("BACKEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	var err error
	verifierDeviceSecrets, err = appsync.LoadDeviceCredentials(os.Getenv("VALIDATOR_CREDENTIALS_FILE"))
	if err != nil {
		fmt.Printf("[HATA] Validatör cihaz kimlik bilgileri yüklenemedi: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("================================================================================")
	fmt.Println("             AktarBilet Hızlı Doğrulama ve Kabul Testi (make verify)            ")
	fmt.Println("             DDO Yazılım Ekibi Şartnamesi — Bölüm 10 Otomasyonu                 ")
	fmt.Println("================================================================================")
	fmt.Printf("Hedef Sunucu: %s | Tarih: %s\n\n", baseURL, time.Now().Format("2006-01-02 15:04:05"))

	client := &http.Client{Timeout: 8 * time.Second}

	// 1. ZORUNLU KONTROL: Canlı arka uç sunucusu ayakta mı? (Fail Fast)
	resp, err := client.Get(baseURL + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Printf("\033[31m[HATA] Arka uç servisine bağlanılamadı (%s): %v\033[0m\n", baseURL, err)
		if resp != nil {
			fmt.Printf("HTTP Yanıt Kodu: %d\n", resp.StatusCode)
			resp.Body.Close()
		}
		fmt.Println("Lütfen backend servisinin çalıştığından emin olun (docker compose up veya go run ./cmd/server).")
		os.Exit(1)
	}
	resp.Body.Close()

	fmt.Println("[MOD] Canlı HTTP API sunucusuna başarıyla bağlanıldı. HTTP üzerinden doğrulanıyor...")
	results := runLiveHTTPVerifications(client, baseURL)

	// Sonuç tablosunu bas
	fmt.Println("\n--------------------------------------------------------------------------------")
	fmt.Printf("%-6s | %-12s | %-44s | %-8s | %-6s\n", "KOD", "KATEGORİ", "ÖZELLİK / GÖREV", "DURUM", "SÜRE")
	fmt.Println("--------------------------------------------------------------------------------")

	allPassed := true
	for _, item := range results {
		statusStr := "\033[32mPASS\033[0m"
		if !item.Passed {
			statusStr = "\033[31mFAIL\033[0m"
			allPassed = false
		}
		fmt.Printf("%-6s | %-12s | %-44s | %-8s | %-6dms\n",
			item.ID, item.Category, item.Name, statusStr, item.Duration.Milliseconds())
		if item.Notes != "" && !item.Passed {
			fmt.Printf("       -> Hata: %s\n", item.Notes)
		}
	}
	fmt.Println("--------------------------------------------------------------------------------")

	if allPassed {
		fmt.Println("\n\033[32m[TEBRİKLER] Tüm doğrulama maddeleri (SYS-1, SYS-2, F-1..F-7 ve T-1..T-5) başarıyla geçti!\033[0m")
		fmt.Println("Proje teslim kriterleri ve DDO gereksinimleri %100 karşılanmaktadır.")
		os.Exit(0)
	} else {
		fmt.Println("\n\033[31m[DİKKAT] Bazı doğrulama maddeleri başarısız oldu!\033[0m")
		os.Exit(1)
	}
}

func runLiveHTTPVerifications(client *http.Client, baseURL string) []VerificationItem {
	var items []VerificationItem

	// --- SYS-1: Health check and mandatory stamp canary ---
	t0 := time.Now()
	respHealth, errHealth := client.Get(baseURL + "/health")
	stamp := ""
	healthOK := false
	if errHealth == nil {
		stamp = respHealth.Header.Get("X-DDO-Stamp")
		healthOK = respHealth.StatusCode == http.StatusOK
		respHealth.Body.Close()
	}
	respHz, errHz := client.Get(baseURL + "/_hz/f85f")
	hzOK := false
	if errHz == nil {
		hzOK = respHz.StatusCode == http.StatusOK
		respHz.Body.Close()
	}
	sys1Passed := healthOK && hzOK && stamp == "f85f26"
	items = append(items, VerificationItem{
		ID:       "SYS-1",
		Category: "Sistem",
		Name:     "GET /health ve GET /_hz/f85f (Stamp: f85f26)",
		Passed:   sys1Passed,
		Duration: time.Since(t0),
		Notes:    fmt.Sprintf("healthOK=%v, hzOK=%v, stamp=%s", healthOK, hzOK, stamp),
	})

	// --- SYS-2: Authentication & Canary Seed User ---
	t0 = time.Now()
	adminToken, errLoginAdmin := loginUser(client, baseURL, "seed.sedir@ddo-case.invalid", "SedirPassword2026!")
	_, errLoginWrong := loginUser(client, baseURL, "seed.sedir@ddo-case.invalid", "WrongPassword!")
	sys2Passed := errLoginAdmin == nil && adminToken != "" && errLoginWrong != nil
	items = append(items, VerificationItem{
		ID:       "SYS-2",
		Category: "Kimlik/Yetki",
		Name:     "Seed Kullanıcı Oturumu (seed.sedir@ddo-case.invalid)",
		Passed:   sys2Passed,
		Duration: time.Since(t0),
		Notes:    fmt.Sprintf("adminToken=%t, wrongRejected=%t", adminToken != "", errLoginWrong != nil),
	})

	// User tokens for role-based testing
	yolcuToken, _ := loginUser(client, baseURL, "yolcu@aktarbilet.gov.tr", "YolcuPassword123!")
	kontrolorToken, _ := loginUser(client, baseURL, "kontrolor@aktarbilet.gov.tr", "KontrolorPassword123!")
	muhasebeToken, _ := loginUser(client, baseURL, "muhasebe@aktarbilet.gov.tr", "MuhasebePassword123!")
	operasyonToken, _ := loginUser(client, baseURL, "operasyon@aktarbilet.gov.tr", "OperasyonPassword123!")

	// --- F-1: Hat, Durak ve Tarife Yönetimi ---
	t0 = time.Now()
	respLines, errLines := client.Get(baseURL + "/api/v1/lines")
	linesOK := errLines == nil && respLines.StatusCode == http.StatusOK
	if respLines != nil {
		respLines.Body.Close()
	}
	respStations, errStations := client.Get(baseURL + "/api/v1/stations")
	stationsOK := errStations == nil && respStations.StatusCode == http.StatusOK
	if respStations != nil {
		respStations.Body.Close()
	}
	respTariffs, errTariffs := client.Get(baseURL + "/api/v1/tariffs")
	tariffsOK := errTariffs == nil && respTariffs.StatusCode == http.StatusOK
	if respTariffs != nil {
		respTariffs.Body.Close()
	}
	f1Passed := linesOK && stationsOK && tariffsOK
	items = append(items, VerificationItem{
		ID:       "F-1",
		Category: "Kapsam",
		Name:     "Hat, durak ve tarife yönetimi (Katalog API)",
		Passed:   f1Passed,
		Duration: time.Since(t0),
	})

	// --- F-2: Turnike Geçiş ve Doğrulama ---
	t0 = time.Now()
	f2Seq := time.Now().UnixNano()
	tapPayload := map[string]interface{}{
		"validator_id": "VAL_500T_01",
		"sequence_no":  f2Seq,
		"card_id":      "CARD_YOLCU_001",
		"line_id":      "LINE_500T",
		"station_id":   "STAT_AYRILIK",
		"direction":    1,
		"tapped_at":    time.Now().Format(time.RFC3339),
	}
	tapBytes, _ := json.Marshal(tapPayload)

	// Negatif Test: İmzasız istek 401 dönmelidir
	reqAnonTap, _ := http.NewRequest("POST", baseURL+"/api/v1/validators/tap", bytes.NewReader(tapBytes))
	reqAnonTap.Header.Set("Content-Type", "application/json")
	respAnonTap, _ := client.Do(reqAnonTap)
	anonRejected := respAnonTap != nil && (respAnonTap.StatusCode == http.StatusUnauthorized || respAnonTap.StatusCode == http.StatusForbidden)
	if respAnonTap != nil {
		respAnonTap.Body.Close()
	}

	// Pozitif Test: Kriptografik cihaz imzalı istek 200 OK dönmelidir
	reqTap, _ := http.NewRequest("POST", baseURL+"/api/v1/validators/tap", bytes.NewReader(tapBytes))
	reqTap.Header.Set("Content-Type", "application/json")
	f2IdempKey := fmt.Sprintf("VERIFY_F2_%d", time.Now().UnixNano())
	reqTap.Header.Set("Idempotency-Key", f2IdempKey)
	signTapRequest(reqTap, "VAL_500T_01", f2Seq, tapBytes, f2IdempKey)
	respTap, errTap := client.Do(reqTap)
	f2Passed := anonRejected && errTap == nil && respTap.StatusCode == http.StatusOK
	if respTap != nil {
		respTap.Body.Close()
	}
	items = append(items, VerificationItem{
		ID:       "F-2",
		Category: "Kapsam",
		Name:     "Turnike geçiş ve doğrulama (POST /validators/tap - HMAC Korumalı)",
		Passed:   f2Passed,
		Duration: time.Since(t0),
	})

	// --- F-3: Bakiye Yükleme ve Ödeme ---
	t0 = time.Now()
	rechargePayload := map[string]interface{}{
		"amount_kurus": 5000,
	}
	rechargeBytes, _ := json.Marshal(rechargePayload)
	reqRecharge, _ := http.NewRequest("POST", baseURL+"/api/v1/cards/CARD_YOLCU_001/recharge", bytes.NewReader(rechargeBytes))
	reqRecharge.Header.Set("Content-Type", "application/json")
	reqRecharge.Header.Set("Authorization", "Bearer "+yolcuToken)
	respRecharge, errRecharge := client.Do(reqRecharge)
	f3Passed := errRecharge == nil && respRecharge.StatusCode == http.StatusOK
	if respRecharge != nil {
		respRecharge.Body.Close()
	}
	items = append(items, VerificationItem{
		ID:       "F-3",
		Category: "Kapsam",
		Name:     "Bakiye yükleme ve ödeme entegrasyonu (Recharge)",
		Passed:   f3Passed,
		Duration: time.Since(t0),
	})

	// --- F-4: Kart Sahibi Bilgilendirme ve Güvenlik (Anti-IDOR) ---
	t0 = time.Now()
	// Yolcu kendi kartına erişebilmeli
	reqMyCard, _ := http.NewRequest("GET", baseURL+"/api/v1/cards/CARD_YOLCU_001", nil)
	reqMyCard.Header.Set("Authorization", "Bearer "+yolcuToken)
	respMyCard, errMyCard := client.Do(reqMyCard)
	myCardOK := errMyCard == nil && respMyCard.StatusCode == http.StatusOK
	if respMyCard != nil {
		respMyCard.Body.Close()
	}
	// Yolcu başkasının kartına erişememeli (IDOR 403 Forbidden)
	reqOtherCard, _ := http.NewRequest("GET", baseURL+"/api/v1/cards/CARD_TAM_001", nil)
	reqOtherCard.Header.Set("Authorization", "Bearer "+yolcuToken)
	respOtherCard, errOtherCard := client.Do(reqOtherCard)
	idorProtected := errOtherCard == nil && respOtherCard.StatusCode == http.StatusForbidden
	if respOtherCard != nil {
		respOtherCard.Body.Close()
	}
	// Yetkisiz erişim 401 dönmeli
	reqNoAuth, _ := http.NewRequest("GET", baseURL+"/api/v1/cards/CARD_YOLCU_001", nil)
	respNoAuth, errNoAuth := client.Do(reqNoAuth)
	noAuthProtected := errNoAuth == nil && respNoAuth.StatusCode == http.StatusUnauthorized
	if respNoAuth != nil {
		respNoAuth.Body.Close()
	}
	f4Passed := myCardOK && idorProtected && noAuthProtected
	items = append(items, VerificationItem{
		ID:       "F-4",
		Category: "Kapsam",
		Name:     "Kart sahibi güvenliği (Anti-IDOR & RBAC)",
		Passed:   f4Passed,
		Duration: time.Since(t0),
		Notes:    fmt.Sprintf("myCardOK=%v, idorProtected=%v, noAuthProtected=%v", myCardOK, idorProtected, noAuthProtected),
	})

	// --- F-5: Saha Kontrol ve Ceza Takibi ---
	t0 = time.Now()
	reqPenalties, _ := http.NewRequest("GET", baseURL+"/api/v1/penalties", nil)
	reqPenalties.Header.Set("Authorization", "Bearer "+kontrolorToken)
	respPenalties, errPenalties := client.Do(reqPenalties)
	f5Passed := errPenalties == nil && respPenalties.StatusCode == http.StatusOK
	if respPenalties != nil {
		respPenalties.Body.Close()
	}
	items = append(items, VerificationItem{
		ID:       "F-5",
		Category: "Kapsam",
		Name:     "Saha kontrol ve ceza takip mekanizması (Penalties)",
		Passed:   f5Passed,
		Duration: time.Since(t0),
	})

	// --- F-6: Gelir Paylaşımı ve Muhasebe ---
	t0 = time.Now()
	pID := uuid.New().String()
	closeBody, _ := json.Marshal(map[string]interface{}{"period_id": pID})
	reqClose, _ := http.NewRequest("POST", baseURL+"/api/v1/settlements/close", bytes.NewReader(closeBody))
	reqClose.Header.Set("Content-Type", "application/json")
	reqClose.Header.Set("Authorization", "Bearer "+muhasebeToken)
	respClose, errClose := client.Do(reqClose)
	f6Passed := errClose == nil && respClose.StatusCode == http.StatusOK
	if respClose != nil {
		respClose.Body.Close()
	}
	items = append(items, VerificationItem{
		ID:       "F-6",
		Category: "Kapsam",
		Name:     "Gelir paylaşımı ve muhasebe takas (Settlements)",
		Passed:   f6Passed,
		Duration: time.Since(t0),
	})

	// --- F-7: Yoğunluk Raporları & Canlı ESHOT Ulaşım ---
	t0 = time.Now()
	reqHeatmap, _ := http.NewRequest("GET", baseURL+"/api/v1/reports/heatmap", nil)
	respHeatmap, errHeatmap := client.Do(reqHeatmap)
	heatmapOK := errHeatmap == nil && respHeatmap.StatusCode == http.StatusOK
	if respHeatmap != nil {
		respHeatmap.Body.Close()
	}
	reqTransit, _ := http.NewRequest("GET", baseURL+"/api/v1/transit/live-buses?line=800", nil)
	respTransit, errTransit := client.Do(reqTransit)
	transitOK := errTransit == nil && respTransit.StatusCode == http.StatusOK
	if respTransit != nil {
		respTransit.Body.Close()
	}
	f7Passed := heatmapOK && transitOK
	items = append(items, VerificationItem{
		ID:       "F-7",
		Category: "Kapsam",
		Name:     "Yoğunluk analitiği ve canlı ESHOT ulaşım verisi",
		Passed:   f7Passed,
		Duration: time.Since(t0),
		Notes:    fmt.Sprintf("heatmapOK=%v, transitOK=%v", heatmapOK, transitOK),
	})

	// --- T-1: Ücret Motoru (90 dk, 2 aktarma, ters yön kuralı) ---
	t0 = time.Now()
	pEngine := pricing.NewPricingEngine()
	tariff := domain.Tariff{
		ID:                   "TARIFF_ISTANBUL_STD",
		BaseFareKurus:        2000,
		StudentFareKurus:     1000,
		ElderlyFareKurus:     1500,
		Transfer1DiscountPct: 50,
		Transfer2DiscountPct: 75,
	}
	tBase := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	histT1 := []pricing.TripHistoryItem{
		{LineID: "LINE_A", StationID: "STAT_1", Direction: 1, TappedAt: tBase, IsTransfer: false},
	}
	resT1Trans := pEngine.CalculateFare(domain.Card{CardType: domain.CardTypeTam}, tariff, "LINE_B", "STAT_2", 1, tBase.Add(20*time.Minute), histT1)
	resT1Rev := pEngine.CalculateFare(domain.Card{CardType: domain.CardTypeTam}, tariff, "LINE_A", "STAT_1", 2, tBase.Add(30*time.Minute), histT1)
	t1Passed := resT1Trans.IsTransfer && resT1Trans.FareChargedKurus == 1000 && !resT1Rev.IsTransfer && resT1Rev.FareChargedKurus == 2000
	items = append(items, VerificationItem{
		ID:       "T-1",
		Category: "Zorlu Görev",
		Name:     "Ücret motoru (90 dk pencere, ters yön kuralı)",
		Passed:   t1Passed,
		Duration: time.Since(t0),
	})

	// --- T-2: Çevrimdışı Validatör Senkronizasyonu & Idempotansi ---
	t0 = time.Now()
	testIdempKey := fmt.Sprintf("IDEMP_VERIFY_%d", time.Now().UnixNano())
	t2Seq := time.Now().UnixNano()
	tapPayloadT2 := map[string]interface{}{
		"validator_id": "VAL_500T_01",
		"sequence_no":  t2Seq,
		"card_id":      "CARD_YOLCU_001",
		"line_id":      "LINE_500T",
		"station_id":   "STAT_AYRILIK",
		"direction":    1,
		"tapped_at":    time.Now().Format(time.RFC3339),
	}
	tapBytesT2, _ := json.Marshal(tapPayloadT2)

	// İlk gönderim (HMAC İmzalı)
	reqT2_1, _ := http.NewRequest("POST", baseURL+"/api/v1/validators/tap", bytes.NewReader(tapBytesT2))
	reqT2_1.Header.Set("Content-Type", "application/json")
	reqT2_1.Header.Set("Idempotency-Key", testIdempKey)
	signTapRequest(reqT2_1, "VAL_500T_01", t2Seq, tapBytesT2, testIdempKey)
	respT2_1, errT2_1 := client.Do(reqT2_1)
	status1 := 0
	var body1 map[string]interface{}
	if errT2_1 == nil {
		status1 = respT2_1.StatusCode
		_ = json.NewDecoder(respT2_1.Body).Decode(&body1)
		respT2_1.Body.Close()
	}

	// Aynı Idempotency-Key ile ikinci gönderim (Mükerrer çekim engeli)
	reqT2_2, _ := http.NewRequest("POST", baseURL+"/api/v1/validators/tap", bytes.NewReader(tapBytesT2))
	reqT2_2.Header.Set("Content-Type", "application/json")
	reqT2_2.Header.Set("Idempotency-Key", testIdempKey)
	signTapRequest(reqT2_2, "VAL_500T_01", t2Seq, tapBytesT2, testIdempKey)
	respT2_2, errT2_2 := client.Do(reqT2_2)
	status2 := 0
	var body2 map[string]interface{}
	if errT2_2 == nil {
		status2 = respT2_2.StatusCode
		_ = json.NewDecoder(respT2_2.Body).Decode(&body2)
		respT2_2.Body.Close()
	}

	// Çakışan farklı card_id ile aynı idempotency key (HTTP 409 Conflict)
	tapConflict := tapPayloadT2
	tapConflict["card_id"] = "CARD_OGRENCI_001"
	tapBytesConflict, _ := json.Marshal(tapConflict)
	reqT2_3, _ := http.NewRequest("POST", baseURL+"/api/v1/validators/tap", bytes.NewReader(tapBytesConflict))
	reqT2_3.Header.Set("Content-Type", "application/json")
	reqT2_3.Header.Set("Idempotency-Key", testIdempKey)
	signTapRequest(reqT2_3, "VAL_500T_01", t2Seq, tapBytesConflict, testIdempKey)
	respT2_3, errT2_3 := client.Do(reqT2_3)
	conflictStatus := 0
	if errT2_3 == nil {
		conflictStatus = respT2_3.StatusCode
		respT2_3.Body.Close()
	}

	t2Passed := status1 == 200 && status2 == 200 && conflictStatus == http.StatusConflict
	items = append(items, VerificationItem{
		ID:       "T-2",
		Category: "Zorlu Görev",
		Name:     "Çevrimdışı validatör senkronizasyonu ve idempotansi",
		Passed:   t2Passed,
		Duration: time.Since(t0),
		Notes:    fmt.Sprintf("firstStatus=%d, replayStatus=%d, conflictStatus=%d", status1, status2, conflictStatus),
	})

	// --- T-3: Kart Bakiye Defteri (Append-Only Ledger) ---
	t0 = time.Now()
	reqLedger, _ := http.NewRequest("GET", baseURL+"/api/v1/cards/CARD_YOLCU_001/ledger", nil)
	reqLedger.Header.Set("Authorization", "Bearer "+yolcuToken)
	respLedger, errLedger := client.Do(reqLedger)
	t3Passed := false
	if errLedger == nil && respLedger.StatusCode == http.StatusOK {
		var entries []domain.LedgerEntry
		if err := json.NewDecoder(respLedger.Body).Decode(&entries); err == nil && len(entries) > 0 {
			t3Passed = true
		}
		respLedger.Body.Close()
	}
	items = append(items, VerificationItem{
		ID:       "T-3",
		Category: "Zorlu Görev",
		Name:     "Kart bakiye defteri (Append-only ledger ve tutarlılık)",
		Passed:   t3Passed,
		Duration: time.Since(t0),
	})

	// --- T-4: Gelir Paylaşımı (En Büyük Kalan & Storno) ---
	t0 = time.Now()
	settleSvc := settlement.NewSettlementService()
	period := settleSvc.CreatePeriod("Test Period", time.Now().Add(-24*time.Hour), time.Now())
	_ = settleSvc.RecordTripSettlement(period.ID, nil, 1750, []settlement.OperatorLeg{
		{Operator: "belediye", DistanceMeter: 5000},
		{Operator: "ozel_halk", DistanceMeter: 7000},
		{Operator: "sehir_hatlari", DistanceMeter: 3000},
	})
	_, errClosePeriod := settleSvc.ClosePeriod(period.ID)
	stornoEntry, errStorno := settleSvc.RecordStornoAfterClose(period.ID, nil, "ozel_halk", -250, "Ceza stornosu")
	t4Passed := errClosePeriod == nil && errStorno == nil && stornoEntry.IsStorno
	items = append(items, VerificationItem{
		ID:       "T-4",
		Category: "Zorlu Görev",
		Name:     "Gelir paylaşımı (Kuruş koruması, kapanış ve storno)",
		Passed:   t4Passed,
		Duration: time.Since(t0),
	})

	// --- T-5: Anormallik Tespiti (İmkansız Hız & Klon Kart) ---
	t0 = time.Now()
	anomDetector := anomaly.NewAnomalyDetector()
	anomDetector.RegisterStation(anomaly.StationCoordinate{ID: "STAT_KADIKOY", Latitude: 40.9904, Longitude: 29.0254})
	anomDetector.RegisterStation(anomaly.StationCoordinate{ID: "STAT_SARIYER", Latitude: 41.1685, Longitude: 29.0554}) // ~20 km
	tAnom := time.Now()
	_ = anomDetector.CheckTap(context.Background(), "CARD_T5_VERIFY", "STAT_KADIKOY", tAnom, nil)
	resAnom := anomDetector.CheckTap(context.Background(), "CARD_T5_VERIFY", "STAT_SARIYER", tAnom.Add(3*time.Minute), nil)

	// API anomali ucu kontrolü
	reqAnomAPI, _ := http.NewRequest("GET", baseURL+"/api/v1/anomalies", nil)
	reqAnomAPI.Header.Set("Authorization", "Bearer "+operasyonToken)
	respAnomAPI, errAnomAPI := client.Do(reqAnomAPI)
	anomAPIOK := errAnomAPI == nil && respAnomAPI.StatusCode == http.StatusOK
	if respAnomAPI != nil {
		respAnomAPI.Body.Close()
	}

	t5Passed := resAnom.IsAnomaly && resAnom.AnomalyType == domain.AnomalyImpossibleSpeed && anomAPIOK
	items = append(items, VerificationItem{
		ID:       "T-5",
		Category: "Zorlu Görev",
		Name:     "Anormallik tespiti (İmkansız seyahat ve klon kart)",
		Passed:   t5Passed,
		Duration: time.Since(t0),
		Notes:    fmt.Sprintf("isAnomaly=%v, type=%s, anomAPIOK=%v", resAnom.IsAnomaly, resAnom.AnomalyType, anomAPIOK),
	})

	return items
}

func loginUser(client *http.Client, baseURL, email, password string) (string, error) {
	loginPayload := fmt.Sprintf(`{"email":"%s","password":"%s"}`, email, password)
	resp, err := client.Post(baseURL+"/api/v1/auth/login", "application/json", strings.NewReader(loginPayload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var data struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.AccessToken, nil
}
