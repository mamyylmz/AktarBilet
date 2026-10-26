# AktarBilet — Final `proje.md` Kabul Denetimi

**Denetçi:** Codex (bağımsız QA)  
**Tarih:** 2026-10-26 (Europe/Istanbul)  
**Bağlayıcı kaynak:** yalnız `proje.md`  
**Denetlenen Git SHA:** `22bf7f41e95d44d58ac49a8206bd508d44342172`  
**Karar:** **SPECIFICATION_NOT_READY**

Bu denetimde production kodu değiştirilmedi. Destructive işlemler yalnız `aktarbilet-final-verify` ve `aktarbilet-final-audit` adlı izole Compose ortamlarında yürütüldü. Lighthouse, 171 maddelik eski matris, 600 saniye ve şartnamede olmayan özel olay/sn eşikleri kabul kapısı yapılmadı.

## A. Sayısal uygunluk

| Kriter | Şartname | Gerçek ölçüm | Sonuç |
|---|---:|---:|---|
| Otomatik test | ≥40 | Go: **106 PASS, 0 FAIL, 5 SKIP**; Playwright: **141 PASS, 8 FAIL, 0 SKIP** (149 seçildi) | **PASS** (eşik), ancak suite bütünü yeşil değil |
| Çekirdek domain coverage | ≥%70 | auth 84.6%, domain 100%, ledger 77.6%, pricing 95.7%, settlement 79.8%, sync 70.9%, transit 78.8% | **PASS** |
| Kart sayısı | ≥200.000 | PostgreSQL: **200.005** | **PASS** |
| Hat sayısı | ≥40 | PostgreSQL: **46** (koleksiyon testinden sonra diğer izole ortamda 47) | **PASS** |
| İşleme kapasitesi | 1.000.000 olay/saat | Mevcut gerçek-yazım koşusu: **100 istek, 0 kabul, 0 persisted event/sn, %100 batch hata** | **FAIL** |
| Docker cold start | ≤5 dakika | Cache'li `up --build --wait`: **92,19 sn**; boş Docker cache ölçümü yapılmadı | **NOT_TESTED** |
| `make verify` / eşdeğeri | ≤15 dakika | Compose verifier: **14/14 PASS, 9,42 sn, exit 0** | **PASS** |
| Mimari karar | ≥5 | `DECISIONS.md`: **5 ADR** | **PASS** |

Coverage komutu: `go test -count=1 -coverprofile=/tmp/aktarbilet-spec.cover ./internal/... ./qa/go/...`. Güncel hedefli regresyon `internal/api`, `internal/sync`, `internal/pricing`, `qa/go` paketlerinde PASS oldu. Yerel race tekrarında kaynak hatası değil, host üzerinde C derleyicisi bulunmaması (`gcc not found`) engeli oluştu; aynı SHA'nın GitHub Actions race aşaması PASS'tir.

## B. İşlevsel uygunluk

| Madde | Sonuç | Bağımsız kanıt |
|---|---|---|
| F-1 Kart/bakiye/dolum | **PASS** | Gerçek PostgreSQL üzerinde kart, ledger ve idempotent recharge; `requests` koşusunda DB etkisi doğrulandı. |
| F-2 Validatör olayları | **PASS** | İmzalı batch, HMAC, replay/duplicate ve kalıcı event testleri; simulator DB'ye olay yazdı. |
| F-3 Hat/durak/tarife | **PASS** | Yetkili hat oluşturma ve katalog sorguları gerçek API/DB ile PASS. |
| F-4 Tam/öğrenci/yaşlı/aktarma | **PASS** | Pricing unit matrisi ve PostgreSQL entegrasyon senaryoları PASS. |
| F-5 İşletmeci gelir paylaşımı | **PASS** | Çoklu işletmeci, dağıtım, kapanış, reopen/storno ve kuruş mutabakatı PASS. |
| F-6 Kontrol/ceza | **PASS** | Kontrolör RBAC ile listeleme ve kayıt oluşturma; DB persistence PASS. |
| F-7 Yoğunluk/gelir raporları | **PASS** | Heatmap ve line-financials gerçek API verisi; dependency hata durumu testi PASS. |
| T-1 Ücret motoru | **FAIL** | Sabit-zaman alt kuralları PASS; fakat üç sırasız olay ters sırada geldiğinde kronolojik sonuç **16.500**, ters sıra sonucu **16.000 kuruş** oldu. |
| T-2 Offline validatör | **PASS** | Kalıcı WAL/queue, signed batch, restart/reconnect recovery, sequence, duplicate, blacklist ve negatif tolerans doğrulandı. Simulator restart'ta 30 bekleyen paketi geri aldı. |
| T-3 Ledger/concurrency | **PASS** | Recharge/tap/refund/correction, ledger reconciliation, iki validatör eşzamanlılığı, duplicate/lost-update negatif testleri PASS. |
| T-4 Gelir paylaşımı | **PASS** | Birden çok işletmeci, transfer, biniş/mesafe dağıtımı, kuruş korunumu ve kapanış sonrası ters kayıt PASS. |
| T-5 Anormallik/toplu veri | **FAIL** | İmkânsız seyahat, clone-card flag ve materialized summary PASS; gerçek 1M/saat persisted+reported kapasite kanıtı yok ve mevcut event-load HMAC doğrulamasında reddediliyor. |

Beş zorunlu rolün gerçek login → portal → yetkili API akışı masaüstü ve 360 px mobil görünümde doğrulandı. Yolcu, kontrolör, hat yöneticisi, muhasebe ve operasyon ekranları gerçek backend verisi gösterdi; temel yatay taşma görülmedi. Backend negatif RBAC/IDOR testlerinde 401/403 PASS. Loading/empty/error durumları kaynakta ve ilgili browser senaryolarında mevcut. Eski bazı Playwright UI senaryolarının anonim kullanıcıyla rol butonu beklemesi güncel zorunlu-login davranışıyla uyumsuzdur; bunlar ürün güvenlik hatası sayılmadı.

## C. Ortak teknik gereksinimler

| Alan | Sonuç | Kanıt / değerlendirme |
|---|---|---|
| Parola/token | **PASS** | bcrypt, JWT erişim kontrolü, yanlış parola ve bozuk/imzası değiştirilmiş token negatif testleri PASS; hassas değer loglama saptanmadı. |
| RBAC ve nesne yetkisi | **PASS** | Beş rol, 401/403, çapraz kullanıcı kart erişimi ve admin ayrımı bağımsız testlerde PASS. |
| DB bütünlüğü | **PASS** | Migration, FK/UNIQUE/CHECK, PostgreSQL 16 ve kritik trigger/constraint testleri PASS. |
| Finansal temsil | **PASS** | Para integer kuruş; finansal hesapta float saptanmadı. |
| Transaction/locking/concurrency | **PASS** | Kart bazında transaction + `FOR UPDATE`; gerçek PostgreSQL concurrency testleri PASS. |
| İdempotency | **PASS** | Validator `(validator_id, sequence_no)` ve kritik POST `Idempotency-Key` tekrarlarında aynı sonuç/tek etki doğrulandı. |
| Audit | **PASS** | `jw_audit_logs` append-only trigger'ı; aktör, zaman, action, record ve değer kaydı testleri PASS. |
| Error/validation/status | **PASS** | Standart JSON hata yapısı, server-side validator ve doğru 4xx durumları negatif testlerle doğrulandı. |
| Büyük listelerde keyset pagination | **FAIL** | Ledger'da `before_id` var; penalty ve anomaly listeleri sırasıyla sabit `LIMIT 100` ve `LIMIT 10` kullanıyor, cursor/keyset sözleşmesi yok (`internal/api/handlers.go:1429`, `:1474`). |
| Dayanıklılık/log/config | **PASS** | Context timeout, graceful shutdown, JSON structured log ve environment configuration mevcut; DB kesilince `/health` 503, geri gelince 200 oldu. |
| İndeks ve EXPLAIN | **PASS** | Kritik indeks migration'ları ve README'de iki sorgu için dizinsiz/dizinli `EXPLAIN ANALYZE` süreleri bulunuyor (ör. 184,2 ms → 0,18 ms). |
| CORS gözlemi | **NON_BLOCKING_WARNING** | Router `http://*` / `https://*` origin desenlerini kabul ediyor. `proje.md` özel CORS allowlist'i istemiyor; buna rağmen üretim dağıtımında daraltılması önerilir. |

T-1 kök nedeni: `internal/sync/service.go:233` sırasız olay uzlaşmasında yalnız `LIMIT 1` ile tek bir sonraki olayı yeniden fiyatlıyor. Üç veya daha fazla olayın etkilenen kronolojik suffix'i bütünüyle replay edilmediği için nihai ücret geliş sırasına bağlı kalıyor.

## D. Kalite ve teslim

| Kriter | Sonuç | Kanıt |
|---|---|---|
| Go testleri | **PASS** | Güncel tam sayım: 106 PASS, 0 FAIL, 5 SKIP; skip'ler PASS sayısına katılmadı. |
| Browser/integration suite | **FAIL** | 149 seçildi: 141 PASS, 8 FAIL. T-1 gerçek ürün hatası; diğerleri arasında stale anonim-UI beklentileri ve bir timeout bulunuyor. |
| Coverage | **PASS** | Her çekirdek domain paketi ayrı ayrı ≥%70; CI coverage profili üretiyor. API HTTP paketi %62,2 olup çekirdek domain eşiğine dahil edilmedi. |
| Frontend build/lint | **PASS** | Güncel SHA: Vite 53 modül, build PASS; 23 dosyada lint 0 error/0 warning. |
| GitHub Actions | **PASS** | SHA `22bf7f4…`, run `40396001836`: build/test/static analysis/race/coverage/container build tamamı success. Önceki production SHA `2465b14…` run `40395585759` da success. |
| OpenAPI | **PASS** | Mevcut API sözleşmesi ve runtime route'ları incelendi; auth, card, sync, catalog, settlement, report ve pagination şemaları mevcut. |
| Metrics | **PASS** | `/metrics` HTTP 200 ve Prometheus metin formatında. |
| Requests collection | **PASS** | `scripts/test_requests_collection.mjs`: **32/32 PASS**, gerçek API, auth, HMAC ve DB etkileri. |
| Load/k6 eşdeğeri | **FAIL** | Event-load mevcut ancak signed batch üretmediği için 5/5 batch reddedildi; gerçek persist/raporlama kapasitesi ölçemiyor. |
| README performans ölçümü | **FAIL** | p50 1,2 ms, p95 4,8 ms, p99 12,1 ms, 24.150 req/sn yazıyor; fakat çalıştırılabilir mevcut yük akışı 0 event kabul etti. Bu değerler 1M/saat persisted+reported kanıtı değildir. |
| Chaos | **PASS** | İzole PostgreSQL stop → `/health` 503; restart → ikinci denemede 200; temiz ortamda ledger mismatch 0. |
| `TESTING.md` ve F/T kanıtları | **PASS** | Rol hesapları, beklenen sonuçlar, manuel UI ve F/T komutları mevcut. |
| `reset-demo` | **PASS** | Yalnız izole DB'de `/app/server -reset`: exit 0; sonra 200.005 kart, 46 hat, 0 validator event, seed ledger 4 ve doğru ilk kullanıcı. |
| README beş soru/kurulum/mimari | **PASS** | Beş cevap, teknoloji gerekçesi, kurulum, mimari, bilinen sınırlama ve ücret doğruluk yaklaşımı mevcut. |
| GitHub/public/history | **PASS** | Repo public; anlamlı commit geçmişi mevcut; current main/origin aynı SHA ve güncel CI yeşil. Force-push yapıldığına dair kanıt görülmedi. |
| Teslim anketi | **NOT_TESTED** | Repository veya erişilebilir dış kanıtta anketin gönderildiği doğrulanamadı. |

Simulator manifest'i seed `42`, 200.000 kart, 40 hat, batch 10 ve sabit clock ile aynı SHA-256 manifestini restart sonrasında korudu. Gerçek PostgreSQL'de simulator olayları arttı ve duplicate sayısı 0 gözlendi. Seed sayıları reset sonrası tekrar doğrulandı; ancak ikinci tamamen bağımsız sıfırdan üretimin tüm satır hash karşılaştırması yapılmadığından “aynı seed → tüm dataset byte-for-byte aynı” alt kanıtı **NOT_TESTED** kabul edilmiştir.

## E. Yedi özel sistem kriteri

| # | Kriter | Gerçek sonuç | Sonuç |
|---:|---|---|---|
| 1 | Bütün uygulama tabloları `jw_` | Migration ve canlı katalog taramasında uygunsuz tablo: 0 | **PASS** |
| 2 | Kullanıcı tablosu `jw_tf_users` | Canlı PostgreSQL ve migration doğrulandı | **PASS** |
| 3 | `X-DDO-Stamp: f85f26` | Health ve API yanıtlarında doğrulandı | **PASS** |
| 4 | `GET /_hz/f85f` → 200 | Gerçek HTTP 200 | **PASS** |
| 5 | İlk seed kullanıcı e-postası | `seed.sedir@ddo-case.invalid` | **PASS** |
| 6 | Backend container adı | `ddo-pelikan-api` (canonical Compose) | **PASS** |
| 7 | README son satırı | `DDO Case Kodu: SEDIR-f85f26` | **PASS** |

## F. Eksik veya başarısız kriterler

| Şartname | Beklenen | Gerçek durum / kanıt | Kök neden / en küçük Development Context düzeltmesi |
|---|---|---|---|
| §6 T-1 | Sırasız olaylarda deterministik nihai ücret | **FAIL:** 16.500 ≠ 16.000 kuruş; hedefli tekrar aynı farkı üretti | `internal/sync/service.go` yalnız bir gelecekteki olayı uzlaştırıyor. Etkilenen tüm kronolojik suffix'i aynı transaction içinde yeniden fiyatlayıp ledger düzeltmelerini idempotent uygulayın; 3+ olay permutation regresyonu ekleyin. |
| §6 T-5 / §9 load | ≥1M olay/saat işleme ve raporlama, persisted ölçüm | **FAIL:** 100 olayın 0'ı kabul; %100 batch hata | Yük istemcisi güncel device HMAC/signed-batch protokolünü üretmiyor. Mevcut harness'i gerçek kayıtlı validator secret ile imzalı batch gönderecek şekilde güncelleyin; persisted count, loss, duplicate, p50/p95/p99, hata ve report refresh ölçün. |
| §7 API | Büyük listelerde keyset pagination | **FAIL:** penalties/anomalies sabit limitli | Cursor parametresi ve kararlı `(created_at,id)` veya `id` keyset koşulu ekleyip OpenAPI/client/testi aynı sözleşmeye bağlayın. |
| §9 bütün suite | Güncel otomatik suite yeşil | **FAIL:** 141/149 Playwright PASS | T-1 ürün hatasını giderin; zorunlu login sonrası eski anonim UI testlerini güncel gerçek login akışına taşıyın; timeout'u kök nedenine göre düzeltin. Assertion zayıflatmayın. |
| §3 cold start | Boş cache ile ≤5 dk | **NOT_TESTED:** yalnız 92,19 sn cache'li ölçüm | Diğer projelerin cache'ini silmeden ayrı builder/cache namespace veya temiz CI runner üzerinde `docker compose up --build --wait` ölçün. |
| §8 determinizm | İki izole üretimde aynı seed aynı veri | **NOT_TESTED:** manifest restart hash'i aynı, tam dataset çift üretim hash'i yok | İki ayrı QA volume'ünde seed üretip sıralı canonical row hash'lerini karşılaştırın. |
| §14 teslim | Teslim anketi gönderilmiş olmalı | **NOT_TESTED:** erişilebilir gönderim kanıtı yok | Proje sahibi anketi gönderip teslim makbuzu/kanıtını saklamalıdır. |

## G. Son özet

Atomik cümleleri yapay biçimde çoğaltmamak için şartname **66 denetlenebilir kapı grubuna** ayrıldı: mimari/Docker 15, roller ve F maddeleri 12, T görevleri 5, ortak teknik 11, kalite/teslim 15, seed determinizmi 1 ve özel kriterler 7. CORS uyarısı ile zorunlu test adedi eşiğini aşan suite-içi ek arızalar ayrıca sayılmadı.

| Özet | Sonuç |
|---|---|
| Zorunlu kriter grubu | **66** |
| PASS | **58** |
| FAIL | **5** |
| BLOCKED | **0** |
| NOT_TESTED | **3** |
| Çalıştırılan testler | Go 111 (106 pass, 5 skip) + Playwright 149 (141 pass, 8 fail); ayrıca requests 32/32 |
| Coverage | Çekirdek paketlerin tamamı ≥%70 |
| F sonucu | **F-1..F-7 PASS** |
| T sonucu | **T-2/T-3/T-4 PASS; T-1/T-5 FAIL** |
| CI | **PASS**, run `40396001836` |
| Git | `main == origin/main == 22bf7f41e95d44d58ac49a8206bd508d44342172`; çalışma ağacı rapor eklenmeden önce temizdi |
| Public repo | **PASS** — `github.com/mamyylmz/AktarBilet` |
| Teslim için kalan ana adım | T-1 suffix replay düzeltmesi, geçerli imzalı 1M/saat persisted/reporting yük kanıtı, keyset pagination ve ardından delta retest; ayrıca cold-start, çift-seed ve anket kanıtı |

Zorunlu T-1 ve T-5 kapıları başarısız olduğundan, test sayısının ve CI'ın yeşil olması release kabulü için yeterli değildir.

## Nihai karar

**SPECIFICATION_NOT_READY**
