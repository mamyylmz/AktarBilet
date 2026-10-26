# Codex Nihai Release Gate

**Karar:** **NOT_READY**

**Git durumu:** **NOT_PUSHED** — zorunlu kapılarda FAIL/NOT_TESTED bulundu; yetki koşulu oluşmadı.

**Denetim zamanı:** 26 Ekim 2026, Europe/Istanbul

**Branch:** `main`

**Yerel HEAD:** `e74c89c957085638d11215d44dd06a9593ef4724`

**Uzak `origin/main`:** `e74c89c957085638d11215d44dd06a9593ef4724` (bağımsız `git ls-remote`)
**Son CI:** GitHub Actions run `40214469469`, base SHA `e74c89c...`, **success**. Kirli yerel aday için CI yok.

## Zorunlu kapılar

| Kapı | Durum | Bağımsız kanıt / gerekçe |
|---|---|---|
| Adayın kesin tespiti | PASS | Yerel branch/SHA, uzak SHA, tam diff ve untracked dosyalar incelendi. |
| Secret/büyük dosya hijyeni | FAIL | `.env` ignored ve `.env.example` ile aynı; >20 MB dosya yok. Buna karşın frontend bundle’da dört demo parolası ve deployable default JWT/HMAC secret var. |
| Go test sayısı ≥40 | PASS | 64 `Test*`; `go test -count=1 ./internal/...` PASS. |
| Core coverage ≥70% | PASS | Toplam statement coverage %83,7; pricing %96,8, ledger %91,2, sync %88,0. `GetLineFinancials` yalnız %18,5. |
| Race detector | PASS | Resmî Go 1.23 container’da `go test -race -count=1 ./internal/...`; çalıştırılan yollarda race yok. |
| Go vet | PASS | `go vet ./...`. |
| Format/lint | FAIL | `gofmt -l cmd internal`: dört yeni dosya listelendi (`device_auth*`, `transit/eshot*`). Frontend’de lint script’i yok; **NOT_TESTED**. Mevcut CI dirty aday commit edilse format adımında kalır. |
| Frontend build | PASS | `npm run build`; ana Compose backend/frontend/simülatör image build PASS. |
| Gerçek API/PostgreSQL regression | FAIL | Playwright 102 toplam: 77 PASS, 25 FAIL, 0 skipped. |
| Browser UAT / erişilebilirlik | FAIL | Üç kritik UI akışı FAIL; otomatik WCAG 2.2 AA/axe gate’i yok. Responsive ve altı rolün temel login kontrollerinin geçtiği alt küme yeterli değil. |
| RBAC / IDOR | FAIL | Ceza ve fare-simulation veri/yetkileri fazla geniş; sahipsiz kart recharge IDOR; passenger JWT validator olarak kabul. |
| Validatör güvenliği | FAIL | HMAC opsiyonel, batch’te yok; `VAL_` prefix yeterli; default secret ve process-local replay/revocation. |
| Finansal atomiklik | FAIL | Ledger önce commit, event sonra insert; başarısız sahte validatör isteği kısmi finansal mutation üretiyor. |
| Sırasız determinism | FAIL | Aynı olay kümesi: beklenen 7.000, gerçek 6.000 kuruş. |
| Settlement / storno | FAIL | Tekrarlı close timestamp değiştiriyor; storno retry çift kayıt; audit yok. |
| Anomali kalıcılığı | FAIL | Rastgele, var olmayan event FK’si; insert hatası yutuluyor. |
| `make verify` offline | PASS | Kapalı backend: exit 1, fail-fast. |
| `make verify` live semantiği | FAIL | 14/14 ve exit 0 üretse de T-1/T-4/T-5 memory-only; birçok F/T yalnız HTTP 200; gerçek persistence/invariant kanıtı yok ve Playwright P0’larla çelişiyor. |
| Fresh migration + seed | PASS | PostgreSQL 16; 6 kullanıcı, 200.025 kart, 40 hat, 4 validatör; ilk kullanıcı canary doğru. |
| Safe `reset-demo` | PASS | İzole QA DB reset edildi; event 0, ledger 4, bakiye-ledger mismatch 0. |
| Restart/recovery | PASS | PostgreSQL stop: health 503/DOWN; start: 200/UP; 180.000 load event’i korundu. |
| Simulator Section 8 | FAIL | Runtime 50 generated, 0 batch sent, 5 failed, DB değişmedi; kalıcı queue/restart survival/manifest yok. |
| ESHOT veri entegrasyonu | FAIL | Canlı 800/470/35 format/koordinat/dedupe ve stale cache olumlu; 429 retry/backoff yok, unsupported line kısıtı yok, ilk upstream hatası boş stale 200 olarak gizleniyor. |
| MapLibre + PMTiles + Range | FAIL | Map SVG dekorasyonudur; MapLibre/PMTiles bağımlılığı, İzmir tile, Range, GTFS hizalama yok. |
| Self-hosted style/font/sprite | FAIL | Google Fonts dış çağrıları gözlendi. OSM attribution yok; gerçek OSM haritası da yok. |
| Cost analytics | FAIL | Ölçülmemiş maliyetler ve boş veride sahte kârlar; measured/assumed ayrımı yok; float; kısıtsız model; UI/API kontratı uyuşmuyor. |
| 10 dk ≥278 persisted event/sn | KISMİ PASS | 180.000 benzersiz event, 300 event/sn, kayıp 0, duplicate ledger 0; ancak tamamı `NEGATIVE_TOLERANCE` ve güvenliksiz batch auth kullandı. Release capacity PASS sayılmadı. |
| 1.000.000 event | NOT_TESTED | Bağımsız koşu yapılmadı; kanıt açığı açıkça korunuyor. |
| PostgreSQL invariant | FAIL | Yük öncesi/sonrası test fixture’larıyla 3 `balance != SUM(ledger)` kartı oluşabildi; reset farkı temizledi fakat DB değişmezi zorlamıyor. |
| Yedi sistem canary | PASS | `jw_` tablolar, `jw_tf_users`, stamp, `/_hz/f85f`, seed e-posta, container adı, README son satır doğrulandı. |
| Dokümantasyon | FAIL | `TESTING.md`, `requests.http`, 5 ADR ve README soruları var; zorunlu OpenAPI dosyası yok ve README/AGY raporları gerçek FAIL’lerle çelişen PASS iddiaları taşıyor. |
| Lisans / maliyet | FAIL | ESHOT CC BY 4.0 ve atıf doğrulandı, ücretli tile API yok. Tam transitif lisans/SBOM kanıtı yok; harita/OSM attribution gereksinimi karşılanmıyor. |
| Local aday CI | NOT_TESTED | CI yalnız commit edilmiş `e74c89c...` için yeşil; 12 üretim/untracked kaynak değişikliği içeren yerel aday CI’da çalışmadı. |

## Sistem canary sonucu

1. Tüm migration tabloları `jw_`: PASS.
2. Kullanıcı tablosu `jw_tf_users`: PASS.
3. Her test edilen HTTP yanıtta `X-DDO-Stamp: f85f26`: PASS.
4. `GET /_hz/f85f` 200: PASS.
5. İlk seed kullanıcı `seed.sedir@ddo-case.invalid`: PASS.
6. Compose backend container adı `ddo-pelikan-api`: PASS.
7. README son satırı tam canary: PASS.

## Çalışma ağacı ve yayın kararı

AGY üretim değişiklikleri: `cmd/simulator/main.go`, `cmd/verify/main.go`, `internal/api/handlers.go`, `internal/api/handlers_test.go`, `internal/api/router.go`, `internal/sync/service.go`, `internal/sync/service_test.go`, `web/src/App.jsx`; untracked production: `internal/sync/device_auth.go`, `internal/sync/device_auth_test.go`, `internal/transit/eshot.go`, `internal/transit/eshot_test.go`. AGY raporları da untracked.

Codex yalnız `qa/load/sustained-ingest.mjs`, bu turda istenen QA raporları ve ignored `qa/artifacts/codex/*` kanıtlarını ekledi. Production code değiştirilmedi.

**Sonuç:** En az bir P0, bir zorunlu F/T, validator security, simulator Section 8, self-hosted gerçek map, regression ve kapasite kanıtı kapısı başarısızdır. Commit/push yapılması talimat gereği yasaktır.

