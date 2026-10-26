package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	appsync "github.com/mamyylmz/AktarBilet/internal/sync"
)

type SimulatorConfig struct {
	Seed          int64
	TotalCards    int
	TotalLines    int
	TargetURL     string
	IntervalMs    int
	BatchSize     int
	QueueCap      int
	WorkerCount   int
	TelemetryPort string
	DeviceSecrets map[string]string
	WALPath       string
	SeqPath       string
	ManifestPath  string
	ManifestHash  string
}

type TapPayload struct {
	IdempotencyKey string    `json:"idempotency_key"`
	ValidatorID    string    `json:"validator_id"`
	SequenceNo     int64     `json:"sequence_no"`
	CardID         string    `json:"card_id"`
	LineID         string    `json:"line_id"`
	StationID      string    `json:"station_id"`
	Direction      int       `json:"direction"`
	TappedAt       time.Time `json:"tapped_at"`
}

type TelemetryStats struct {
	EventsGenerated int64
	BatchesSent     int64
	BatchesFailed   int64
	BatchesDropped  int64
	BatchesReplayed int64
	StartTime       time.Time
}

var stats TelemetryStats

type WALEntry struct {
	BatchID   string       `json:"batch_id"`
	Status    string       `json:"status"` // "PENDING" | "ACKED"
	Validator string       `json:"validator"`
	Payload   []TapPayload `json:"payload,omitempty"`
	Timestamp int64        `json:"timestamp"`
	Tick      int64        `json:"tick,omitempty"`
}

type WALJournal struct {
	mu   sync.Mutex
	file *os.File
	path string
}

func OpenWAL(path string) (*WALJournal, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	return &WALJournal{file: f, path: path}, nil
}

func (w *WALJournal) AppendBatch(batchID, valID string, tick int64, batch []TapPayload) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	entry := WALEntry{
		BatchID:   batchID,
		Status:    "PENDING",
		Validator: valID,
		Payload:   batch,
		Timestamp: time.Now().Unix(),
		Tick:      tick,
	}
	b, _ := json.Marshal(entry)
	b = append(b, '\n')
	if _, err := w.file.Write(b); err != nil {
		return err
	}
	return w.file.Sync()
}

func (w *WALJournal) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}

func (w *WALJournal) RecoverState(state *SimulatorState) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.file.Sync(); err != nil {
		return err
	}
	f, err := os.Open(w.path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry WALEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil || entry.Status != "PENDING" {
			continue
		}
		for _, event := range entry.Payload {
			if event.SequenceNo > state.Sequences[event.ValidatorID] {
				state.Sequences[event.ValidatorID] = event.SequenceNo
			}
		}
		if entry.Tick > state.TickCount {
			state.TickCount = entry.Tick
		}
	}
	return scanner.Err()
}

func (w *WALJournal) AckBatch(batchID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	entry := WALEntry{
		BatchID:   batchID,
		Status:    "ACKED",
		Timestamp: time.Now().Unix(),
	}
	b, _ := json.Marshal(entry)
	b = append(b, '\n')
	if _, err := w.file.Write(b); err != nil {
		return err
	}
	return w.file.Sync()
}

func (w *WALJournal) ReplayPending(client *http.Client, baseURL string, deviceSecrets map[string]string) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.file.Sync()

	f, err := os.Open(w.path)
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	pending := make(map[string][]TapPayload)
	for scanner.Scan() {
		var entry WALEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
			if entry.Status == "PENDING" {
				pending[entry.BatchID] = entry.Payload
			} else if entry.Status == "ACKED" {
				delete(pending, entry.BatchID)
			}
		}
	}

	type sortableBatch struct {
		batchID string
		payload []TapPayload
		valID   string
		minSeq  int64
	}

	var batchList []sortableBatch
	for bID, batch := range pending {
		if len(batch) > 0 {
			valID := batch[0].ValidatorID
			var minSeq int64 = batch[0].SequenceNo
			for _, item := range batch {
				if item.SequenceNo < minSeq {
					minSeq = item.SequenceNo
				}
			}
			batchList = append(batchList, sortableBatch{
				batchID: bID,
				payload: batch,
				valID:   valID,
				minSeq:  minSeq,
			})
		}
	}

	// Sıralı replay: önce validatör bazında, sonra artan sequence_no bazında deterministik sırala
	sort.Slice(batchList, func(i, j int) bool {
		if batchList[i].valID != batchList[j].valID {
			return batchList[i].valID < batchList[j].valID
		}
		return batchList[i].minSeq < batchList[j].minSeq
	})

	replayed := 0
	for _, item := range batchList {
		if sendBatchWithRetryDirect(client, baseURL, item.payload, deviceSecrets) {
			replayed++
			ackEntry := WALEntry{BatchID: item.batchID, Status: "ACKED", Timestamp: time.Now().Unix()}
			b, _ := json.Marshal(ackEntry)
			b = append(b, '\n')
			_, _ = w.file.Write(b)
			_ = w.file.Sync()
		}
	}
	return replayed
}

type SimulatorState struct {
	TickCount int64            `json:"tick_count"`
	Sequences map[string]int64 `json:"sequences"`
}

func loadSimulatorState(path string) SimulatorState {
	state := SimulatorState{Sequences: make(map[string]int64)}
	data, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(data, &state); err != nil || state.Sequences == nil {
			// Backwards-compatible read of the former sequence-only format.
			state = SimulatorState{Sequences: make(map[string]int64)}
			_ = json.Unmarshal(data, &state.Sequences)
		}
	}
	return state
}

func saveSimulatorState(path string, state SimulatorState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

type SimulatorManifest struct {
	GeneratorVersion string   `json:"generator_version"`
	Seed             int64    `json:"seed"`
	TotalCards       int      `json:"total_cards"`
	TotalLines       int      `json:"total_lines"`
	BatchSize        int      `json:"batch_size"`
	BaseClockUTC     string   `json:"base_clock_utc"`
	Validators       []string `json:"validators"`
}

func writeManifest(path string, cfg SimulatorConfig) (string, error) {
	manifest := SimulatorManifest{
		GeneratorVersion: "aktarbilet-simulator-v2",
		Seed:             cfg.Seed,
		TotalCards:       cfg.TotalCards,
		TotalLines:       cfg.TotalLines,
		BatchSize:        cfg.BatchSize,
		BaseClockUTC:     "2026-10-24T06:00:00Z",
		Validators:       []string{"VAL_500T_01", "VAL_14ES_01", "VAL_T1_01", "VAL_VAPUR_01"},
	}
	canonical, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	hash := hex.EncodeToString(digest[:])
	envelope := struct {
		SimulatorManifest
		SHA256 string `json:"sha256"`
	}{SimulatorManifest: manifest, SHA256: hash}
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return "", err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return "", err
	}
	return hash, nil
}

func computeHMAC(key []byte, message string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

type BatchJob struct {
	BatchID string
	Payload []TapPayload
}

func main() {
	var cfg SimulatorConfig
	defaultDataDir := "/var/lib/aktarbilet-simulator"
	if _, err := os.Stat(defaultDataDir); err != nil {
		_ = os.MkdirAll(defaultDataDir, 0755)
	}
	defaultWAL := filepath.Join(defaultDataDir, "simulator_wal.jsonl")
	defaultSeq := filepath.Join(defaultDataDir, "simulator_seq.json")
	defaultManifest := filepath.Join(defaultDataDir, "simulator_manifest.json")
	if _, err := os.Stat(defaultDataDir); err != nil {
		defaultWAL = "/tmp/simulator_wal.jsonl"
		defaultSeq = "/tmp/simulator_seq.json"
		defaultManifest = "/tmp/simulator_manifest.json"
	}

	flag.Int64Var(&cfg.Seed, "seed", 42, "Deterministik tohum (Aynı tohum = aynı veri)")
	flag.IntVar(&cfg.TotalCards, "cards", 200000, "Toplam simüle edilecek kart sayısı")
	flag.IntVar(&cfg.TotalLines, "lines", 40, "Toplam simüle edilecek hat sayısı")
	flag.StringVar(&cfg.TargetURL, "target-url", "http://localhost:8080", "AktarBilet Backend API adresi")
	flag.IntVar(&cfg.IntervalMs, "interval", 1000, "Olay üretim sıklığı (ms)")
	flag.IntVar(&cfg.BatchSize, "batch-size", 10, "Çevrimdışı araç paket boyutu")
	flag.IntVar(&cfg.QueueCap, "queue-cap", 100, "Kuyruk kapasitesi (Bounded queue)")
	flag.IntVar(&cfg.WorkerCount, "workers", 3, "Eşzamanlı işçi (worker) sayısı")
	flag.StringVar(&cfg.TelemetryPort, "telemetry-port", "8082", "Telemetri durum sunucu portu")
	flag.StringVar(&cfg.WALPath, "wal-path", defaultWAL, "Disk-backed WAL dosya yolu")
	flag.StringVar(&cfg.SeqPath, "seq-path", defaultSeq, "Kalıcı sequence sayaç yolu")
	flag.StringVar(&cfg.ManifestPath, "manifest-path", defaultManifest, "Deterministik üretim manifesti yolu")
	flag.Parse()

	if envURL := os.Getenv("BACKEND_URL"); envURL != "" {
		cfg.TargetURL = envURL
	}
	if envPort := os.Getenv("TELEMETRY_PORT"); envPort != "" {
		cfg.TelemetryPort = envPort
	}
	credentialsFile := os.Getenv("VALIDATOR_CREDENTIALS_FILE")
	deviceSecrets, err := appsync.LoadDeviceCredentials(credentialsFile)
	if err != nil {
		slog.Error("Validatör cihaz kimlik bilgileri yüklenemedi", "error", err)
		os.Exit(1)
	}
	cfg.DeviceSecrets = deviceSecrets
	if envWAL := os.Getenv("SIMULATOR_WAL_PATH"); envWAL != "" {
		cfg.WALPath = envWAL
	}
	if envSeq := os.Getenv("SIMULATOR_SEQ_PATH"); envSeq != "" {
		cfg.SeqPath = envSeq
	}
	if envManifest := os.Getenv("SIMULATOR_MANIFEST_PATH"); envManifest != "" {
		cfg.ManifestPath = envManifest
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	stats.StartTime = time.Now()
	manifestHash, err := writeManifest(cfg.ManifestPath, cfg)
	if err != nil {
		slog.Error("Deterministik simülatör manifesti yazılamadı", "error", err)
		os.Exit(1)
	}
	cfg.ManifestHash = manifestHash

	slog.Info("AktarBilet Çevrimdışı Araç ve Biniş Simülatörü (Clause 8 Deterministic Engine)",
		"seed", cfg.Seed,
		"cards", cfg.TotalCards,
		"lines", cfg.TotalLines,
		"target", cfg.TargetURL,
		"workers", cfg.WorkerCount,
		"queue_capacity", cfg.QueueCap,
		"wal_path", cfg.WALPath,
		"manifest_hash", cfg.ManifestHash,
	)

	// Bounded channel for batch queue
	batchQueue := make(chan BatchJob, cfg.QueueCap)

	// Başlangıç telemetri HTTP sunucusu
	go startTelemetryServer(cfg.TelemetryPort, &cfg, batchQueue)

	type SeedValidatorRef struct {
		ID        string
		LineID    string
		StationID string
	}

	seedValidators := []SeedValidatorRef{
		{ID: "VAL_500T_01", LineID: "LINE_500T", StationID: "STAT_AYRILIK"},
		{ID: "VAL_14ES_01", LineID: "LINE_14ES", StationID: "STAT_KADIKOY"},
		{ID: "VAL_T1_01", LineID: "LINE_T1", StationID: "STAT_KABATAS"},
		{ID: "VAL_VAPUR_01", LineID: "LINE_VAPUR_KE", StationID: "STAT_KADIKOY"},
	}

	client := &http.Client{Timeout: 6 * time.Second}

	// Backend'in hazır olmasını bekle
	waitForBackend(client, cfg.TargetURL)

	// Open Disk-backed WAL Journal
	wal, err := OpenWAL(cfg.WALPath)
	if err != nil {
		slog.Error("Kalıcı WAL açılamadı; olay kaybını önlemek için simülatör durduruluyor", "error", err)
		os.Exit(1)
	}
	state := loadSimulatorState(cfg.SeqPath)
	if err := wal.RecoverState(&state); err != nil {
		slog.Error("WAL durum uzlaşması başarısız", "error", err)
		os.Exit(1)
	}
	if err := saveSimulatorState(cfg.SeqPath, state); err != nil {
		slog.Error("Simülatör durumu kalıcılaştırılamadı", "error", err)
		os.Exit(1)
	}

	// Crash/Restart Recovery: Replay unacked batches from WAL
	if wal != nil {
		replayed := wal.ReplayPending(client, cfg.TargetURL, cfg.DeviceSecrets)
		if replayed > 0 {
			atomic.AddInt64(&stats.BatchesReplayed, int64(replayed))
			slog.Info("WAL'dan kurtarılan çevrimdışı paketler başarıyla gönderildi", "replayed_count", replayed)
		}
	}

	validatorSeqCounters := state.Sequences
	var tickCount = state.TickCount

	// Recreate the deterministic PRNG position after a restart. Each completed
	// tick consumes one validator choice and three random values per event.
	rng := rand.New(rand.NewSource(cfg.Seed))
	for tick := int64(0); tick < tickCount; tick++ {
		_ = rng.Intn(len(seedValidators))
		for i := 0; i < cfg.BatchSize; i++ {
			_ = rng.Intn(cfg.TotalCards)
			_ = rng.Intn(2)
			_ = rng.Intn(300)
		}
	}

	// Continuous background retry worker for offline/pending WAL replay
	retryCtx, cancelRetry := context.WithCancel(context.Background())
	go func() {
		retryTicker := time.NewTicker(2 * time.Second)
		defer retryTicker.Stop()
		for {
			select {
			case <-retryCtx.Done():
				return
			case <-retryTicker.C:
				replayed := wal.ReplayPending(client, cfg.TargetURL, cfg.DeviceSecrets)
				if replayed > 0 {
					atomic.AddInt64(&stats.BatchesReplayed, int64(replayed))
					slog.Info("Arka plan WAL yeniden deneme başarılı", "replayed_count", replayed)
				}
			}
		}
	}()

	// Worker Pool başlat with WaitGroup for graceful drain
	var workerWg sync.WaitGroup
	for w := 1; w <= cfg.WorkerCount; w++ {
		workerWg.Add(1)
		go func(workerID int) {
			defer workerWg.Done()
			batchWorker(workerID, client, cfg.TargetURL, cfg.DeviceSecrets, batchQueue, wal)
		}(w)
	}

	ticker := time.NewTicker(time.Duration(cfg.IntervalMs) * time.Millisecond)
	defer ticker.Stop()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	baseEpoch := time.Date(2026, 10, 24, 6, 0, 0, 0, time.UTC)
	slog.Info("Canlı biniş simülasyonu çalışıyor...")

	for {
		select {
		case <-shutdown:
			slog.Info("Kapatma sinyali alındı, kuyruk boşaltılıyor (Graceful Drain)...")
			cancelRetry()
			close(batchQueue)
			workerWg.Wait()
			_ = saveSimulatorState(cfg.SeqPath, SimulatorState{TickCount: tickCount, Sequences: validatorSeqCounters})
			_ = wal.Close()
			slog.Info("Simülatör tüm işleri tamamlayarak güvenle durduruldu.")
			return
		case <-ticker.C:
			tickCount++
			valRef := seedValidators[rng.Intn(len(seedValidators))]
			batch := make([]TapPayload, cfg.BatchSize)

			for i := 0; i < cfg.BatchSize; i++ {
				validatorSeqCounters[valRef.ID]++
				seq := validatorSeqCounters[valRef.ID]

				cardIdx := rng.Intn(cfg.TotalCards) + 1
				cardID := fmt.Sprintf("CARD_SIM_%06d", cardIdx)
				if cardIdx == 1 {
					cardID = "CARD_YOLCU_001"
				}

				dir := rng.Intn(2) + 1

				eventOffsetSec := tickCount*60 + int64(rng.Intn(300))
				eventTime := baseEpoch.Add(time.Duration(eventOffsetSec) * time.Second)

				batch[i] = TapPayload{
					IdempotencyKey: fmt.Sprintf("SIM:%s:%d", valRef.ID, seq),
					ValidatorID:    valRef.ID,
					SequenceNo:     seq,
					CardID:         cardID,
					LineID:         valRef.LineID,
					StationID:      valRef.StationID,
					Direction:      dir,
					TappedAt:       eventTime,
				}
			}

			batchBytes, _ := json.Marshal(batch)
			batchDigest := sha256.Sum256(batchBytes)
			batchID := fmt.Sprintf("BATCH-%s", hex.EncodeToString(batchDigest[:12]))

			// Write to WAL before enqueueing
			if err := wal.AppendBatch(batchID, valRef.ID, tickCount, batch); err != nil {
				slog.Error("WAL yazımı başarısız; yeni olay üretimi durduruluyor", "error", err)
				cancelRetry()
				close(batchQueue)
				workerWg.Wait()
				_ = wal.Close()
				return
			}
			atomic.AddInt64(&stats.EventsGenerated, int64(len(batch)))

			// Save persistent sequence state
			if err := saveSimulatorState(cfg.SeqPath, SimulatorState{TickCount: tickCount, Sequences: validatorSeqCounters}); err != nil {
				slog.Error("Sequence durumu yazılamadı; WAL kurtarması kullanılacak", "error", err)
			}

			job := BatchJob{BatchID: batchID, Payload: batch}

			select {
			case batchQueue <- job:
				// Enqueued
			default:
				atomic.AddInt64(&stats.BatchesDropped, 1)
				slog.Warn("Simülatör kuyruğu dolu, paket atlandı (Graceful backoff)",
					"queue_cap", cfg.QueueCap,
					"dropped", atomic.LoadInt64(&stats.BatchesDropped),
				)
			}
		}
	}
}

func batchWorker(id int, client *http.Client, baseURL string, deviceSecrets map[string]string, queue <-chan BatchJob, wal *WALJournal) {
	for job := range queue {
		success := sendBatchWithRetryDirect(client, baseURL, job.Payload, deviceSecrets)
		if success && wal != nil {
			_ = wal.AckBatch(job.BatchID)
		}
	}
}

func sendBatchWithRetryDirect(client *http.Client, baseURL string, batch []TapPayload, deviceSecrets map[string]string) bool {
	if len(batch) == 0 {
		return true
	}
	url := baseURL + "/api/v1/validators/batch-sync"
	bodyBytes, err := json.Marshal(batch)
	if err != nil {
		atomic.AddInt64(&stats.BatchesFailed, 1)
		return false
	}

	valID := batch[0].ValidatorID
	nowTs := time.Now().Unix()
	var maxSeq int64 = 0
	for _, p := range batch {
		if p.SequenceNo > maxSeq {
			maxSeq = p.SequenceNo
		}
	}
	nonce := fmt.Sprintf("sim-%s-%d-%d", valID, batch[0].SequenceNo, maxSeq)

	bodyHash := appsync.ComputePayloadHash(bodyBytes)
	deviceSecret, ok := deviceSecrets[valID]
	if !ok {
		atomic.AddInt64(&stats.BatchesFailed, 1)
		return false
	}
	message := fmt.Sprintf("%s:%d:%d:%s:%s", valID, maxSeq, nowTs, nonce, bodyHash)
	sig := computeHMAC([]byte(deviceSecret), message)

	maxRetries := 2
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			atomic.AddInt64(&stats.BatchesFailed, 1)
			return false
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Validator-ID", valID)
		req.Header.Set("X-Device-Signature", sig)
		req.Header.Set("X-Device-Timestamp", fmt.Sprintf("%d", nowTs))
		req.Header.Set("X-Device-Nonce", nonce)

		resp, err := client.Do(req)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				atomic.AddInt64(&stats.BatchesSent, 1)
				return true
			}
			resp.Body.Close()
		}

		if attempt < maxRetries {
			time.Sleep(time.Duration(100*(1<<attempt)) * time.Millisecond)
		}
	}
	atomic.AddInt64(&stats.BatchesFailed, 1)
	return false
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

func startTelemetryServer(port string, cfg *SimulatorConfig, queue chan BatchJob) {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		uptime := time.Since(stats.StartTime).Seconds()
		payload := map[string]interface{}{
			"service":          "AktarBilet Clause 8 Simulator",
			"status":           "UP",
			"seed":             cfg.Seed,
			"events_generated": atomic.LoadInt64(&stats.EventsGenerated),
			"batches_sent":     atomic.LoadInt64(&stats.BatchesSent),
			"batches_failed":   atomic.LoadInt64(&stats.BatchesFailed),
			"batches_dropped":  atomic.LoadInt64(&stats.BatchesDropped),
			"batches_replayed": atomic.LoadInt64(&stats.BatchesReplayed),
			"queue_length":     len(queue),
			"queue_capacity":   cfg.QueueCap,
			"uptime_seconds":   uptime,
			"manifest_hash":    cfg.ManifestHash,
		}
		_ = json.NewEncoder(w).Encode(payload)
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	_ = server.ListenAndServe()
}
