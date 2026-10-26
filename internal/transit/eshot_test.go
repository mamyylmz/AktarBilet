package transit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTransitService_NormalizationAndDeduplication(t *testing.T) {
	// Mock ESHOT API server
	mockResponse := `{
		"HataMesaj": "",
		"HataVarMi": false,
		"HatOtobusKonumlari": [
			{"OtobusId": 22021, "Yon": 0, "KoorX": "38,47593167", "KoorY": "27,10214333"},
			{"OtobusId": 22021, "Yon": 0, "KoorX": "38,47600000", "KoorY": "27,10220000"},
			{"OtobusId": 22019, "Yon": 1, "KoorX": "38,45871667", "KoorY": "27,21478667"},
			{"OtobusId": 99999, "Yon": 0, "KoorX": "0", "KoorY": "0"},
			{"OtobusId": 88888, "Yon": 0, "KoorX": "41,0082", "KoorY": "28,9784"}
		]
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer ts.Close()

	svc := NewTransitService(ts.Client())
	svc.SetBaseURL(ts.URL)

	ctx := context.Background()
	res, err := svc.GetLiveBuses(ctx, "800")
	if err != nil {
		t.Fatalf("GetLiveBuses failed: %v", err)
	}

	if res.LineNo != "800" {
		t.Errorf("expected line 800, got %s", res.LineNo)
	}

	// 22021 duplicate should be merged to 1
	// 99999 (0,0) should be filtered
	// 88888 (Istanbul coordinates) should be filtered out of Izmir bbox
	// Only 22021 and 22019 should remain
	if len(res.Buses) != 2 {
		t.Fatalf("expected 2 normalized buses, got %d", len(res.Buses))
	}

	bus1 := res.Buses[0]
	if bus1.BusID != 22021 {
		t.Errorf("expected bus 22021, got %d", bus1.BusID)
	}
	if bus1.Latitude < 38.47 || bus1.Latitude > 38.48 {
		t.Errorf("unexpected latitude: %f", bus1.Latitude)
	}
	if bus1.Longitude < 27.10 || bus1.Longitude > 27.11 {
		t.Errorf("unexpected longitude: %f", bus1.Longitude)
	}

	// Verify Attribution and License
	if res.Attribution != ESHOTAttribution {
		t.Errorf("missing or incorrect attribution: %s", res.Attribution)
	}
	if res.License != ESHOTLicense {
		t.Errorf("missing or incorrect license: %s", res.License)
	}
}

func TestTransitService_CachingAndStaleFallback(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"HataVarMi":false,"HatOtobusKonumlari":[{"OtobusId":123,"Yon":0,"KoorX":"38,42","KoorY":"27,14"}]}`))
			return
		}
		// Subsequent calls simulate upstream 503 error
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	svc := NewTransitService(ts.Client())
	svc.SetBaseURL(ts.URL)
	svc.cacheTTL = 50 * time.Millisecond // short TTL for test

	ctx := context.Background()

	// 1. First call succeeds
	res1, err := svc.GetLiveBuses(ctx, "470")
	if err != nil || len(res1.Buses) != 1 {
		t.Fatalf("first call failed: %v", err)
	}
	if res1.IsStale {
		t.Errorf("first response should not be stale")
	}

	// 2. Call within TTL returns from cache without extra HTTP call
	res2, err := svc.GetLiveBuses(ctx, "470")
	if err != nil || len(res2.Buses) != 1 {
		t.Fatalf("cache call failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", callCount)
	}

	// 3. Wait for TTL to expire, call again when upstream returns 503 -> should return stale cache gracefully
	time.Sleep(60 * time.Millisecond)
	res3, err := svc.GetLiveBuses(ctx, "470")
	if err != nil {
		t.Fatalf("stale fallback failed: %v", err)
	}
	if !res3.IsStale {
		t.Errorf("expected response to be marked as stale")
	}
	if len(res3.Buses) != 1 {
		t.Errorf("expected 1 cached bus from stale fallback, got %d", len(res3.Buses))
	}
}

func TestTransitService_RateLimitRetryAfter(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"HataVarMi":true,"HataMesaj":"Rate limit exceeded"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"HataVarMi":false,"HatOtobusKonumlari":[{"OtobusId":35001,"Yon":0,"KoorX":"38,41","KoorY":"27,13"}]}`))
	}))
	defer ts.Close()

	svc := NewTransitService(ts.Client())
	svc.SetBaseURL(ts.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := svc.GetLiveBuses(ctx, "35")
	if err != nil {
		t.Fatalf("expected successful retry after 429, got error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls (1 rate limit + 1 success), got: %d", callCount)
	}

	if len(res.Buses) != 1 || res.Buses[0].BusID != 35001 {
		t.Errorf("expected bus 35001 after retry, got: %+v", res.Buses)
	}
}
