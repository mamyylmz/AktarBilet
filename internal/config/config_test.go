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
