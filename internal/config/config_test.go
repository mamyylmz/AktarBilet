package config

import (
	"os"
	"testing"
)

func TestConfig_LoadDefaults(t *testing.T) {
	cfg := Load()
	if cfg.Port == "" {
		t.Fatalf("Port boş olamaz")
	}
	if cfg.DDOStampHeader != "f85f26" {
		t.Fatalf("DDO Stamp varsayılan f85f26 olmalı, alınan: %s", cfg.DDOStampHeader)
	}
}

func TestConfig_EnvOverrides(t *testing.T) {
	os.Setenv("PORT", "9090")
	defer os.Unsetenv("PORT")

	cfg := Load()
	if cfg.Port != "9090" {
		t.Fatalf("Çevre değişkeni PORT uygulanamadı, alınan: %s", cfg.Port)
	}
}

func TestConfig_GetEnvInt(t *testing.T) {
	// 1. Varsayılan değer
	if val := getEnvInt("NON_EXISTENT_VAR", 42); val != 42 {
		t.Fatalf("Varsayılan değer 42 dönmeli, alınan: %d", val)
	}

	// 2. Set edilmiş int
	os.Setenv("TEST_INT_VAR", "123")
	defer os.Unsetenv("TEST_INT_VAR")
	if val := getEnvInt("TEST_INT_VAR", 42); val != 123 {
		t.Fatalf("Çevre değişkeni 123 dönmeli, alınan: %d", val)
	}

	// 3. Geçersiz int
	os.Setenv("TEST_INVALID_INT", "not-a-number")
	defer os.Unsetenv("TEST_INVALID_INT")
	if val := getEnvInt("TEST_INVALID_INT", 42); val != 42 {
		t.Fatalf("Hatalı değerde varsayılan 42 dönmeli, alınan: %d", val)
	}
}
