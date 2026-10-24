.PHONY: all test verify reset-demo load build up down chaos help

# Varsayılan hedef
all: build test

help:
	@echo "AktarBilet — Kullanılabilir Make Komutları:"
	@echo "  make verify      - Tüm gereksinimleri (F-1..F-7, T-1..T-5) tek komutla test eder"
	@echo "  make test        - Go birim ve entegrasyon testlerini koşturur"
	@echo "  make load        - k6 yük testini başlatır"
	@echo "  make reset-demo  - Veritabanını başlangıç seed durumuna sıfırlar"
	@echo "  make chaos       - Hata enjeksiyonu ve bağımlılık kurtarma senaryosunu dener"
	@echo "  make up          - Docker compose ile tüm servisleri ayağa kaldırır"
	@echo "  make down        - Docker compose servislerini durdurur"

# Otomatik Doğrulama Komutu (Bölüm 10 Zorunlu)
verify:
	@go run ./cmd/verify

# Birim ve Kapsam Testleri
test:
	@go test -v ./internal/...

# Demo Sıfırlama (Bölüm 10 Zorunlu)
reset-demo:
	@bash ./scripts/reset-demo.sh

# k6 Yük Testi (Bölüm 9 Zorunlu)
load:
	@if command -v k6 >/dev/null 2>&1; then \
		k6 run ./scripts/load-test.js; \
	else \
		go test -v -bench=. ./internal/anomaly/...; \
	fi

# Kaos Senaryosu (Bölüm 9 Zorunlu)
chaos:
	@bash ./scripts/chaos.sh

# Derleme
build:
	@go build -o bin/server ./cmd/server
	@go build -o bin/simulator ./cmd/simulator
	@go build -o bin/verify ./cmd/verify

# Docker Compose Komutları
up:
	@docker compose up --build -d

down:
	@docker compose down -v
