# Codex Güvenlik Bulguları

**Denetim zamanı:** 26 Ekim 2026, Europe/Istanbul

**Aday:** yerel `main` çalışma ağacı, taban `e74c89c957085638d11215d44dd06a9593ef4724`
**Sonuç:** **FAIL — çözümlenmemiş P0/P1 güvenlik ve finansal bütünlük açıkları var.**

## P0 bulguları

### SEC-P0-01 — Validatör kimliği `VAL_` önekiyle taklit edilebiliyor

- **Dosya/satır:** `internal/api/handlers.go:368-394`, `internal/api/handlers.go:457-500`, `internal/api/router.go:65-69`.
- **Tekrar üretim:** `cd qa && QA_PROJECT_NAME=aktarbilet-codex-rc2 npx playwright test security/go_live.security.spec.js -g 'SEC-05'`.
- **Beklenen:** Tekil ve toplu validatör uçları, kayıtlı/aktif ve doğru hat-durak atamalı cihazın zorunlu HMAC/certificate kimliğini istemeli; eksik imza 401/403 ve sıfır DB mutasyonu üretmeli.
- **Gerçek:** Tekil uçta HMAC yalnız `X-Device-Signature` gönderilmişse doğrulanıyor; imza yoksa `VAL_` öneki yetiyor. Batch uçta HMAC yok. Seed validatör anonim isteği kabul ediliyor.
- **Kanıt:** Playwright `SEC-05 a VAL_ prefix...` ve `SEC-05 even a seeded validator...` FAIL; `qa/artifacts/codex/playwright-final.json`.
- **Minimal düzeltme:** Her iki uçta imzayı zorunlu kıl; validatörü PostgreSQL’den `is_active`, secret/key-id, line/station assignment ile doğrula; prefix kontrolünü kaldır.

### SEC-P0-02 — Yolcu JWT’si cihaz kimliği yerine geçiyor

- **Dosya/satır:** `internal/api/handlers.go:366-394`, `internal/api/handlers.go:458-490`.
- **Tekrar üretim:** `cd qa && QA_PROJECT_NAME=aktarbilet-codex-rc2 npx playwright test security/final_rbac.audit.spec.js -g 'ordinary passenger JWT'`.
- **Beklenen:** Kullanıcı oturumu ile cihaz kimliği birbirinden bağımsız güvenlik alanları olmalı.
- **Gerçek:** Context’te herhangi bir JWT claim’i varsa imzasız validatör fallback kontrolü atlanıyor.
- **Kanıt:** Passenger JWT ile validatör isteği reddedilmedi; ilgili Playwright testi FAIL.
- **Minimal düzeltme:** Validatör rotalarında kullanıcı JWT’sini cihaz doğrulamasından tamamen ayır; mTLS veya zorunlu cihaz HMAC middleware’i kullan.

### SEC-P0-03 — Başarısız sahte validatör isteği finansal olarak atomik değil

- **Dosya/satır:** `internal/sync/service.go:214-283`, `internal/ledger/ledger.go:302-383`.
- **Tekrar üretim:** `cd qa && QA_PROJECT_NAME=aktarbilet-codex-rc2 npx playwright test security/go_live.security.spec.js -g 'failure is atomic'`.
- **Beklenen:** Kart kilidi, ledger, event, sayaç ve anomali tek transaction içinde commit/rollback olmalı.
- **Gerçek:** `ProcessTap` ledger transaction’ını önce commit ediyor; event insert’i daha sonra ayrı bağlantıda FK hatası alabiliyor. HTTP başarısızken bakiye/ledger değişebiliyor.
- **Kanıt:** Testte event sayısı 0 kalırken kart bakiyesi ve ledger başlangıç değerinde kalmadı; Playwright FAIL.
- **Minimal düzeltme:** Tüm ingest akışına tek `pgx.Tx` geçir; validatör/referans doğrulamasını mutasyondan önce yap; herhangi bir hatada tamamını rollback et.

### SEC-P0-04 — Yolcu sahipsiz karta dolum yapabiliyor

- **Dosya/satır:** `internal/api/handlers.go:282-296`.
- **Tekrar üretim:** `cd qa && QA_PROJECT_NAME=aktarbilet-codex-rc2 npx playwright test security/go_live.security.spec.js -g 'SEC-03'`.
- **Beklenen:** Yolcu yalnız kendisine ait karta dolum yapabilmeli; `user_id IS NULL` kartlar da reddedilmeli.
- **Gerçek:** Kontrol yalnız `cardUserID != nil` durumunda sahipliği karşılaştırıyor. `NULL` sahiplik yetkiyi fail-open bırakıyor; ayrıca özel açıklama/idempotency metnine bağlı test-bypass kodu var.
- **Kanıt:** Yolcu sahipsiz kartı doldurdu; HTTP ve PostgreSQL mutasyonu testi FAIL.
- **Minimal düzeltme:** Sahiplik eşleşmesini `cardUserID != nil && *cardUserID == claims.UserID` olarak zorunlu kıl; probe-string koşullarını kaldır.

### SEC-P0-05 — Refresh token access token olarak kabul ediliyor

- **Dosya/satır:** `internal/auth/auth.go:48-105`, `internal/api/middleware.go:66-90`.
- **Tekrar üretim:** `cd qa && QA_PROJECT_NAME=aktarbilet-codex-rc2 npx playwright test security/go_live.security.spec.js -g 'refresh token'`.
- **Beklenen:** Refresh token yalnız refresh endpoint’inde geçerli olmalı.
- **Gerçek:** Access/refresh token aynı claims tipi ve secret ile, token-purpose claim’i olmadan doğrulanıyor; refresh token `/auth/me` üzerinde geçiyor.
- **Kanıt:** Playwright SEC-02 FAIL.
- **Minimal düzeltme:** `typ`/audience/issuer claim’leri ekle, access middleware’de `typ=access` zorunlu kıl; refresh rotation/revocation ve gerçek logout ekle.

### SEC-P0-06 — Sırasız olaylar deterministik finansal sonuç vermiyor

- **Dosya/satır:** `internal/sync/service.go:185-218`, `internal/sync/service.go:252-283`.
- **Tekrar üretim:** `cd qa && QA_PROJECT_NAME=aktarbilet-codex-rc2 npx playwright test db/go_live.db.spec.js -g 'out-of-order'`.
- **Beklenen:** Aynı olay kümesi geliş sırasından bağımsız aynı nihai bakiye ve ücretleri vermeli.
- **Gerçek:** Her olay geliş anındaki geçmişe göre hemen fiyatlanıyor; geç gelen eski olay sonrası yeniden hesaplama yok. Kronolojik sonuç 7.000 kuruş beklenirken sırasız sonuç 6.000 kuruş oldu.
- **Kanıt:** `CORE-04` Playwright FAIL.
- **Minimal düzeltme:** Kart olaylarını event-time sırasıyla transaction içinde yeniden fiyatla veya deterministik reconciliation kuyruğu/versiyonlama uygula.

### SEC-P0-07 — Bakiye = ledger toplamı DB seviyesinde değişmez değil

- **Dosya/satır:** `migrations/001_initial_schema.sql:51-79`, `internal/ledger/ledger.go:350-383`.
- **Tekrar üretim:** 10 dakikalık yük sonrası rapor sorgusu; sonuç `qa/artifacts/codex/sustained-ingest-db.json`.
- **Beklenen:** Her kartta `jw_cards.balance_kurus = SUM(jw_ledger.amount_kurus)`; doğrudan bakiye manipülasyonu mümkün olmamalı.
- **Gerçek:** Şema `jw_cards.balance_kurus` doğrudan insert/update’a açık. Bağımsız test fixture’larının oluşturduğu 3 gerçek DB kartında 6.500/20.000/20.000 kuruş fark bulundu. `reset-demo` sonrasında fark 0’a döndü; bu, değişmezin yapısal olarak korunmadığını değiştirmiyor.
- **Kanıt:** `post_run_global_balance_mismatches: 3`; `qa/artifacts/codex/sustained-ingest-db.json`.
- **Minimal düzeltme:** Bakiyeyi ledger’dan türet veya sadece güvenli DB fonksiyonu/trigger üzerinden güncellenebilir yap; invariant constraint/audit ve migration ekle.

## P1 bulguları

### SEC-P1-01 — RBAC matrisi hassas verileri gereğinden geniş açıyor

- **Dosya/satır:** `internal/api/router.go:71-95`.
- **Tekrar üretim:** `cd qa && QA_PROJECT_NAME=aktarbilet-codex-rc2 npx playwright test security/final_rbac.audit.spec.js`.
- **Beklenen:** Ceza listesi yalnız kontrolör/admin; tarife simülasyonu yalnız ilgili yönetici/admin; yönetim rolü için gerçek sunucu mutation rotası.
- **Gerçek:** Ceza listesi tüm authenticated rollere, fare simulation tüm authenticated rollere açık; hat yöneticisi için line-management mutation rotası yok.
- **Kanıt:** 10 endpoint×role testi FAIL (`qa/artifacts/codex/playwright-final.json`).
- **Minimal düzeltme:** Her endpoint için açık allow-list; eksik yönetim API’si; rol matrisi kontrat testleri.

### SEC-P1-02 — Settlement close/storno idempotent değil ve audit üretmiyor

- **Dosya/satır:** `internal/api/handlers.go:601-728`, `migrations/001_initial_schema.sql:181-210`.
- **Tekrar üretim:** `cd qa && QA_PROJECT_NAME=aktarbilet-codex-rc2 npx playwright test db/go_live.db.spec.js -g 'settlement close|storno retries|audit record'`.
- **Beklenen:** İkinci close durumu değiştirmemeli; storno retry tek ters kayıt üretmeli; her kritik mutation append-only audit yazmalı.
- **Gerçek:** İkinci close `closed_at` değerini değiştiriyor; storno aynı istekte ikinci satırı ekliyor; audit satırı oluşmuyor.
- **Kanıt:** Üç DB testi FAIL; yük sonrası `audit_rows=0`.
- **Minimal düzeltme:** Close için koşullu `WHERE is_closed=false`; storno için zorunlu idempotency key + unique constraint; aynı transaction içinde audit insert.

### SEC-P1-03 — Anomali kaydı gerçek event’e bağlanmıyor

- **Dosya/satır:** `internal/api/handlers.go:405-413`, `internal/api/handlers.go:428-454`.
- **Tekrar üretim:** `cd qa && QA_PROJECT_NAME=aktarbilet-codex-rc2 npx playwright test db/final_anomaly.audit.spec.js`.
- **Beklenen:** Önce event kalıcılaşmalı; anomali onun gerçek `event_id` değerine atomik bağlanmalı.
- **Gerçek:** Rastgele, henüz var olmayan UUID ile FK insert deneniyor ve hata yutuluyor; event daha sonra farklı UUID ile yazılıyor.
- **Kanıt:** İki event var, ilişkili anomali yok; test FAIL.
- **Minimal düzeltme:** Event ID’yi önceden üretip ingest transaction’ına geçir; anomaly insert hatasını yutma ve aynı transaction’da bağla.

### SEC-P1-04 — Deployable varsayılan secret ve bellek içi replay/revocation durumu

- **Dosya/satır:** `internal/sync/device_auth.go:21-40`, `internal/sync/device_auth.go:84-135`, `internal/api/handlers.go:60-70`, `docker-compose.yml:31`.
- **Tekrar üretim:** `rg -n 'default-secret|JWT_SECRET:-' internal/sync/device_auth.go docker-compose.yml`; restart sonrası replay state’in sıfırlandığını kaynak üzerinden doğrula.
- **Beklenen:** Secret yoksa startup fail; cihaz nonce/sequence/revocation kalıcı ve çok-instance tutarlı olmalı.
- **Gerçek:** Sabit HMAC ve JWT varsayılanları var; cihaz durumu process memory’de; server env’den DeviceAuth’a secret enjekte etmiyor. Nonce, imza doğrulanmadan önce tüketiliyor (`device_auth.go:104-130`).
- **Kanıt:** Kaynak incelemesi; `gofmt -l` ayrıca yeni device-auth dosyalarını CI’ya uygunsuz gösterdi.
- **Minimal düzeltme:** Secret manager/env zorunluluğu ve startup validation; per-device key; nonce/sequence/revocation için transactional PostgreSQL/Redis; nonce’ı başarılı imzadan sonra kaydet.

### SEC-P1-05 — Backend PostgreSQL olmadan fresh-start’ta sağlıklı görünebiliyor

- **Dosya/satır:** `cmd/server/main.go:45-54`, `internal/api/handlers.go:86-103`.
- **Tekrar üretim:** `cd qa && QA_PROJECT_NAME=aktarbilet-codex-rc2 npx playwright test api/go_live.api.spec.js -g 'fresh backend'`.
- **Beklenen:** DB zorunlu mimaride startup fail veya `/health` 503.
- **Gerçek:** Bağlantı kurulamazsa memory mode’a geçiliyor; pool `nil` iken `/health` DB’yi `healthy` bildiriyor.
- **Kanıt:** ARCH-04 FAIL. Buna karşılık çalışan instance’ta PostgreSQL kesintisi 503 verdi; problem yalnız fresh-start fail-open yolunda.
- **Minimal düzeltme:** Production/compose ortamında DB bağlantısını startup zorunluluğu yap; `nil pool` health durumunu DOWN döndür.

### SEC-P1-06 — Frontend demo parolaları ve localStorage oturum otoritesi

- **Dosya/satır:** `web/src/App.jsx:4-24`, `web/src/App.jsx:212-224`, `web/src/App.jsx:663-699`.
- **Tekrar üretim:** `rg -n 'Password123|SedirPassword|localStorage' web/src/App.jsx`.
- **Beklenen:** Frontend bundle’da parola olmamalı; rol `/auth/me` ile server’dan belirlenmeli; token için XSS’e dayanıklı oturum yaklaşımı tercih edilmeli.
- **Gerçek:** Dört rolün parolası bundle’a gömülü; rol/token/user ve finansal UI durumu localStorage’da tutuluyor.
- **Kanıt:** Kaynak satırları ve bağımsız secret taraması.
- **Minimal düzeltme:** Hızlı rol/parola kodunu kaldır; server-authoritative session/role kullan; mümkünse HttpOnly Secure SameSite cookie ve logout/revocation ekle.

## Olumlu kontroller

- Altı rolün gerçek login’i, yanlış parola/bilinmeyen kullanıcı, bozuk/süresi dolmuş JWT ve hassas rotalarda eksik token kontrolleri geçti.
- `jw_ledger` ve `jw_audit_logs` UPDATE/DELETE trigger’ları çalışıyor; sorun, audit’in hiç yazılmaması ve bakiye cache sütununun korunmaması.
- 20 paralel tap, paralel tap+recharge, concurrent duplicate ve retry/restart finans testlerinin altısı geçti.

