# Codex Bağımsız Hata Kaydı

Denetim zamanı: 2026-10-25 04:56 TRT  
Yerel kaynak: `main@6ac85910e3cc8312b1759827b87342adc4d08b41` + altı değiştirilmiş üretim dosyası  
Durum: **AÇIK — release blocker mevcut**

## P0

### CX-P0-01 — `make verify` erişilemeyen backend ile 14/14 PASS dönüyor

- Beklenen: `BACKEND_URL=http://127.0.0.1:1 make verify` non-zero ile FAIL.
- Gerçek: exit `0`, 14/14 PASS; canlı koşuda da yalnız SYS-1/SYS-2 gerçek HTTP, diğer 12 madde in-memory.
- Kök neden: `cmd/verify/main.go:48-62` erişim hatasını `runDirectVerifications()` ile başarı yoluna çeviriyor; `cmd/verify/main.go:410-416` canlı koşuda F/T sonuçlarını yine bu fonksiyondan alıyor. In-memory bağımlılıklar `cmd/verify/main.go:94-104`.
- Tekrar üretim: `docker run --rm -v "$PWD":/src:ro -w /src -e BACKEND_URL=http://127.0.0.1:1 golang:1.23 bash -lc 'make verify'`.
- Kanıt: `qa/reports/CODEX_VERIFY_AUDIT.md`.
- Minimal düzeltme: erişilemeyen backend'i hard FAIL yap; F-1..T-5 için gerçek HTTP çağrısı ve PostgreSQL öncesi/sonrası assertion kullan; her madde başarısızlığını non-zero çıkışa bağla.

### CX-P0-02 — Validatör olayı atomik değil; olay insert hatasında finansal etki kalıyor

- Beklenen: kart debiti, sayaçlar ve `jw_validator_events` tek transaction; herhangi bir hata tümünü rollback etmeli.
- Gerçek: sahte `VAL_...` isteği önce kartı/ledger'ı değiştirdi, sonra validator FK insert'i HTTP 500 ile başarısız oldu; kısmi finansal mutasyon kaldı.
- Kök neden: `internal/sync/service.go:211-246` ledger transaction'ını commit ediyor; olay ayrı olarak `internal/sync/service.go:249-280` yazılıyor.
- Tekrar üretim: `env QA_PROJECT_NAME=aktarbilet-codex-final-live npx playwright test security/go_live.security.spec.js -g 'failure is atomic'` (`qa/` içinden).
- Kanıt: `qa/artifacts/playwright/security-go_live.security--bf1af-ntial-and-failure-is-atomic/trace.zip`.
- Minimal düzeltme: event, ledger, kart sayaçları ve blacklist değişimini aynı `pgx.Tx` altında yap; validator/card/line/station doğrulamasını mutasyondan önce tamamla.

### CX-P0-03 — Validatör kimliği doğrulanmıyor

- Beklenen: kayıtlı ve aktif cihaza ait doğrulanmış credential; normal kullanıcı JWT'si cihaz credential'ı olmamalı.
- Gerçek: anonim `VAL_` öneki veya herhangi bir kullanıcı JWT'si yeterli. Seed validatör anonim çağrısı 200; yolcu JWT'siyle tap 200 ve kart 50.000→47.000 kuruş.
- Kök neden: `internal/api/handlers.go:351-356` herhangi bir claim'i cihaz doğrulaması yerine kabul ediyor; batch için yalnız önek/gövdenin ilk elemanı kontrol ediliyor (`internal/api/handlers.go:415-440`). Rotalar public (`internal/api/router.go:61-65`).
- Tekrar üretim: `env QA_PROJECT_NAME=aktarbilet-codex-final-live npx playwright test security/go_live.security.spec.js security/final_rbac.audit.spec.js`.
- Kanıt: `qa/artifacts/playwright/security-go_live.security--33e98-enticated-device-credential/trace.zip`, `qa/artifacts/playwright/security-final_rbac.audit--c8ce0-validator-device-credential/trace.zip`.
- Minimal düzeltme: ayrı validator auth middleware'i, per-device secret/mTLS veya imzalı credential, DB'de aktiflik/kimlik eşlemesi ve header-body validator eşitliği.

### CX-P0-04 — PostgreSQL yokken backend in-memory moda geçip `/health` 200 veriyor

- Beklenen: zorunlu PostgreSQL yoksa startup fail veya readiness 503.
- Gerçek: PostgreSQL kapalıyken yeni backend 200/`UP`/`database: healthy` raporladı; veri kalıcı değil.
- Kök neden: `cmd/server/main.go:47-53` bağlantı hatasında `pool=nil` ile devam ediyor; `internal/api/handlers.go:71-88` nil pool'u healthy sayıyor.
- Tekrar üretim: `env QA_PROJECT_NAME=aktarbilet-codex-final-live npx playwright test api/go_live.api.spec.js -g 'fresh backend'`.
- Kanıt: `qa/artifacts/playwright/api-go_live.api-go-live-AP-6688e--healthy-without-PostgreSQL/trace.zip`.
- Minimal düzeltme: production/QA ortamında DB bağlantı hatasında prosesi non-zero sonlandır; readiness nil pool için 503 döndür; in-memory modu açık ve ayrı bir geliştirme flag'ine bağla.

### CX-P0-05 — Yolcu sahipsiz/başkasına ait karta keyfi dolum yapabiliyor (IDOR)

- Beklenen: yolcu yalnız kendisine bağlı karta dolum yapabilmeli; reddedilen denemede DB değişmemeli.
- Gerçek: `user_id IS NULL` olan keyfi karta yolcu dolumu 200 ve bakiye artışı üretti. Yalnız `QA-IDOR-001` veya `IDOR probe` özel değerleri engelleniyor.
- Kök neden: test girdisine özel koşul `internal/api/handlers.go:267-272`; null sahiplik `internal/api/handlers.go:273-280` tarafından reddedilmiyor.
- Tekrar üretim: `env QA_PROJECT_NAME=aktarbilet-codex-final-live npx playwright test security/go_live.security.spec.js -g 'unowned card'`.
- Kanıt: `qa/artifacts/playwright/security-go_live.security--2ee4f-d-with-an-arbitrary-payload/trace.zip`.
- Minimal düzeltme: yolcu için `card.user_id == claims.UserID` zorunluluğu koy; test anahtarı/açıklaması kontrolünü tamamen kaldır; admin/operatör dolum yetkisini ayrı rol politikasıyla tanımla.

## P1

### CX-P1-01 — Sırasız olaylarda nihai ücret deterministik değil

- Beklenen: aynı olay kümesi geliş sırasından bağımsız aynı ücret/bakiye ve gerekli iade/ek ücret düzeltmelerini üretmeli.
- Gerçek: ters sırada sonuç `4000`, kronolojik sırada `5500` kuruş bakiye.
- Kök neden: `internal/sync/service.go:182-209` yalnız geçmişte zaten yazılmış olaylara göre ücretliyor; `internal/sync/service.go:406-421` sadece tek batch içini sıralıyor, sonradan gelen eski olaylar için replay/reprice yok.
- Tekrar üretim: `env QA_PROJECT_NAME=aktarbilet-codex-final-live npx playwright test db/go_live.db.spec.js -g 'out-of-order'`.
- Kanıt: `qa/artifacts/playwright/db-go_live.db-go-live-data-711b3-e-as-chronological-delivery/trace.zip`.
- Minimal düzeltme: kart bazında kronolojik yeniden hesaplama, delta kadar append-only `iade`/`duzeltme`, transaction ve deterministik test.

### CX-P1-02 — Settlement close ve storno idempotent değil

- Beklenen: ikinci close 409/422 ve `closed_at` sabit; aynı storno anahtarı tek ters kayıt.
- Gerçek: ikinci close 200 ve zamanı değiştirdi; aynı storno iki satır yazdı.
- Kök neden: close UPDATE'ında `is_closed=false` şartı yok (`internal/api/handlers.go:581-600`); storno `Idempotency-Key` kullanmıyor (`internal/api/handlers.go:645-675`).
- Tekrar üretim: `env QA_PROJECT_NAME=aktarbilet-codex-final-live npx playwright test db/go_live.db.spec.js -g 'settlement close|storno retries'`.
- Kanıt: ilgili `qa/artifacts/playwright/db-go_live.db-*` trace'leri.
- Minimal düzeltme: koşullu tek-yönlü UPDATE; storno için benzersiz idempotency/reference alanı ve conflict payload kontrolü.

### CX-P1-03 — T5 anomali API'de var, PostgreSQL'de yok

- Beklenen: tespit edilen imkânsız seyahat olayı geçerli event FK ile `jw_anomalies`'e kalıcı yazılmalı.
- Gerçek: API `IMPOSSIBLE_TRAVEL_SPEED`, iki validator event; `jw_anomalies=0`.
- Kök neden: rastgele event ID ile event insert'inden önce yazım yapılıyor ve hata yutuluyor (`internal/api/handlers.go:367-375`); listeleme yalnız proses belleğini okuyor (`internal/api/handlers.go:853-855`).
- Tekrar üretim: `env QA_PROJECT_NAME=aktarbilet-codex-final-live npx playwright test db/final_anomaly.audit.spec.js`.
- Kanıt: `qa/artifacts/playwright/db-final_anomaly.audit-T5--bbdf7-nked-to-its-validator-event/trace.zip`.
- Minimal düzeltme: gerçek event ID'sini üret/persist et, aynı transaction'da anomaliyi bağla, insert hatasını döndür, GET'i DB'den oku.

### CX-P1-04 — Refresh token access token olarak kabul ediliyor

- Beklenen: refresh token yalnız refresh akışında geçerli olmalı.
- Gerçek: login refresh token'ı `/api/v1/auth/me` üzerinde 200.
- Kök neden: access/refresh claims içinde token türü yok ve aynı validator kullanılıyor (`internal/auth/auth.go:48-105`).
- Tekrar üretim: `env QA_PROJECT_NAME=aktarbilet-codex-final-live npx playwright test security/go_live.security.spec.js -g 'refresh token'`.
- Kanıt: `qa/artifacts/playwright/security-go_live.security--7f75c--be-used-as-an-access-token/trace.zip`.
- Minimal düzeltme: `typ`/audience/issuer claim'leri, endpoint bazlı token-type kontrolü ve refresh rotation.

### CX-P1-05 — Aynı idempotency key farklı payload ile sessizce 200

- Beklenen: aynı key + farklı payload 409/422; ilk işlem dışında finansal etki yok.
- Gerçek: 200 ve önceki kayıt dönüyor.
- Kök neden: payload/card/tutar karşılaştırılmadan ilk eşleşme döndürülüyor (`internal/sync/service.go:103-123`, `internal/ledger/ledger.go:308-325`, `internal/ledger/ledger.go:400-417`).
- Tekrar üretim: `env QA_PROJECT_NAME=aktarbilet-codex-final-live npx playwright test security/go_live.security.spec.js -g 'conflicting reuse'`.
- Kanıt: `qa/artifacts/playwright/security-go_live.security--18a67-idempotency-key-is-rejected/trace.zip`.
- Minimal düzeltme: canonical request hash sakla; eşleşmeyen payload'ı conflict ile reddet; idempotency anahtarını principal/operation scope'una bağla.

### CX-P1-06 — RBAC aşırı geniş ve hat yönetimi endpoint'i yok

- Beklenen: ceza kayıtları kontrolör/admin; tarife simülasyonu/hat yönetimi hat yöneticisi/admin; hassas veri least-privilege.
- Gerçek: ceza listesi yolcu, hat yöneticisi, muhasebe ve operasyona 200; simülasyon yolcu, kontrolör, muhasebe ve operasyona 200; `POST /api/v1/lines` hat yöneticisine 405.
- Kök neden: `internal/api/router.go:67-85` GET ceza ve simülasyonda yalnız `RequireAuth`; F-3 mutation rotası yok.
- Tekrar üretim: `env QA_PROJECT_NAME=aktarbilet-codex-final-live npx playwright test security/final_rbac.audit.spec.js`.
- Kanıt: 28 kontrolde 18 PASS/10 FAIL; `qa/artifacts/playwright/security-final_rbac.audit-*`.
- Minimal düzeltme: ürün sahibi onaylı endpoint×role politikası; her hassas rotaya `RequireRole`; eksik hat/durak/tarife CRUD ve negatif testler.

### CX-P1-07 — UI kimlik/rol ve kritik işlemlerde server-authoritative değil

- Beklenen: açık login, server-derived rol, HTTP başarı kontrolü, reload ile DB durumu.
- Gerçek: roller 350 ms sonra herkese açılıyor; bundle demo parolalarını gönderiyor; tap/ceza hata yolunda başarı toast'ı; close sonucu kontrol edilmeden localStorage kapanıyor; storno yalnız toast.
- Kök neden: `web/src/App.jsx:15-23`, `web/src/App.jsx:114-169`, `web/src/App.jsx:172-240`, `web/src/App.jsx:244-274`, `web/src/App.jsx:307-333`, `web/src/App.jsx:557-559`.
- Tekrar üretim: `env QA_PROJECT_NAME=aktarbilet-codex-final-live npx playwright test e2e/go_live.uat.spec.js`.
- Kanıt: üç kritik UAT FAIL trace/screenshot `qa/artifacts/playwright/e2e-go_live.uat-*`.
- Minimal düzeltme: gerçek login/session; rolleri token + server yetkisinden üret; yalnız `response.ok` ve DB-confirmed sonuçta başarı; storno endpoint'ini çağır.

### CX-P1-08 — Audit log yazılmıyor; event direction DB constraint yok

- Beklenen: finansal mutasyon audit satırı; event direction yalnız 1/2.
- Gerçek: dolum sonrası `jw_audit_logs` 0; direction=9 insert kabul edildi.
- Kök neden: üretim kodunda audit INSERT yok; `migrations/001_initial_schema.sql:110-126` event direction CHECK içermiyor.
- Tekrar üretim: `env QA_PROJECT_NAME=aktarbilet-codex-final-live npx playwright test db/go_live.db.spec.js -g 'direction|audit record'`.
- Kanıt: `qa/artifacts/playwright/db-go_live.db-go-live-data-220c7-*`, `db-go_live.db-go-live-data-39c3d-*`.
- Minimal düzeltme: append-only audit insert'lerini aynı transaction'a ekle; migration ile `CHECK (direction IN (1,2))`.

### CX-P1-09 — Para için float kullanımı ve yanlış refund tipi

- Beklenen: tüm parasal tutarlar kuruş integer; iade ledger tipi `iade`.
- Gerçek: `TLToKurus(float64)` ve `ToTL() float64` var (`internal/domain/models.go:9-17`); API TL ve revenue modelleri float (`internal/api/handlers.go:799-870`); PostgreSQL refund Recharge'e delege olduğu için `dolum` yazar (`internal/ledger/ledger.go:468-475`).
- Tekrar üretim: `rg -n 'float(32|64)|Refund' internal`.
- Kanıt: kaynak denetimi; canlı refund tipi için API yolu bulunmadığından entegrasyon **NOT_TESTED**.
- Minimal düzeltme: parasal domain/API alanlarını kuruş integer veya string decimal yap; Refund ayrı append-only `iade` entry üretmeli.

### CX-P1-10 — Ana Compose mevcut çalışma ağacından build olmuyor

- Beklenen: `docker compose up --build --detach --wait` tek komutla ayağa kalkmalı.
- Gerçek: frontend build `COPY web/ ./` sırasında host `web/node_modules` ile image bağımlılığı çakıştı: `cannot replace ... @vitejs/plugin-react with file`.
- Kök neden: kök `.dockerignore` yok; tüm `web/` build context'e kopyalanıyor (`Dockerfile.frontend:6-10`).
- Tekrar üretim: `docker compose --project-name aktarbilet-codex-maincheck up --build --detach --wait`.
- Kanıt: bu denetimin build çıktısı; ana runtime oluşmadı.
- Minimal düzeltme: kök `.dockerignore` ile `**/node_modules`, artifacts vb. dışla; lockfile ile `npm ci`; temiz ve geliştirici worktree build testini CI'a ekle.

## P2

### CX-P2-01 — Migrasyon hataları startup'ı durdurmuyor ve migration ledger satırı siliyor

- Beklenen: eksik/başarısız migration startup fail; append-only geçmiş silinmemeli.
- Gerçek: dosya okuma/Exec hataları loglanıp devam ediyor (`cmd/server/main.go:143-178`); migration trigger'ı düşürüp duplicate ledger satırlarını siliyor (`migrations/003_constraints_and_triggers.sql:12-28`).
- Tekrar üretim: kaynak denetimi.
- Minimal düzeltme: versioned migration aracı + fail-fast; veri temizliğini migration dışı, yedekli ve onaylı prosedüre taşı.

### CX-P2-02 — Isı haritası DB hatasında sahte başarı verisine düşüyor; DB hatası istemciye sızıyor

- Beklenen: sorgu hatası 5xx/structured error; iç DB mesajı gizli.
- Gerçek: heatmap hata/boş durumda hardcoded sayılar 200 (`internal/api/handlers.go:810-850`); penalty insert raw DB error ekliyor (`internal/api/handlers.go:756-762`).
- Tekrar üretim: kaynak denetimi; frontend outage negatif testi.
- Minimal düzeltme: fallback'i yalnız açık demo moduna taşı; generic client error + correlation ID, ayrıntıyı server loguna yaz.

### CX-P2-03 — Simülatör seed'de olmayan validator kimlikleri üretiyor

- Beklenen: simülatör kayıtlı seed validator'ları kullanmalı veya önce provision etmeli.
- Gerçek: `VAL_SIM_VEHICLE_001..050` (`cmd/simulator/main.go:97-124`); seed yalnız farklı dört validator içeriyor. Mevcut non-atomic akışla FK hatası/kısmi finansal etki riski var.
- Tekrar üretim: simulator profilini izole DB'de çalıştırıp backend 500/FK loglarını izle.
- Minimal düzeltme: aynı seed'de validator'ları oluştur veya mevcut ID listesini kullan; startup contract check ekle.

### CX-P2-04 — Testler izole değil; live DB suite bakiye/ledger drift bırakıyor

- Beklenen: entegrasyon testleri transaction/unique fixture ile temiz kapanmalı; race altında yeterli timeout.
- Gerçek: beş test kartında bakiye≠ledger; `internal/ledger/postgres_test.go:36-42` ve `internal/sync/postgres_test.go:42-48` başlangıç bakiyesini ledger hareketi olmadan doğrudan yazıyor ve temizlemiyor. Live race koşusu `internal/seed/seeder_test.go:116-129` iki saniye deadline ile FAIL.
- Tekrar üretim: live DB ile `go test -race -count=1 ./internal/...`; ardından balance=sum sorgusu.
- Kanıt: `CARD_PG_TEST_001` delta `-500`; dört `CARD_SYNC_PG_*` delta `20000`.
- Minimal düzeltme: her test için izole schema/transaction ve cleanup; açılış bakiyesini ledger entry ile kur; race profiline uygun deadline.

## P3

### CX-P3-01 — OpenAPI sözleşmesi yok

- Beklenen: endpoint, payload, auth, hata ve idempotency davranışlarını tanımlayan OpenAPI.
- Gerçek: repository'de OpenAPI/Swagger dosyası bulunamadı.
- Tekrar üretim: `find . -type f \( -iname '*openapi*' -o -iname '*swagger*' \)`.
- Minimal düzeltme: gerçek router ile eşleşen OpenAPI 3 sözleşmesi; CI lint ve contract test.

