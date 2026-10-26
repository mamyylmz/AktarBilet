package goqa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mamyylmz/AktarBilet/internal/transit"
)

func TestRetryAfterSecondsAreHonored(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"HataVarMi":false,"HatOtobusKonumlari":[]}`))
	}))
	defer upstream.Close()

	service := transit.NewTransitService(upstream.Client())
	service.SetBaseURL(upstream.URL)
	started := time.Now()
	_, err := service.GetLiveBuses(context.Background(), "800")
	elapsed := time.Since(started)

	if err != nil {
		t.Fatalf("429 retry failed: %v", err)
	}
	if elapsed < 900*time.Millisecond {
		t.Fatalf("Retry-After: 1 is seconds; retry happened after %s", elapsed)
	}
}

func TestRetryAfterHTTPDateIsHonored(t *testing.T) {
	calls := 0
	retryAt := time.Now().Add(2 * time.Second)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", retryAt.UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"HataVarMi":false,"HatOtobusKonumlari":[]}`))
	}))
	defer upstream.Close()

	service := transit.NewTransitService(upstream.Client())
	service.SetBaseURL(upstream.URL)
	started := time.Now()
	_, err := service.GetLiveBuses(context.Background(), "800")
	elapsed := time.Since(started)

	if err != nil {
		t.Fatalf("HTTP-date 429 retry failed: %v", err)
	}
	if elapsed < 900*time.Millisecond {
		t.Fatalf("Retry-After HTTP-date was not honored; retry happened after %s", elapsed)
	}
}

func TestLongRetryAfterIsContextBounded(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer upstream.Close()

	service := transit.NewTransitService(upstream.Client())
	service.SetBaseURL(upstream.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	started := time.Now()
	response, err := service.GetLiveBuses(ctx, "800")
	elapsed := time.Since(started)

	if err != nil {
		t.Fatalf("graceful degradation unexpectedly returned an error: %v", err)
	}
	if elapsed > time.Second {
		t.Fatalf("long Retry-After ignored context bound: %s", elapsed)
	}
	if response == nil || !response.IsStale {
		t.Fatalf("expected bounded graceful stale response, got %+v", response)
	}
}
