# AktarBilet Nihai Bağımsız Denetim

**Denetçi:** Codex — Principal SDET / Security / Performance / Release Gate

**Tarih:** 26 Ekim 2026, Europe/Istanbul

**Kapsam:** Yerel, commit edilmemiş çalışma ağacının `proje.md` ve final kabul talebine göre bağımsız doğrulaması
**Nihai karar:** **NOT_READY — NOT_PUSHED**

## Bulgular — P0

### AUD-P0-01 — Mandatory regression 25 gerçek ürün hatasıyla başarısız

- **Dosya/satır:** `internal/api/handlers.go:258-500`, `internal/api/handlers.go:601-728`, `internal/api/router.go:65-95`, `internal/sync/service.go:103-283`, `cmd/server/main.go:45-54`.
- **Tekrar üretim:** `cd qa && QA_PROJECT_NAME=aktarbilet-codex-rc2 PLAYWRIGHT_JSON_OUTPUT_FILE=artifacts/codex/playwright-final.json npx playwright test --reporter=list,json,junit`.
- **Beklenen:** Zorunlu API/DB/RBAC/UAT testlerinde sıfır failure.
- **Gerçek:** 102 test; 77 PASS, 25 FAIL, 0 skipped, 116,4 sn.
- **Kanıt:** `qa/artifacts/codex/playwright-final.json` ve `.xml`; hata alanları health fail-open, anomaly FK, direction constraint, audit, out-of-order pricing, settlement/storno, RBAC/IDOR, validator auth, refresh token ve üç UI finans akışı.
- **Minimal düzeltme:** `CODEX_SECURITY_FINDINGS.md` P0/P1 maddelerini kapat; aynı izole test paketi sıfır failure olana kadar release talep etme.

### AUD-P0-02 — Section 8 simülatörü gerçek çalışmada event teslim edemiyor

- **Dosya/satır:** `cmd/simulator/main.go:85-90`, `cmd/simulator/main.go:94-113`, `cmd/simulator/main.go:142-249`.
- **Tekrar üretim:** QA Compose’u `--profile simulator` ile başlat; `http://simulator:8082/status` ve `jw_validator_events` sayısını karşılaştır.
- **Beklenen:** 200.000 kart/40 hatla geçerli referanslı deterministik event, offline kalıcı queue, restart flush, duplicate/out-of-order güvenliği ve safe shutdown.
- **Gerçek:** Seed 200.025 kart/40 hat fakat yalnız 4 validatör. Runtime 50 generated, 0 sent, 5 failed; DB 66→66. Queue yalnız RAM channel; full durumda drop; restart persistence, manifest ve shutdown drain yok.
- **Kanıt:** `qa/artifacts/codex/simulator-runtime.json`.
- **Minimal düzeltme:** Seed ile tek referans manifesti; kalıcı WAL/SQLite queue; item-level ack/retry/DLQ; manifest command; restart/partition/invariant testleri.

### AUD-P0-03 — Validatör ve finansal güvenlik kapısı başarısız

- **Dosya/satır:** `internal/api/handlers.go:368-500`, `internal/sync/service.go:214-283`, `internal/ledger/ledger.go:302-383`.
- **Tekrar üretim:** `cd qa && npx playwright test security/go_live.security.spec.js security/final_rbac.audit.spec.js db/go_live.db.spec.js`.
- **Beklenen:** Kriptografik cihaz kimliği, tam rollback, IDOR/RBAC ayrımı ve deterministik finans.
- **Gerçek:** Prefix auth, passenger-JWT bypass, partial mutation, recharge IDOR, refresh-token confusion ve out-of-order farkı var.
- **Kanıt:** `CODEX_SECURITY_FINDINGS.md`; Playwright JSON.
- **Minimal düzeltme:** Güvenlik raporundaki P0-01…P0-07 maddelerini tek tek kapat.

## Bulgular — P1

### AUD-P1-01 — “Canlı harita” MapLibre/PMTiles değil

- **Dosya/satır:** `web/package.json:1-18`, `web/src/App.jsx:824-999`, `web/index.html:8-10`.
- **Tekrar üretim:** `rg -n 'maplibre|pmtiles|<svg|fonts.googleapis' web` ve browser network kaydı.
- **Beklenen:** MapLibre render, İzmir PMTiles, Range request, self-hosted style/font/sprite, OSM attribution, GTFS route/stop hizası.
- **Gerçek:** React içinde statik SVG koordinat projeksiyonu var; MapLibre/PMTiles bağımlılığı ve tile/Range/GTFS yok. Google Fonts dış istekleri gözlendi.
- **Kanıt:** `qa/artifacts/codex/frontend-navigation.json`; `web/src/App.jsx:885-898`.
- **Minimal düzeltme:** Lisanslı İzmir bölgesel PMTiles + MapLibre GL JS; nginx Range testi; self-hosted font/style/sprite; OSM/ESHOT attribution; GTFS katmanı ve erişilebilir fallback.

### AUD-P1-02 — Cost analytics sahte/etiketsiz varsayımlar üretiyor; UI kontratı kırık

- **Dosya/satır:** `internal/api/handlers.go:978-1069`, `internal/api/handlers.go:1073-1145`, `web/src/App.jsx:169-182`, `web/src/App.jsx:418-438`, `web/src/App.jsx:1060-1138`.
- **Tekrar üretim:** Boş DB’de `GET /api/v1/reports/line-financials`; sonra UI’dan headway scenario isteği ve response alanlarını izle.
- **Beklenen:** Gerçek operator/mode/line/hourly data; measured/assumed etiketi; eksik veride açık hata/empty; kuruş tutarlılığı; kapasite/minimum-service constraints ve doğru UI/API schema.
- **Gerçek:** Boş veride dört hardcoded kârlı hat dönüyor. Maliyetler `vKm*2500`, sabit crew ve varsayılan discount ile uyduruluyor; measured/assumed ayrımı yok. `float64` finans alanları var. UI `target_headway_mins/peak_hour` gönderiyor, backend farklı alanlar bekliyor; UI response’ta var olmayan alanları okuyor. `fleet_capacity_per_vehicle` hesapta kullanılmıyor; onay kaydı yok.
- **Kanıt:** Kaynak incelemesi; `GetLineFinancials` coverage yalnız %18,5.
- **Minimal düzeltme:** Provenance’lı ölçüm/varsayım modeli; missing-data 422/empty; integer/rational para; OpenAPI ile tek schema; talep, minimum sefer ve kapasite constraints; immutable scenario approval kaydı.

### AUD-P1-03 — `make verify` 14/14 false-green üretiyor

- **Dosya/satır:** `cmd/verify/main.go:138-461`.
- **Tekrar üretim:** Temiz seed’de resmi Go 1.23 container ile live verify; ayrıca `BACKEND_URL=http://127.0.0.1:1` negatif koşu.
- **Beklenen:** F-1…F-7/T-1…T-5 gerçek API + persistence + DB invariant doğrulaması; fail’de non-zero.
- **Gerçek:** Offline negatif doğru biçimde exit 1. Live 14/14 exit 0; fakat T-1, T-4, T-5 doğrudan memory engine çağırıyor. F maddeleri şartnamedeki kimliklerle kaymış; F2/F3/F5/F6/F7 çoğunlukla HTTP 200 kontrol ediyor; T3 yalnız ledger body alıyor. Aynı aday bağımsız testte 25 failure verdi.
- **Kanıt:** Verify konsol çıktısı ve `cmd/verify/main.go` kaynak izi.
- **Minimal düzeltme:** Kimlikleri `proje.md` ile eşleştir; her mutation için before/after HTTP+DB assertion, negatif ve rollback; memory-only kontrolleri ayrı unit gate’e taşı.

### AUD-P1-04 — Frontend kritik state’i server-authoritative değil

- **Dosya/satır:** `web/src/App.jsx:4-55`, `web/src/App.jsx:196-224`, `web/src/App.jsx:663-699`, `web/src/App.jsx:715-735`, `web/src/App.jsx:1218-1249`.
- **Tekrar üretim:** `cd qa && npx playwright test tests/browser.uat.spec.js`.
- **Beklenen:** Gerçek login sonrası authoritative kart/ledger/settlement; başarısız istekte görünür hata; refresh sonrası DB state.
- **Gerçek:** Recharge failure görünür hata/state kuralı, settlement refresh persistence ve başarılı recharge sonrası 500→550 TL akışı FAIL. Başlangıç bakiye/ledger ve settlement state localStorage/sabit data içeriyor; demo parolalar hardcoded.
- **Kanıt:** Üç browser testi FAIL ve trace’ler `qa/artifacts/playwright/`.
- **Minimal düzeltme:** Login’den sonra `/auth/me` + server fetch; sabit finans verisini kaldır; yalnız doğrulanmış DB response ile UI başarı/state güncelle.

### AUD-P1-05 — ESHOT 429/retry ve hata semantiği eksik

- **Dosya/satır:** `internal/transit/eshot.go:93-169`, `internal/api/handlers.go:956-968`.
- **Tekrar üretim:** 429 döndüren test server ile `GetLiveBuses`; cold-cache upstream 503/429 çağrısı.
- **Beklenen:** 429 için `Retry-After`/bounded exponential backoff; cold-cache hatasının açık error/unavailable semantiği; supported-line doğrulaması.
- **Gerçek:** Her non-200 tek denemede stale/boş 200’e çevriliyor; retry/backoff ve line allow-list yok. `time.Sleep` request mutex altında ve context-aware değil.
- **Kanıt:** Kaynak; mevcut unit test yalnız 503 stale fallback’i kapsıyor.
- **Minimal düzeltme:** 429/5xx sınıflı retry policy, jitter, context cancellation; cold-cache 503 + structured stale reason; supported line doğrulaması ve cache cardinality limiti.

## Bulgular — P2

### AUD-P2-01 — CI/dokümantasyon adayı yayınlanabilir durumda değil

- **Dosya/satır:** `.github/workflows/ci.yml:1-87`, yeni `internal/sync/device_auth*.go`, yeni `internal/transit/eshot*.go`, repo kökü.
- **Tekrar üretim:** `gofmt -l cmd internal`; `test -f openapi.yaml`; `npm run lint --if-present` ve `web/package.json` incelemesi.
- **Beklenen:** Yerel adayın CI ile aynı format/lint/build kapılarından geçmesi; zorunlu OpenAPI ve dürüst doküman.
- **Gerçek:** Dört yeni Go dosyası gofmt dışı; frontend lint script’i yok; OpenAPI yok. README/AGY raporları başarısız özellikleri PASS/production-ready gösteriyor.
- **Kanıt:** Kaynak/komut çıktıları; uzak CI yalnız eski base SHA için success.
- **Minimal düzeltme:** Gofmt, frontend lint/typecheck, OpenAPI contract test; rapor/README iddialarını gerçek test çıktılarıyla yeniden üret.

## Kesin release adayı

- Branch: `main`.
- Yerel HEAD: `e74c89c957085638d11215d44dd06a9593ef4724`.
- Uzak `origin/main`: aynı SHA.
- Yerel aday dirty: 8 modified üretim dosyası, 4 untracked üretim dosyası, 7 AGY/ESHOT raporu ve Codex QA çıktıları.
- `.env` ignored ve `.env.example` ile byte-identical; >20 MB dosya yok. Repo içinde gerçek API token/private key bulunmadı.
- Mevcut `ddo-*` ana Compose container’ları kullanıcı ortamına ait olduğundan değiştirilmedi; mutasyon ve yük testleri ayrı `aktarbilet-codex-rc2` projesinde yapıldı.

## Bağımsız test özeti

| Alan | Sonuç |
|---|---|
| Go unit/integration | PASS, 64 test fonksiyonu |
| Coverage | PASS, toplam %83,7 |
| Race | PASS, çalıştırılan yollarda 0 race |
| Vet | PASS |
| Gofmt | FAIL, 4 dosya |
| Frontend build | PASS |
| Frontend lint | NOT_TESTED — script yok |
| Ana Compose image build | PASS, 3 image |
| Playwright API/DB/security/browser | FAIL, 77/102 PASS |
| `make verify` offline | PASS (exit 1) |
| `make verify` live | 14/14 gösteriyor fakat semantik gate FAIL |
| Fresh seed/reset-demo | PASS |
| DB interruption/recovery | PASS |
| 7 canary | PASS |
| OpenAPI | FAIL — yok |

## ESHOT canlı veri

Resmî endpoint güvenli aralıkla doğrudan sorgulandı:

- Hat 800: 7 raw / 7 unique, İzmir bounds PASS.
- Hat 470: 9 raw / 3 unique; upstream duplicate normalizasyon ihtiyacı gerçek veriyle doğrulandı.
- Hat 35: geçerli boş liste.
- Türkçe virgüllü koordinat parsing, bounds, dedupe, 25 sn cache ve stale labeling unit testleri PASS.
- ESHOT veri portalı lisansı CC BY 4.0 ve UI atfı doğrulandı: `https://acikveri.bizizmir.com/tr/license`.
- 429 retry/backoff ve gerçek MapLibre/PMTiles kapıları FAIL.

Kanıtlar: `qa/artifacts/codex/eshot_800.json`, `eshot_470.json`, `eshot_35.json`.

## Performans

10 dakika boyunca 180.000 benzersiz event, 300 persisted event/sn, sıfır missing ve sıfır duplicate ledger etkisi ölçüldü. Ancak tümü `NEGATIVE_TOLERANCE`, batch auth güvenliksiz ve 1.000.000-event koşusu NOT_TESTED olduğundan capacity gate FAIL. Ayrıntı: `CODEX_PERFORMANCE_EVIDENCE.md`.

## Git ve CI

- Commit oluşturulmadı.
- Push yapılmadı.
- `origin/main` değiştirilmedi.
- Son GitHub Actions: run `40214469469`, SHA `e74c89c...`, conclusion `success`.
- Yerel dirty adayın CI sonucu **NOT_TESTED**; ayrıca mevcut gofmt sonucu workflow’un format adımını geçmeyeceğini gösteriyor.

## Sonuç

Yedi canary, Go coverage/race/vet, build, seed/reset, DB recovery ve 10 dakikalık ham throughput olumlu. Bunlar; forge edilebilir validatör, kısmi finansal mutation, IDOR/token/RBAC açıkları, sırasız finans hatası, başarısız simülatör, sahte harita, kırık maliyet analitiği ve 25 mandatory regression failure’ını telafi etmez. **VERIFIED_READY kullanılmadı; karar NOT_READY ve NOT_PUSHED.**

