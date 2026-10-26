# Codex → Antigravity Düzeltme Talepleri

**Durum:** Release **NOT_READY**. Aşağıdaki işler production sahibi AGY tarafından tamamlanmadan yeniden release denetimi istenmemeli. Codex production code değiştirmedi.

## P0 — FR-01: Validatör güvenliği ve ingest atomikliği

- **Dosya/satır:** `internal/api/handlers.go:359-500`, `internal/sync/device_auth.go:21-135`, `internal/sync/service.go:103-283`, `internal/ledger/ledger.go:295-383`, `cmd/server/main.go:100-102`.
- **Tekrar üretim:** `cd qa && npx playwright test security/go_live.security.spec.js -g 'SEC-05'` ve `security/final_rbac.audit.spec.js -g 'ordinary passenger JWT'`.
- **Beklenen:** Eksik/bozuk/revoked/yanlış atamalı device credential 401/403; sıfır DB etkisi; batch ve single aynı kontrol; tüm finans/event/anomaly/audit tek transaction.
- **Gerçek:** `VAL_` prefix veya passenger JWT yeterli; batch’te HMAC yok; default secret; sahte validatör hatasında event yokken ledger/bakiye değişebiliyor.
- **Kanıt:** `CODEX_SECURITY_FINDINGS.md` SEC-P0-01/02/03; Playwright JSON.
- **Minimal düzeltme:** Zorunlu device-auth middleware; DB-backed cihaz/secret/assignment/revocation/replay; tek `pgx.Tx`; mutation öncesi referans validation; item-level batch error.
- **Kapanış kriteri:** Missing/forged/replayed/revoked/disabled/wrong-line/wrong-station/passenger-JWT testlerinin tamamı geçer; başarısız her istekte before/after DB eşit.

## P0 — FR-02: Deterministik sırasız olay ve ledger değişmezi

- **Dosya/satır:** `internal/sync/service.go:103-286`, `migrations/001_initial_schema.sql:51-79`.
- **Tekrar üretim:** `cd qa && npx playwright test db/go_live.db.spec.js -g 'out-of-order'`; tüm kartlarda balance-vs-ledger sorgusu.
- **Beklenen:** Arrival order’dan bağımsız aynı ücret/bakiye; her kartta bakiye ledger toplamı.
- **Gerçek:** 7.000 beklenen bakiye 6.000 çıktı; DB doğrudan non-ledger opening balance kabul ediyor.
- **Kanıt:** CORE-04 FAIL ve `qa/artifacts/codex/sustained-ingest-db.json`.
- **Minimal düzeltme:** Event-time reconciliation/repricing; ledger source-of-truth; doğrudan balance mutation’ı engelleyen DB sınırı ve transactional audit.
- **Kapanış kriteri:** Tüm permütasyonlarda aynı event fares/balance; 100+ concurrent tap/recharge sonrası global mismatch 0.

## P0 — FR-03: Token purpose, logout, IDOR ve endpoint×role matrisi

- **Dosya/satır:** `internal/auth/auth.go:48-105`, `internal/api/middleware.go:66-160`, `internal/api/handlers.go:282-296`, `internal/api/router.go:71-95`.
- **Tekrar üretim:** `cd qa && npx playwright test security/go_live.security.spec.js security/final_rbac.audit.spec.js`.
- **Beklenen:** Refresh access olarak geçmez; yolcu yalnız kendi kartı; ceza/simulation verisi least privilege; gerçek logout invalidation.
- **Gerçek:** Refresh token access olarak geçiyor; sahipsiz kart recharge ediliyor; penalty/fare simulation fazla role açık; logout revocation yok.
- **Kanıt:** İlgili 13 security FAIL.
- **Minimal düzeltme:** Token type/aud/iss; refresh rotation ve revocation; strict ownership; route bazlı allow-list; test metnine bağlı IDOR bypass’ını kaldır.
- **Kapanış kriteri:** Altı rol × tüm endpoint matrisi, direct URL ve no-unauthorized-DB-mutation paketi sıfır failure.

## P0 — FR-04: Settlement, anomali ve audit kalıcılığı

- **Dosya/satır:** `internal/api/handlers.go:405-413`, `internal/api/handlers.go:601-728`, `migrations/001_initial_schema.sql:129-192`.
- **Tekrar üretim:** `cd qa && npx playwright test db/final_anomaly.audit.spec.js db/go_live.db.spec.js -g 'anomaly|settlement|storno|audit'`.
- **Beklenen:** Gerçek event FK’li anomali; one-way/idempotent close; retry-safe storno; her kritik mutation audit.
- **Gerçek:** Anomali insert hatası yutuluyor; close timestamp yeniden yazılıyor; storno duplicate; audit 0.
- **Kanıt:** Dört DB test failure.
- **Minimal düzeltme:** Event/anomaly tek transaction; conditional close; storno idempotency unique key; old/new audit row’u aynı transaction.
- **Kapanış kriteri:** Retry/restart/concurrency altında tek sonuç; anomali join sayısı 1; audit actor/time/old/new değerleri doğru.

## P0 — FR-05: Section 8 simülatörü

- **Dosya/satır:** `cmd/simulator/main.go:85-294`, `internal/seed/seeder.go:149-157`, `docker-compose.yml:66-77`.
- **Tekrar üretim:** QA Compose simulator profile + `/status` + DB before/after.
- **Beklenen:** Valid seed refs; persistent offline queue; restart survival; reconnect flush; duplicate/out-of-order correctness; manifest determinism; safe drain.
- **Gerçek:** 50 generated / 0 sent / 5 failed; only-memory/drop queue; 6 olmayan validator; manifest yok.
- **Kanıt:** `qa/artifacts/codex/simulator-runtime.json`.
- **Minimal düzeltme:** Tek seed catalog; disk WAL/SQLite; per-event result validation; retry/DLQ; manifest export; waitgroup drain ve provenance.
- **Kapanış kriteri:** Aynı seed/config/clock iki özdeş manifest; network cut + process restart sonrası eksiksiz flush; DB/event/ledger invariant PASS.

## P1 — FR-06: Gerçek self-hosted harita

- **Dosya/satır:** `web/package.json:1-18`, `web/src/App.jsx:824-999`, `web/index.html:8-10`, `Dockerfile.frontend:24-48`.
- **Tekrar üretim:** Browser network + `rg 'maplibre|pmtiles|fonts.googleapis' web`.
- **Beklenen:** MapLibre + İzmir PMTiles + HTTP Range + self-hosted style/font/sprite + OSM/ESHOT atıf + GTFS alignment.
- **Gerçek:** SVG pseudo-map; tile/range/GTFS yok; Google Fonts dış çağrısı.
- **Kanıt:** `qa/artifacts/codex/frontend-navigation.json`.
- **Minimal düzeltme:** Lisans kaynağını belgeleyerek regional PMTiles; Range-enabled nginx; local assets; map load/error/stale/a11y E2E.
- **Kapanış kriteri:** Browser’da canvas/map instance, `.pmtiles` 206 Range, dış ücretli çağrı 0, atıf görünür, GTFS route/stop overlay doğrulanmış.

## P1 — FR-07: ESHOT dayanıklılığı

- **Dosya/satır:** `internal/transit/eshot.go:93-169`.
- **Tekrar üretim:** Mock upstream 429 + `Retry-After`, cold-cache 503 ve cancelled context testleri.
- **Beklenen:** Bounded retry/backoff, context-aware wait, explicit stale/unavailable semantics, line allow-list.
- **Gerçek:** Retry yok; tüm non-200 cold-cache’de boş stale 200; key cardinality açık; mutex altında `Sleep`.
- **Kanıt:** Kaynak ve eksik branch testleri.
- **Minimal düzeltme:** 429/5xx policy + jitter; context timer; stale reason/upstream status; supported line validation.
- **Kapanış kriteri:** 429→retry→success, retry exhaustion, stale cache ve cold-cache failure testleri geçer.

## P1 — FR-08: Maliyet analitiği ve optimizasyon kontratı

- **Dosya/satır:** `internal/api/handlers.go:978-1145`, `web/src/App.jsx:169-182`, `web/src/App.jsx:418-438`, `web/src/App.jsx:1060-1138`.
- **Tekrar üretim:** Boş data financial report; missing/zero/unrealistic/contradictory headway request; UI network response.
- **Beklenen:** Gerçek/varsayımsal veri ayrımı, eksik data negatif yolu, capacity/min-service constraints, deterministik integer finance ve UI schema uyumu.
- **Gerçek:** Hardcoded sahte kâr; etiketsiz formüller; float; invalid input defaults; kapasite parametresi kullanılmıyor; UI request/response alanları backend ile uyuşmuyor.
- **Kanıt:** Kaynak; handler coverage %18,5.
- **Minimal düzeltme:** OpenAPI-first DTO; source/provenance; no-data 422/empty; rational/integer accounting; constraint engine; immutable scenario approval.
- **Kapanış kriteri:** Negatif test matrisi ve UI→HTTP→DB scenario kayıt testi PASS; sahte real-world profit yok.

## P1 — FR-09: `make verify` gerçek acceptance gate olmalı

- **Dosya/satır:** `cmd/verify/main.go:138-461`, `TESTING.md:35-59`.
- **Tekrar üretim:** Live verify 14/14; sonra full Playwright 25 failure.
- **Beklenen:** Aynı kritik kusurlarda verify non-zero.
- **Gerçek:** Memory-only T-1/T-4/T-5 ve HTTP-200 kontrolleri false green üretiyor; F kimlikleri şartnameyle uyumsuz.
- **Kanıt:** Live çıktı ve kaynak call graph.
- **Minimal düzeltme:** Gerçek API+PostgreSQL assertions, doğru F/T eşlemesi, rollback/negative/retry checks; uzun yük ayrı komut.
- **Kapanış kriteri:** Bilinen her P0 testinde verify FAIL; temiz durumda repeatable PASS; ≤15 dakika.

## P1 — FR-10: Frontend authoritative akış ve credential temizliği

- **Dosya/satır:** `web/src/App.jsx:4-55`, `web/src/App.jsx:196-224`, `web/src/App.jsx:663-699`, `web/src/App.jsx:1218-1265`.
- **Tekrar üretim:** `cd qa && npx playwright test tests/browser.uat.spec.js`; bundle secret scan.
- **Beklenen:** Gerçek login/session, gerçek DB state, başarısız istekte error, refresh’te persistence, bundle’da parola yok.
- **Gerçek:** Üç finans UI testi FAIL; localStorage/sabit state ve hardcoded demo passwords var.
- **Kanıt:** Playwright trace/video/screenshots ve source.
- **Minimal düzeltme:** Sabit finans/credential state’i kaldır; response schema/adapters; server-authoritative refresh; güvenli session.
- **Kapanış kriteri:** Altı rol kritik UAT + direct URL + refresh + error/empty/loading + mobile/keyboard/200% zoom + axe sıfır mandatory failure.

## P2 — FR-11: CI, format, lint, OpenAPI ve dürüst raporlama

- **Dosya/satır:** `.github/workflows/ci.yml:1-87`, `web/package.json:1-18`, yeni device/transit Go dosyaları, `README.md`, `qa/reports/AGY_*.md`.
- **Tekrar üretim:** `gofmt -l cmd internal`; `npm run lint --if-present`; OpenAPI file check; full local test.
- **Beklenen:** Dirty aday CI eşdeğerinde yeşil; frontend lint; OpenAPI; raporlar gerçek sonuçlarla uyumlu.
- **Gerçek:** 4 Go dosyası format dışı; lint script’i/OpenAPI yok; AGY raporları 25 regression failure’a rağmen %100 verified iddiasında.
- **Kanıt:** Komut çıktıları; GitHub CI yalnız eski SHA’da success.
- **Minimal düzeltme:** Format; ESLint/typecheck; OpenAPI lint/contract; current evidence ile README/raporları düzelt.
- **Kapanış kriteri:** Temiz worktree adayı için yerel CI eşdeğeri ve push sonrası GitHub Actions aynı SHA’da success.

## Codex yeniden denetim sırası

1. FR-01…FR-05 tamamlanır.
2. İzole fresh seed üzerinde security + financial + simulator testleri sıfır failure çalıştırılır.
3. FR-06…FR-10 ve browser/accessibility doğrulanır.
4. Güvenli pipeline ile 10 dakika ve 1.000.000-event kapasite koşuları yapılır.
5. FR-11 sonrası clean candidate SHA bağımsız release gate’e sunulur.

