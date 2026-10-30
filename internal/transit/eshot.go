package transit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// BusLocation represents a normalized vehicle location from ESHOT telemetry
type BusLocation struct {
	BusID      int       `json:"bus_id"`
	LineNo     string    `json:"line_no"`
	Direction  int       `json:"direction"` // 0: Gidiş, 1: Dönüş
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	ObservedAt time.Time `json:"observed_at"`
}

// LiveBusResponse is the API response model served to AktarBilet clients
type LiveBusResponse struct {
	LineNo      string        `json:"line_no"`
	Buses       []BusLocation `json:"buses"`
	TotalBuses  int           `json:"total_buses"`
	RetrievedAt time.Time     `json:"retrieved_at"`
	IsStale     bool          `json:"is_stale"`
	Attribution string        `json:"attribution"`
	License     string        `json:"license"`
}

// RawESHOTResponse maps the JSON payload returned by openapi.izmir.bel.tr
type RawESHOTResponse struct {
	HataMesaj          string `json:"HataMesaj"`
	HataVarMi          bool   `json:"HataVarMi"`
	HatOtobusKonumlari []struct {
		OtobusId int    `json:"OtobusId"`
		Yon      int    `json:"Yon"`
		KoorX    string `json:"KoorX"` // Enlem (Latitude) - örn: "38,47957333"
		KoorY    string `json:"KoorY"` // Boylam (Longitude) - örn: "27,09299833"
	} `json:"HatOtobusKonumlari"`
}

type cachedLine struct {
	response  *LiveBusResponse
	expiresAt time.Time
}

// TransitService manages live transit telemetry with rate-limited caching
type TransitService struct {
	client       *http.Client
	cache        map[string]*cachedLine
	cacheMu      sync.RWMutex
	cacheTTL     time.Duration
	fetchTimeout time.Duration
	requestMu    sync.Mutex
	lastReqTime  time.Time
	throttleWait time.Duration
	baseURL      string
}

const (
	DefaultESHOTBaseURL = "https://openapi.izmir.bel.tr/api/iztek/hatotobuskonumlari"
	ESHOTAttribution    = "İzmir Büyükşehir Belediyesi ESHOT Genel Müdürlüğü Açık Veri Portalı"
	ESHOTLicense        = "Creative Commons Atıf 4.0 Uluslararası (CC BY 4.0)"
)

// SupportedLines lists verified starter lines
var SupportedLines = []string{"800", "470", "35", "5", "253", "680"}

func NewTransitService(client *http.Client) *TransitService {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &TransitService{
		client:       client,
		cache:        make(map[string]*cachedLine),
		cacheTTL:     25 * time.Second,
		fetchTimeout: 4 * time.Second,
		throttleWait: 1100 * time.Millisecond, // Min 1.1s between outgoing calls to avoid HTTP 429
		baseURL:      DefaultESHOTBaseURL,
	}
}

func (s *TransitService) SetBaseURL(url string) {
	s.requestMu.Lock()
	defer s.requestMu.Unlock()
	s.baseURL = url
}

// GetLiveBuses returns live normalized bus coordinates for the specified line
func (s *TransitService) GetLiveBuses(ctx context.Context, lineNo string) (*LiveBusResponse, error) {
	cleanLine := strings.TrimSpace(lineNo)
	if cleanLine == "" {
		cleanLine = "800"
	}

	// 1. Cache hit check
	s.cacheMu.RLock()
	c, found := s.cache[cleanLine]
	s.cacheMu.RUnlock()

	now := time.Now()
	if found && now.Before(c.expiresAt) {
		return c.response, nil
	}

	// 2. Fetch fresh telemetry with request throttle
	fetchCtx, cancel := context.WithTimeout(ctx, s.fetchTimeout)
	freshResp, err := s.fetchFromESHOT(fetchCtx, cleanLine)
	cancel()
	if err != nil {
		// Graceful degradation: return stale cache if available
		if found && c.response != nil {
			staleCopy := *c.response
			staleCopy.IsStale = true
			return &staleCopy, nil
		}
		// Return empty structured response with graceful indicator
		return &LiveBusResponse{
			LineNo:      cleanLine,
			Buses:       []BusLocation{},
			TotalBuses:  0,
			RetrievedAt: now,
			IsStale:     true,
			Attribution: ESHOTAttribution,
			License:     ESHOTLicense,
		}, nil
	}

	// 3. Save to cache
	s.cacheMu.Lock()
	s.cache[cleanLine] = &cachedLine{
		response:  freshResp,
		expiresAt: now.Add(s.cacheTTL),
	}
	s.cacheMu.Unlock()

	return freshResp, nil
}

func (s *TransitService) fetchFromESHOT(ctx context.Context, lineNo string) (*LiveBusResponse, error) {
	// Throttle outgoing requests
	s.requestMu.Lock()
	elapsed := time.Since(s.lastReqTime)
	if elapsed < s.throttleWait {
		timer := time.NewTimer(s.throttleWait - elapsed)
		select {
		case <-ctx.Done():
			timer.Stop()
			s.requestMu.Unlock()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	s.lastReqTime = time.Now()
	baseURL := s.baseURL
	s.requestMu.Unlock()

	url := fmt.Sprintf("%s/%s", baseURL, lineNo)

	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("istek oluşturulamadı: %w", err)
		}
		req.Header.Set("User-Agent", "AktarBilet-Reliability-Agent/1.0")
		req.Header.Set("Accept", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("eshot api bağlantı hatası: %w", err)
			if attempt < maxRetries {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Duration(50*(1<<attempt)) * time.Millisecond):
				}
				continue
			}
			break
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("eshot api hız sınırı aşıldı (429 Too Many Requests)")
			if attempt < maxRetries {
				backoff := 1 * time.Second
				if ra := resp.Header.Get("Retry-After"); ra != "" {
					if sec, err := strconv.Atoi(ra); err == nil && sec > 0 {
						backoff = time.Duration(sec) * time.Second
					} else if t, err := http.ParseTime(ra); err == nil {
						if d := time.Until(t); d > 0 {
							backoff = d
						}
					}
				}
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
				}
				continue
			}
			break
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("eshot api beklenmeyen durum kodu: %d", resp.StatusCode)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("yanıt okunamadı: %w", err)
		}

		var raw RawESHOTResponse
		if err := json.Unmarshal(bodyBytes, &raw); err != nil {
			return nil, fmt.Errorf("json parse hatası: %w", err)
		}

		retrievedTime := time.Now()
		var buses []BusLocation
		seenIDs := make(map[int]bool)

		for _, item := range raw.HatOtobusKonumlari {
			lat := parseCoordinate(item.KoorX)
			lon := parseCoordinate(item.KoorY)

			// Filter invalid GPS points and enforce İzmir Bounding Box [37.8, 39.3] x [26.2, 28.5]
			if lat == 0 || lon == 0 || lat < 37.8 || lat > 39.3 || lon < 26.2 || lon > 28.5 {
				continue
			}

			// Deduplicate repeated bus IDs in same payload
			if seenIDs[item.OtobusId] {
				continue
			}
			seenIDs[item.OtobusId] = true

			buses = append(buses, BusLocation{
				BusID:      item.OtobusId,
				LineNo:     lineNo,
				Direction:  item.Yon,
				Latitude:   lat,
				Longitude:  lon,
				ObservedAt: retrievedTime,
			})
		}

		return &LiveBusResponse{
			LineNo:      lineNo,
			Buses:       buses,
			TotalBuses:  len(buses),
			RetrievedAt: retrievedTime,
			IsStale:     false,
			Attribution: ESHOTAttribution,
			License:     ESHOTLicense,
		}, nil
	}

	return nil, lastErr
}

func parseCoordinate(s string) float64 {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return 0
	}
	normalized := strings.ReplaceAll(trimmed, ",", ".")
	val, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0
	}
	return val
}
