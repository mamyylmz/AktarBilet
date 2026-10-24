package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type SimulatorConfig struct {
	Seed       int64
	TotalCards int
	TotalLines int
	TargetURL  string
	IntervalMs int
	BatchSize  int
}

type TapPayload struct {
	ValidatorID string    `json:"validator_id"`
	SequenceNo  int64     `json:"sequence_no"`
	CardID      string    `json:"card_id"`
	LineID      string    `json:"line_id"`
	StationID   string    `json:"station_id"`
	Direction   int       `json:"direction"`
	TappedAt    time.Time `json:"tapped_at"`
}

func main() {
	var cfg SimulatorConfig
	flag.Int64Var(&cfg.Seed, "seed", 42, "Deterministik tohum (Aynı tohum = aynı veri)")
	flag.IntVar(&cfg.TotalCards, "cards", 200000, "Toplam simüle edilecek kart sayısı")
	flag.IntVar(&cfg.TotalLines, "lines", 40, "Toplam simüle edilecek hat sayısı")
	flag.StringVar(&cfg.TargetURL, "target-url", "http://localhost:8080", "AktarBilet Backend API adresi")
	flag.IntVar(&cfg.IntervalMs, "interval", 1000, "Olay üretim sıklığı (ms)")
	flag.IntVar(&cfg.BatchSize, "batch-size", 10, "Çevrimdışı araç paket boyutu")
	flag.Parse()

	if envURL := os.Getenv("BACKEND_URL"); envURL != "" {
		cfg.TargetURL = envURL
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("AktarBilet Çevrimdışı Araç ve Biniş Simülatörü Başlatıldı",
		"seed", cfg.Seed,
		"cards", cfg.TotalCards,
		"lines", cfg.TotalLines,
		"target", cfg.TargetURL,
	)

	rng := rand.New(rand.NewSource(cfg.Seed))

	// 40 Gerçek Tohumlanmış Hat
	lines := []string{
		"LINE_500T", "LINE_14ES", "LINE_T1", "LINE_VAPUR_KE", "LINE_M1A",
		"LINE_M2", "LINE_M3", "LINE_M4", "LINE_M5", "LINE_M7",
		"LINE_M8", "LINE_M9", "LINE_M11", "LINE_T4", "LINE_T5",
		"LINE_VAPUR_BU", "LINE_VAPUR_KK", "LINE_VAPUR_BO", "LINE_VAPUR_ADALAR", "LINE_34AS",
		"LINE_34BZ", "LINE_34G", "LINE_34Z", "LINE_15F", "LINE_16D",
		"LINE_19F", "LINE_25T", "LINE_40T", "LINE_70FE", "LINE_89C",
		"LINE_97GE", "LINE_145T", "LINE_150", "LINE_151", "LINE_152",
		"LINE_153", "LINE_154", "LINE_250", "LINE_500ES", "LINE_E10",
	}

	stations := []string{"STAT_KADIKOY", "STAT_EMINONU", "STAT_KABATAS", "STAT_BAGCIAR", "STAT_AYRILIK", "STAT_SARIYER"}

	client := &http.Client{Timeout: 5 * time.Second}

	// Backend'in hazır olmasını bekle
	waitForBackend(client, cfg.TargetURL)

	ticker := time.NewTicker(time.Duration(cfg.IntervalMs) * time.Millisecond)
	defer ticker.Stop()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	validatorSeqCounters := make(map[string]int64)

	slog.Info("Canlı biniş simülasyonu çalışıyor...")

	for {
		select {
		case <-shutdown:
			slog.Info("Simülatör durduruldu.")
			return
		case <-ticker.C:
			// Çevrimdışı araç tamponu simülasyonu (Batch event üretimi)
			valID := fmt.Sprintf("VAL_SIM_VEHICLE_%03d", rng.Intn(50)+1)
			lineID := lines[rng.Intn(len(lines))]
			batch := make([]TapPayload, cfg.BatchSize)

			for i := 0; i < cfg.BatchSize; i++ {
				validatorSeqCounters[valID]++
				seq := validatorSeqCounters[valID]

				cardIdx := rng.Intn(cfg.TotalCards) + 1
				cardID := fmt.Sprintf("CARD_SIM_%06d", cardIdx)

				station := stations[rng.Intn(len(stations))]
				dir := rng.Intn(2) + 1 // 1: gidiş, 2: dönüş

				// Sırasız ve gecikmeli olay simülasyonu (-10 dk ile şimdi arası)
				delaySeconds := rng.Intn(600)
				tappedAt := time.Now().Add(-time.Duration(delaySeconds) * time.Second)

				batch[i] = TapPayload{
					ValidatorID: valID,
					SequenceNo:  seq,
					CardID:      cardID,
					LineID:      lineID,
					StationID:   station,
					Direction:   dir,
					TappedAt:    tappedAt,
				}
			}

			// Simülatörden backend'e toplu gönder
			go sendBatch(client, cfg.TargetURL, batch)
		}
	}
}

func waitForBackend(client *http.Client, baseURL string) {
	healthURL := baseURL + "/health"
	slog.Info("Backend bağlantısı bekleniyor...", "url", healthURL)

	for i := 0; i < 30; i++ {
		resp, err := client.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			slog.Info("Backend API hazır!")
			return
		}
		time.Sleep(1 * time.Second)
	}
	slog.Warn("Backend sağlık kontrolüne ulaşılamadı, veri üretimi yine de başlatılıyor.")
}

func sendBatch(client *http.Client, baseURL string, batch []TapPayload) {
	url := baseURL + "/api/v1/validators/batch-sync"
	bodyBytes, err := json.Marshal(batch)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		slog.Debug("Paket senkronize edildi", "validator", batch[0].ValidatorID, "size", len(batch))
	} else {
		slog.Warn("Paket senkronizasyonu başarısız", "status", resp.StatusCode, "validator", batch[0].ValidatorID)
	}
}
