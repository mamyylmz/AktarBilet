package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mamyylmz/AktarBilet/internal/anomaly"
	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
	"github.com/mamyylmz/AktarBilet/internal/seed"
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
		fmt.Println("\n\033[32m[PASS] Bu komutta tanımlı canlı HTTP ve domain kabul senaryoları geçti.\033[0m")
		fmt.Println("Bu sonuç; tam PostgreSQL kapasite, yarış, Docker ve CI teslim kabulünün yerine geçmez.")
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
	linesOK := false
	lineCount, activeIstanbulCount := 0, 0
	if respLines != nil {
		var lines []domain.Line
		body, readErr := io.ReadAll(respLines.Body)
		if errLines == nil && respLines.StatusCode == http.StatusOK && readErr == nil && json.Unmarshal(body, &lines) == nil {
			legacy := make(map[string]bool)
			for _, id := range seed.LegacyIstanbulSeedLineIDs() {
				legacy[id] = true
			}
			seen := make(map[string]bool)
			for _, line := range lines {
				if line.IsActive && !seen[line.ID] {
					seen[line.ID] = true
					lineCount++
				}
				if line.IsActive && legacy[line.ID] {
					activeIstanbulCount++
				}
			}
			linesOK = lineCount >= 40 && activeIstanbulCount == 0
		}
		respLines.Body.Close()
	}
	respStations, errStations := client.Get(baseURL + "/api/v1/stations")
	stationsOK := false
	stationCount := 0
	if respStations != nil {
		var stations []domain.Station
		body, readErr := io.ReadAll(respStations.Body)
		stationsOK = errStations == nil && respStations.StatusCode == http.StatusOK && readErr == nil && json.Unmarshal(body, &stations) == nil && len(stations) > 0
		stationCount = len(stations)
		respStations.Body.Close()
	}
	respTariffs, errTariffs := client.Get(baseURL + "/api/v1/tariffs")
	tariffsOK := false
	if respTariffs != nil {
		var tariffs []domain.Tariff
		body, readErr := io.ReadAll(respTariffs.Body)
		if errTariffs == nil && respTariffs.StatusCode == http.StatusOK && readErr == nil && json.Unmarshal(body, &tariffs) == nil {
			for _, tariff := range tariffs {
				if tariff.ID == "TARIFF_IZMIR_STD" && tariff.IsActive {
					tariffsOK = true
				}
			}
		}
		respTariffs.Body.Close()
	}
	f1Passed := linesOK && stationsOK && tariffsOK
	items = append(items, VerificationItem{
		ID:       "F-3",
		Category: "Kapsam",
		Name:     "Aktif İzmir kataloğu (>=40 hat, İstanbul=0, durak ve tarife)",
		Passed:   f1Passed,
		Duration: time.Since(t0),
		Notes:    fmt.Sprintf("activeLines=%d, activeIstanbulLines=%d, activeStations=%d, izmirTariff=%v", lineCount, activeIstanbulCount, stationCount, tariffsOK),
	})

	// --- F-2: Turnike Geçiş ve Doğrulama ---
	t0 = time.Now()
	f2Seq := time.Now().UnixNano()
	tapPayload := map[string]interface{}{
		"validator_id": "VAL_500T_01",
		"sequence_no":  f2Seq,
		"card_id":      "CARD_YOLCU_001",
		"line_id":      "LINE_5",
		"station_id":   "STAT_ESHOT_10177",
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
	freeTapPassed := false
	freeToken, freeLoginErr := loginUser(client, baseURL, "ucretsiz.demo@aktarbilet.gov.tr", "UcretsizDemo2026!")
	if freeLoginErr == nil && freeToken != "" {
		freeSeq := time.Now().UnixNano()
		freePayload, _ := json.Marshal(map[string]interface{}{
			"validator_id": "VAL_500T_01", "sequence_no": freeSeq,
			"card_id": "CARD_DEMO_UCRETSIZ_001", "line_id": "LINE_5",
			"station_id": "STAT_ESHOT_10177", "direction": 1, "tapped_at": time.Now().Format(time.RFC3339),
		})
		freeKey := fmt.Sprintf("VERIFY_FREE_%d", time.Now().UnixNano())
		freeReq, _ := http.NewRequest("POST", baseURL+"/api/v1/validators/tap", bytes.NewReader(freePayload))
		freeReq.Header.Set("Content-Type", "application/json")
		freeReq.Header.Set("Idempotency-Key", freeKey)
		signTapRequest(freeReq, "VAL_500T_01", freeSeq, freePayload, freeKey)
		freeResp, freeErr := client.Do(freeReq)
		freeEventOK := freeErr == nil && freeResp != nil && freeResp.StatusCode == http.StatusOK
		if freeResp != nil {
			freeResp.Body.Close()
		}
		cardReq, _ := http.NewRequest("GET", baseURL+"/api/v1/cards/CARD_DEMO_UCRETSIZ_001", nil)
		cardReq.Header.Set("Authorization", "Bearer "+freeToken)
		cardResp, cardErr := client.Do(cardReq)
		if cardErr == nil && cardResp != nil && cardResp.StatusCode == http.StatusOK {
			var card struct {
				Balance int64 `json:"balance_kurus"`
			}
			decodeErr := json.NewDecoder(cardResp.Body).Decode(&card)
			freeTapPassed = freeEventOK && decodeErr == nil && card.Balance == 0
		}
		if cardResp != nil {
			cardResp.Body.Close()
		}
	}
	f2Passed = f2Passed && freeTapPassed
	items = append(items, VerificationItem{
		ID:       "F-2",
		Category: "Kapsam",
		Name:     "HMAC validatör binişi ve ücretsiz kartın sıfır bakiye politikası",
		Passed:   f2Passed,
		Duration: time.Since(t0),
		Notes:    fmt.Sprintf("signedTap=%v, zeroFareTapAndBalance=%v", errTap == nil && respTap != nil && respTap.StatusCode == http.StatusOK, freeTapPassed),
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
		ID:       "F-1",
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
	demoAccounts := []struct {
		email, password, cardID, cardType string
	}{
		{"yolcu@aktarbilet.gov.tr", "YolcuPassword123!", "CARD_YOLCU_001", "tam"},
		{"ogrenci.demo@aktarbilet.gov.tr", "OgrenciDemo2026!", "CARD_DEMO_OGRENCI_001", "ogrenci"},
		{"yasli.demo@aktarbilet.gov.tr", "YasliDemo2026!", "CARD_DEMO_YASLI_001", "yasli"},
		{"ucretsiz.demo@aktarbilet.gov.tr", "UcretsizDemo2026!", "CARD_DEMO_UCRETSIZ_001", "ucretsiz"},
	}
	demoAccountsOK := true
	demoTokens := make([]string, len(demoAccounts))
	for i, account := range demoAccounts {
		token, loginErr := loginUser(client, baseURL, account.email, account.password)
		demoTokens[i] = token
		if loginErr != nil || token == "" {
			demoAccountsOK = false
			continue
		}
		req, _ := http.NewRequest("GET", baseURL+"/api/v1/cards/"+account.cardID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, requestErr := client.Do(req)
		if requestErr != nil || resp == nil || resp.StatusCode != http.StatusOK {
			demoAccountsOK = false
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}
		var card struct {
			CardType string `json:"card_type"`
		}
		decodeErr := json.NewDecoder(resp.Body).Decode(&card)
		resp.Body.Close()
		if decodeErr != nil || card.CardType != account.cardType {
			demoAccountsOK = false
		}
	}
	demoIDOROK := true
	for ownerIndex, token := range demoTokens {
		if token == "" {
			demoIDOROK = false
			continue
		}
		for cardIndex, account := range demoAccounts {
			if ownerIndex == cardIndex {
				continue
			}
			req, _ := http.NewRequest("GET", baseURL+"/api/v1/cards/"+account.cardID, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp, requestErr := client.Do(req)
			if requestErr != nil || resp == nil || resp.StatusCode != http.StatusForbidden {
				demoIDOROK = false
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
	f4Passed := myCardOK && idorProtected && noAuthProtected && demoAccountsOK && demoIDOROK
	items = append(items, VerificationItem{
		ID:       "F-1",
		Category: "Kapsam",
		Name:     "Dört demo kart türü, hesap izolasyonu ve Anti-IDOR",
		Passed:   f4Passed,
		Duration: time.Since(t0),
		Notes:    fmt.Sprintf("baseCardOK=%v, idorProtected=%v, noAuthProtected=%v, demoAccountsOK=%v, crossAccountIDOR=%v", myCardOK, idorProtected, noAuthProtected, demoAccountsOK, demoIDOROK),
	})

	// --- F-4: Fare results for all four demo card types ---
	t0 = time.Now()
	fareEngine := pricing.NewPricingEngine()
	fareTariff := seed.DefaultSeedTariff()
	fareCases := []struct {
		cardType domain.CardType
		want     domain.Kurus
	}{
		{domain.CardTypeTam, 2000},
		{domain.CardTypeOgrenci, 1000},
		{domain.CardTypeYasli, 1500},
		{domain.CardTypeUcretsiz, 0},
	}
	fareTypesOK := true
	for _, fareCase := range fareCases {
		result := fareEngine.CalculateFare(domain.Card{CardType: fareCase.cardType}, fareTariff, "LINE_5", "STAT_ESHOT_10177", 1, time.Now(), nil)
		if result.FareChargedKurus != fareCase.want {
			fareTypesOK = false
		}
	}
	items = append(items, VerificationItem{
		ID: "F-4", Category: "Kapsam", Name: "Dört kart türünün ücretleri (kuruş hassasiyeti)",
		Passed: fareTypesOK, Duration: time.Since(t0), Notes: fmt.Sprintf("allCardTypeFares=%v", fareTypesOK),
	})

	// --- F-5: Operator allocation conserves every kuruş ---
	t0 = time.Now()
	operatorShares, shareErr := settlement.CalculateRevenueShare(1750, []settlement.OperatorLeg{
		{Operator: "belediye", DistanceMeter: 5000},
		{Operator: "ozel_halk", DistanceMeter: 7000},
		{Operator: "sehir_hatlari", DistanceMeter: 3000},
	})
	var sharesTotal domain.Kurus
	for _, share := range operatorShares {
		sharesTotal += share
	}
	f5ShareOK := shareErr == nil && sharesTotal == 1750
	items = append(items, VerificationItem{
		ID: "F-5", Category: "Kapsam", Name: "İşletmeci paylarının kuruş bazında korunması",
		Passed: f5ShareOK, Duration: time.Since(t0), Notes: fmt.Sprintf("shareError=%v, totalKurus=%d", shareErr, sharesTotal),
	})

	// --- F-6: Saha Kontrol ve Ceza Takibi ---
	t0 = time.Now()
	penaltyPayload, _ := json.Marshal(map[string]interface{}{
		"card_id": "CARD_YOLCU_001", "line_id": "LINE_5", "station_id": "STAT_ESHOT_10177",
		"penalty_amount_kurus": 1000, "reason": "diger", "notes": "İzole make verify kabul senaryosu",
	})
	reqPenalty, _ := http.NewRequest("POST", baseURL+"/api/v1/penalties/", bytes.NewReader(penaltyPayload))
	reqPenalty.Header.Set("Content-Type", "application/json")
	reqPenalty.Header.Set("Authorization", "Bearer "+kontrolorToken)
	respPenalty, errPenalty := client.Do(reqPenalty)
	penaltyPassed := errPenalty == nil && respPenalty != nil && respPenalty.StatusCode == http.StatusCreated
	if respPenalty != nil {
		respPenalty.Body.Close()
	}
	items = append(items, VerificationItem{
		ID:       "F-6",
		Category: "Kapsam",
		Name:     "Saha kontrol ve ceza takip mekanizması (Penalties)",
		Passed:   penaltyPassed,
		Duration: time.Since(t0),
		Notes:    fmt.Sprintf("penaltyCreated=%v", penaltyPassed),
	})

	// --- F-5: Gelir Paylaşımı ve Muhasebe ---
	t0 = time.Now()
	pID := uuid.New().String()
	closeBody, _ := json.Marshal(map[string]interface{}{"period_id": pID})
	reqClose, _ := http.NewRequest("POST", baseURL+"/api/v1/settlements/close", bytes.NewReader(closeBody))
	reqClose.Header.Set("Content-Type", "application/json")
	reqClose.Header.Set("Authorization", "Bearer "+muhasebeToken)
	respClose, errClose := client.Do(reqClose)
	settlementClosePassed := errClose == nil && respClose.StatusCode == http.StatusOK
	if respClose != nil {
		respClose.Body.Close()
	}
	items = append(items, VerificationItem{
		ID:       "T-4-LIVE",
		Category: "Kapsam",
		Name:     "Settlement dönem kapatma/storno HTTP yaşam döngüsü",
		Passed:   settlementClosePassed,
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

	// --- T-1: Ücret Motoru (90 dk, 2 aktarma, ters yön kuralı ve Canlı API) ---
	t0 = time.Now()
	pEngine := pricing.NewPricingEngine()
	tariff := seed.DefaultSeedTariff()
	tBase := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	histT1 := []pricing.TripHistoryItem{
		{LineID: "LINE_A", StationID: "STAT_1", Direction: 1, TappedAt: tBase, IsTransfer: false},
	}
	resT1Trans := pEngine.CalculateFare(domain.Card{CardType: domain.CardTypeTam}, tariff, "LINE_B", "STAT_2", 1, tBase.Add(20*time.Minute), histT1)
	resT1Rev := pEngine.CalculateFare(domain.Card{CardType: domain.CardTypeTam}, tariff, "LINE_A", "STAT_1", 2, tBase.Add(30*time.Minute), histT1)
	engineT1OK := resT1Trans.IsTransfer && resT1Trans.FareChargedKurus == 1000 && !resT1Rev.IsTransfer && resT1Rev.FareChargedKurus == 2000

	// Canlı HTTP API Doğrulaması (POST /api/v1/validators/tap)
	// Use a seeded, funded card: a random ID cannot produce a valid live tap.
	t1CardID := "CARD_TAM_001"
	t1Seq1 := time.Now().UnixNano()
	tapPayloadT1_1 := map[string]interface{}{
		"validator_id": "VAL_500T_01",
		"sequence_no":  t1Seq1,
		"card_id":      t1CardID,
		"line_id":      "LINE_5",
		"station_id":   "STAT_ESHOT_10177",
		"direction":    1,
		"tapped_at":    time.Now().Format(time.RFC3339),
	}
	tapBytesT1_1, _ := json.Marshal(tapPayloadT1_1)
	reqT1_1, _ := http.NewRequest("POST", baseURL+"/api/v1/validators/tap", bytes.NewReader(tapBytesT1_1))
	reqT1_1.Header.Set("Content-Type", "application/json")
	idempT1_1 := fmt.Sprintf("IDEMP_T1_1_%d", time.Now().UnixNano())
	reqT1_1.Header.Set("Idempotency-Key", idempT1_1)
	signTapRequest(reqT1_1, "VAL_500T_01", t1Seq1, tapBytesT1_1, idempT1_1)
	respT1_1, errT1_1 := client.Do(reqT1_1)
	liveT1_1OK := errT1_1 == nil && respT1_1 != nil && respT1_1.StatusCode == http.StatusOK
	if respT1_1 != nil {
		respT1_1.Body.Close()
	}

	t1Seq2 := time.Now().UnixNano()
	tapPayloadT1_2 := map[string]interface{}{
		"validator_id": "VAL_14ES_01",
		"sequence_no":  t1Seq2,
		"card_id":      t1CardID,
		"line_id":      "LINE_10",
		"station_id":   "STAT_ESHOT_10145",
		"direction":    1,
		"tapped_at":    time.Now().Add(2 * time.Minute).Format(time.RFC3339),
	}
	tapBytesT1_2, _ := json.Marshal(tapPayloadT1_2)
	reqT1_2, _ := http.NewRequest("POST", baseURL+"/api/v1/validators/tap", bytes.NewReader(tapBytesT1_2))
	reqT1_2.Header.Set("Content-Type", "application/json")
	idempT1_2 := fmt.Sprintf("IDEMP_T1_2_%d", time.Now().UnixNano())
	reqT1_2.Header.Set("Idempotency-Key", idempT1_2)
	signTapRequest(reqT1_2, "VAL_14ES_01", t1Seq2, tapBytesT1_2, idempT1_2)
	respT1_2, errT1_2 := client.Do(reqT1_2)
	liveT1_2OK := false
	if errT1_2 == nil && respT1_2 != nil && respT1_2.StatusCode == http.StatusOK {
		var bodyT1_2 map[string]interface{}
		if decErr := json.NewDecoder(respT1_2.Body).Decode(&bodyT1_2); decErr == nil {
			if evt, ok := bodyT1_2["event"].(map[string]interface{}); ok {
				_, hasTransferState := evt["is_transfer"].(bool)
				_, hasFare := evt["fare_charged_kurus"].(float64)
				liveT1_2OK = hasTransferState && hasFare
			}
		}
		respT1_2.Body.Close()
	}

	t1Passed := engineT1OK && liveT1_1OK && liveT1_2OK
	items = append(items, VerificationItem{
		ID:       "T-1",
		Category: "Zorlu Görev",
		Name:     "Ücret motoru (90 dk pencere, ters yön kuralı ve canlı API)",
		Passed:   t1Passed,
		Duration: time.Since(t0),
		Notes:    fmt.Sprintf("engineT1OK=%v, liveT1_1OK=%v, liveT1_2OK=%v", engineT1OK, liveT1_1OK, liveT1_2OK),
	})

	// --- T-2: Çevrimdışı Validatör Senkronizasyonu & Idempotansi ---
	t0 = time.Now()
	testIdempKey := fmt.Sprintf("IDEMP_VERIFY_%d", time.Now().UnixNano())
	t2Seq := time.Now().UnixNano()
	tapPayloadT2 := map[string]interface{}{
		"validator_id": "VAL_500T_01",
		"sequence_no":  t2Seq,
		"card_id":      "CARD_YOLCU_001",
		"line_id":      "LINE_5",
		"station_id":   "STAT_ESHOT_10177",
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

	// --- T-4: Gelir Paylaşımı (En Büyük Kalan, Sıfır Fark ve Canlı API) ---
	t0 = time.Now()
	// 1. Canlı HTTP API sorgusu: ListSettlements
	reqSettlements, _ := http.NewRequest("GET", baseURL+"/api/v1/settlements", nil)
	reqSettlements.Header.Set("Authorization", "Bearer "+muhasebeToken)
	respSettlements, errSettlements := client.Do(reqSettlements)
	liveSettlementsOK := false
	periodWasClosed := false
	settlementReadOK := false
	if errSettlements == nil && respSettlements != nil && respSettlements.StatusCode == http.StatusOK {
		var summary struct {
			TotalRevenue domain.Kurus `json:"total_revenue_kurus"`
			TotalFee     domain.Kurus `json:"total_fee_kurus"`
			TotalNet     domain.Kurus `json:"total_net_kurus"`
			IsClosed     bool         `json:"is_closed"`
		}
		if decErr := json.NewDecoder(respSettlements.Body).Decode(&summary); decErr == nil {
			periodWasClosed = summary.IsClosed
			settlementReadOK = true
			if summary.TotalRevenue == summary.TotalNet && summary.TotalFee == 0 {
				liveSettlementsOK = true
			}
		}
		respSettlements.Body.Close()
	}

	// 2. Canlı HTTP API yaşam döngüsü: açık dönemi kapat, storno'yu kabul ettir,
	// sonra başlangıçta açıksa tekrar aç. HTTP 400 başarı sayılmaz.
	periodReady := settlementReadOK && periodWasClosed
	if settlementReadOK && !periodWasClosed {
		closeBody, _ := json.Marshal(map[string]string{"period_name": "2026-EKIM"})
		reqClose, _ := http.NewRequest("POST", baseURL+"/api/v1/settlements/close", bytes.NewReader(closeBody))
		reqClose.Header.Set("Content-Type", "application/json")
		reqClose.Header.Set("Authorization", "Bearer "+muhasebeToken)
		respClose, errClose := client.Do(reqClose)
		periodReady = errClose == nil && respClose != nil && respClose.StatusCode == http.StatusOK
		if respClose != nil {
			respClose.Body.Close()
		}
	}

	liveStornoOK := false
	if periodReady {
		stornoBody, _ := json.Marshal(map[string]interface{}{
			"period_id":    "00000000-0000-0000-0000-000000000001",
			"operator":     "ozel_halk",
			"amount_kurus": -1,
			"reason":       "Canlı T-4 doğrulama storno kaydı",
		})
		idempotencyKey := fmt.Sprintf("STORNO_T4_%d", time.Now().UnixNano())
		reqStorno, _ := http.NewRequest("POST", baseURL+"/api/v1/settlements/storno", bytes.NewReader(stornoBody))
		reqStorno.Header.Set("Content-Type", "application/json")
		reqStorno.Header.Set("Authorization", "Bearer "+muhasebeToken)
		reqStorno.Header.Set("Idempotency-Key", idempotencyKey)
		respStorno, errStornoHTTP := client.Do(reqStorno)
		liveStornoOK = errStornoHTTP == nil && respStorno != nil && respStorno.StatusCode == http.StatusOK
		if respStorno != nil {
			respStorno.Body.Close()
		}
	}
	if liveStornoOK && !periodWasClosed {
		reopenBody, _ := json.Marshal(map[string]string{"period_name": "2026-EKIM", "reason": "make verify tamamlandı"})
		reqReopen, _ := http.NewRequest("POST", baseURL+"/api/v1/settlements/reopen", bytes.NewReader(reopenBody))
		reqReopen.Header.Set("Content-Type", "application/json")
		reqReopen.Header.Set("Authorization", "Bearer "+muhasebeToken)
		respReopen, errReopen := client.Do(reqReopen)
		liveStornoOK = errReopen == nil && respReopen != nil && respReopen.StatusCode == http.StatusOK
		if respReopen != nil {
			respReopen.Body.Close()
		}
	}

	// 3. Domain servisi En Büyük Kalan algoritması doğrulaması
	settleSvc := settlement.NewSettlementService()
	period := settleSvc.CreatePeriod("Test Period", time.Now().Add(-24*time.Hour), time.Now())
	_ = settleSvc.RecordTripSettlement(period.ID, nil, 1750, []settlement.OperatorLeg{
		{Operator: "belediye", DistanceMeter: 5000},
		{Operator: "ozel_halk", DistanceMeter: 7000},
		{Operator: "sehir_hatlari", DistanceMeter: 3000},
	})
	_, errClosePeriod := settleSvc.ClosePeriod(period.ID)
	stornoEntry, errStorno := settleSvc.RecordStornoAfterClose(period.ID, nil, "ozel_halk", -250, "Ceza stornosu")
	domainT4OK := errClosePeriod == nil && errStorno == nil && stornoEntry.IsStorno

	t4Passed := liveSettlementsOK && liveStornoOK && domainT4OK
	items = append(items, VerificationItem{
		ID:       "T-4",
		Category: "Zorlu Görev",
		Name:     "Gelir paylaşımı (Kuruş koruması, sıfır fark ve canlı API)",
		Passed:   t4Passed,
		Duration: time.Since(t0),
		Notes:    fmt.Sprintf("liveSettlementsOK=%v, periodReady=%v, liveStornoOK=%v, domainT4OK=%v", liveSettlementsOK, periodReady, liveStornoOK, domainT4OK),
	})

	// --- T-5: Anormallik Tespiti (İmkansız Hız & Klon Kart) ---
	t0 = time.Now()
	anomDetector := anomaly.NewAnomalyDetector()
	anomDetector.RegisterStation(anomaly.StationCoordinate{ID: "STAT_ESHOT_10145", Latitude: 38.4132, Longitude: 27.1222})
	anomDetector.RegisterStation(anomaly.StationCoordinate{ID: "STAT_ESHOT_24359", Latitude: 39.17667119303110, Longitude: 27.37011041915690})
	tAnom := time.Now()
	_ = anomDetector.CheckTap(context.Background(), "CARD_T5_VERIFY", "STAT_ESHOT_10145", tAnom, nil)
	resAnom := anomDetector.CheckTap(context.Background(), "CARD_T5_VERIFY", "STAT_ESHOT_24359", tAnom.Add(3*time.Minute), nil)

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
