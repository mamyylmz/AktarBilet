# Codex Final Delivery Acceptance

**Tarih:** 2026-10-30  
**Branch / HEAD:** `main` / `c2701cf7e9af412c8723f4b1c8c39467f5945c23`  
**Durum:** `PROJE_MD_ACCEPTANCE: NOT_READY` · `GITHUB_CI: NOT_RUN_FOR_WORKTREE` · `DELIVERY_STATUS: NOT_READY`

Bu kabul turu, `prompts/current-task.md` kapsamındaki çalışmadır. Yalnızca aşağıdaki gerçek kanıtlar PASS sayılmıştır; sınırlı verifier veya eski CI sonucu, tüm şartnameyi geçmiş göstermez.

## Son doğrulama özeti

| Kontrol | Sonuç | Kanıt / sınır |
|---|---|---|
| Race | **BLOCKED (güncel kaynak)** | Host `CGO_ENABLED=0`, `gcc` yok. Go Docker denemesi bu turda I/O/SIGBUS aldı; eski race sonucu mevcut ledger değişikliğinden önceydi. |
| Go test / vet | **PASS** | Host `go test -count=1 ./...` ve `go vet ./...` güncel kaynakta PASS; tüm Go paketleri geçti. |
| Domain coverage | **PASS (çekirdek paketler)** | Core domain (`anomaly`, `domain`, `ledger`, `pricing`, `settlement`, `sync`) **77.5% (592/764)**; `proje.md` alt sınırı ≥70%. |
| F/T live verifier | **BLOCKED (bu koşu)** | Önceki sınırlı verifier çıktıları vardır; bu turdaki güncel eşdeğer, QA backend suite sonrası ulaşılamaz olduğundan yeniden PASS sayılmadı. |
| `make verify` / 15 dakika | **BLOCKED (ortam)** | GNU `make` WSL’de yok; `go run ./cmd/verify` doğru QA URL/credentials ile denendi fakat backend bağlantısı reddedildi. |
| PostgreSQL finans / katalog | **PASS (ölçülen invariants)** | Önceki salt-okunur ana DB sorgularında 200.008 kart, 450 aktif İzmir hattı, 11.010 aktif durak, ledger-bakiye farkı 0; dört demo hesabının her biri tek kartla ve ledger toplamıyla eşleşti. QA testleri ayrı DB kullandı. |
| Seed / temiz Docker | **PASS (izole QA)** | Migration 001–012 ve seed ile yeni QA DB açıldı; backend/frontend healthcheck sağlıklı oldu. `docker-compose.qa.yml` dosya hazırlayan init konteynerlerini ağ dışına alır, portlar ortam değişkeniyle seçilebilir. Bu test QA ağı/volümleri içindir. |
| Playwright tam paket | **PASS — 157 PASS / 0 FAIL** | Recovery koşusu temiz `aktarbilet-finalaccept9` PostgreSQL 16 stack’inde `/tmp/aktarbilet-playwright-final-clean.json` ile doğrulandı. Önceki 127/30 sonucu recovery bölümünde tarihçe olarak korunmuştur. |
| README koruması | **PASS** | Korunan `## 12. README’de Cevaplanacak Doğrulama Soruları` bölümü HEAD ve çalışma ağacında birebir aynı: SHA-256 `a279402b114fd6b02891df55f4cb1e36a9c6537509dc8331a0f90aaab249fc26` (6.525 byte). Son satır canary: `DDO Case Kodu: SEDIR-f85f26`. |
| GitHub | **BLOCKED FOR CURRENT WORKTREE** | `gh auth status` PASS. [En son `main` CI run `41056498730`](https://github.com/mamyylmz/AktarBilet/actions/runs/41056498730) başarılı: Backend Testleri/Kapsam/`make verify`, Docker imajları ve Frontend Derleme job’ları PASS. Bu run mevcut kirli çalışma ağacına/HEAD’e ait değil. Yerel zorunlu kabul başarısız olduğu için commit/push yapılmadı; yeni CI run yok. |

## Tarife ve tarihi kayıtlar

Seed tarife `AktarBilet Simülasyon Tarifesi` olarak adlandırılır; `proje.md` içindeki simülasyon ücret motoru kuralları uygulanır. Resmî İzmir tarifesi simülasyon değerleriyle karıştırılmaz. Yaşlı (60+), öğrenci/genç ve ücretsiz kart sınıfları ayrı kalır. Bu kabul kapsamında geçmiş ledger/settlement verileri veya tarifeye bağlı tarihsel finansal kayıtlar geriye dönük değiştirilmedi; gerçek bir resmi tarife mali kaydı olmadan tutar üretilmedi. Resmî ücretlerin canlı sisteme kaynaklı ve yürürlük tarihli uygulanması henüz ayrı uçtan uca kanıtlanmadığından F-4 genel kabulü **BLOCKED** kalır.

## Şartname durumu

| Kapsam | Durum | Açık kabul sınırı |
|---|---|---|
| F-1 kart sahipliği / bakiye / dolum | **BLOCKED** | Tam browser suite ve ledger/bakiye akışları PASS; tüm alt senaryolar için tek bir tam live acceptance çıktısı yok. |
| F-2 validator / HMAC / tekrar koruması | **BLOCKED** | Tam browser suite, signed retry ve replay senaryoları PASS; bütün offline recovery alt maddeleri tek matriste kanıtlanmadı. |
| F-3 İzmir hat/durak | **BLOCKED** | Katalog sayıları kanıtlı; kaynak veride durak sırası olmadığından route-stop dizisi uydurulmadı. |
| F-4 ücret ve sınıflar | **BLOCKED** | Simülasyon ve resmî tarife ayrıldı; resmi tarifenin yürürlük tarihli tam uçtan uca uygulaması kanıtlanmadı. |
| F-5 gelir paylaşımı | **PASS (tanımlı verifier/DB senaryoları)** | Kuruş toplam korunumu ve tam Playwright paketi PASS; kapsam tam şartname matrisi değil. |
| F-6 ceza | **PASS (tanımlı verifier/browser senaryoları)** | Ceza API ve gerçek login ile UI POST akışı tam suite’te PASS. |
| F-7 heatmap / gelir raporu | **BLOCKED** | Bazı canlı API/rapor reconciliation senaryoları geçti; tüm şartname alt koşulları geçmedi. |
| T-1 ücret motoru | **BLOCKED** | Verifier ve Go testleri geçti; tam sıra/gece/90 dakika/günlük-aylık tavan matrisi bir kabul çıktısında kanıtlanmadı. |
| T-2 offline recovery | **BLOCKED** | HMAC, kuyruk/ingest ve replay’in bir kısmı test edildi; tam kabul matrisi PASS değil. |
| T-3 ledger/concurrency | **BLOCKED** | PostgreSQL concurrency ve tam Playwright suite PASS; güncel race detector koşusu gcc/Docker altyapısı nedeniyle alınamadı. |
| T-4 settlement/storno/subsidy | **PASS (tanımlı senaryolar)** | Adjustment ayrı, append-only ve audit’li; close/reopen/storno yaşam döngüsü tam suite’te PASS. |
| T-5 anomaly / 200K / kapasite | **BLOCKED** | 200.008 kart ve önceki signed 18.000 olay kanıtı var; performans fixture’ı son kaynak durumu için tekrarlanmadı. |
| OpenAPI / metrics / k6 / chaos | **BLOCKED** | Metrics HTTP matrisi ve önceki signed kapasite artefaktı mevcut; OpenAPI, k6 ve chaos’ın hepsi için bu turda güncel ayrı PASS kanıtı yok. |

## Bu turdaki QA düzeltmeleri ve sonuç

- QA helper/init konteynerlerinin gereksiz varsayılan ağ kullanımını kaldırdım; QA backend/frontend portları seçilebilir oldu. Go-live outage testine isteğe bağlı Compose override desteği eklendi. Hiçbir test assertion’ı gevşetilmedi.
- Docker adres havuzu sınırı nedeniyle suite, önceki QA verisini koruyarak mevcut QA bridge ağı üzerinde yeni proje/volume ile açıldı.
- Tam Playwright recovery koşusu: **157 PASS, 0 FAIL**. Tarihsel `127/30` hataların tamamı bu rapordaki recovery addendum’da listelenmiştir.
- `git diff --check` temiz. README korunan bölüm ve canary doğrulandı.

## Git ve teslim kararı

```text
HEAD: c2701cf7e9af412c8723f4b1c8c39467f5945c23
COMMIT: NOT CREATED
PUSH: NOT RUN
LATEST REMOTE MAIN CI: run 41056498730 — success (not this worktree)
PROJE_MD_ACCEPTANCE: NOT_READY
DELIVERY_STATUS: NOT_READY
```

Playwright FAIL, eksik F/T kabul kanıtları, route-stop sırası ve güncel OpenAPI/k6/chaos kabul kanıtı nedeniyle READY verilemez. Önceki finansal veriler, QA kanıtları ve kullanıcı çalışma ağacı korunmuştur.

## 2026-10-30 — Playwright acceptance recovery (latest)

Bu bölüm önceki Playwright `127/30` sonucunu günceller. Önceki rapordaki `138/19` koşusu da daha eski bir QA stack/çalışma ağacına aittir.

| Kontrol | Sonuç | Kanıt |
|---|---|---|
| Playwright tam suite | **PASS — 157 PASS / 0 FAIL / 0 SKIP / 0 FLAKY** | Temiz `aktarbilet-finalaccept9` PostgreSQL 16 QA stack’i; suite JSON: `/tmp/aktarbilet-playwright-final-clean.json`; süre 217.0 sn. |
| Eski 30 başarısız test | **30/30 artık PASS** | Hata grupları aşağıda listelendi; son temiz tam suite bunların hepsini içerdi. |
| Hedefli düzeltilmiş regression | **PASS** | Settlement, fare report, anomaly, UAT, axe ve DB integrity hedefli koşuları geçti. Lifecycle önce aynı DB’de yeniden çalıştığı için tek audit sayımı farkı verdi; yeni temiz DB’de tam suite PASS oldu. |
| Go test / vet | **PASS** | Host Go `go test -count=1 ./...` ve `go vet ./...` PASS. |
| Çekirdek domain coverage | **PASS — 77.5% (592/764 statement)** | `anomaly`, `domain`, `ledger`, `pricing`, `settlement`, `sync`; `proje.md` alt sınırı %70. |
| Frontend | **PASS** | `npm run lint` 0 error / 0 warning; `npm run build` 53 modül. |
| 7 sistem canary’si | **PASS — 7/7** | Tam Playwright JSON’daki `AGY Mandatory 7-Canary Audit` kontrolleri geçti; DB prefix/user, HTTP stamp, `/_hz/f85f`, seed e-posta, backend container adı, README son satırı. |
| Race | **BLOCKED — güncel kaynakta ölçülemedi** | Hostta `gcc` yok (`CGO_ENABLED=0`). Go 1.23 Docker denemeleri modül cache I/O hatası/SIGBUS aldı; sonrasında Docker Desktop WSL CLI entegrasyonu kullanılamadı. Önceki race PASS, ledger kaynak değişikliğinden önceki duruma ait olduğundan güncel kanıt sayılmadı. |
| `make verify` / canlı F-T | **BLOCKED** | WSL’de GNU `make` kurulu değil. Aynı `go run ./cmd/verify` reçetesi `VALIDATOR_CREDENTIALS_FILE` ile QA portuna yönlendirildi; suite sonrası Docker backend artık erişilebilir olmadığından bağlantı reddedildi. Gerçek `/app/verify` bu koşuda PASS sayılmadı. |
| GitHub CI / commit / push | **NOT RUN** | Tam F/T ve race kabulü kapanmadı; kullanıcı değişiklikleriyle dolu worktree üzerinde commit/push yapılmadı. Eski run `41056498730` bu worktree’yi doğrulamaz. |

### Önceki 30 Playwright hata ID’si, hata mesajı ve kapanış

| # | Test ID | Önceki gerçek hata | Kök neden / son durum |
|---:|---|---|---|
| 1 | `api/codex_cost_contract.audit.spec.js` — authorized optimization approval | `jw_optimization_approvals` görünmüyordu | Paylaşılan Docker service alias’ı yanlış PostgreSQL’e gidiyordu; proje-özel alias ile PASS. |
| 2 | Aynı dosya — passenger cannot persist approval | `jw_optimization_approvals` sorgusu hata verdi | Aynı QA DB yönlendirme sorunu; PASS. |
| 3 | Aynı dosya — retry creates one durable decision | `jw_optimization_approvals` sorgusu hata verdi | Aynı QA DB yönlendirme sorunu; PASS. |
| 4 | Aynı dosya — invalid negative approval DTO | `jw_optimization_approvals` sorgusu hata verdi | Aynı QA DB yönlendirme sorunu; PASS. |
| 5 | `api/go_live.api.spec.js` — ARCH-04 backend unhealthy without PostgreSQL | Beklenen 503, alınan 200 | Outage probe varsayılan portta yanlış stack’i kontrol ediyordu; project override/port env ile PASS. |
| 6 | `db/codex_posthardening_financial_report.spec.js` — persisted fare updates report | Beklenen 200, alınan 403 | Validator line/station fixture atamayla uyuşmuyordu; `LINE_5`/atama eşleşmesiyle PASS. |
| 7 | `db/codex_posthardening_settlement_report.spec.js` — signed income/storno | Beklenen 200, alınan 400 | Storno açık dönem fixture’ında deneniyordu; QA dönemini kapatma adımıyla gerçek invariant korunarak PASS. |
| 8 | `db/codex_settlement_lifecycle.audit.spec.js` — restart/seed lifecycle | Beklenen 200, alınan 401 | Test API URL’si suite’in project backend’ini göstermiyordu; doğru QA endpoint’iyle PASS. |
| 9 | `db/final_anomaly.audit.spec.js` — impossible travel | Tap beklenen 200, alınan 500; fixture ikinci tapte geçersizdi | Atanmış validator/station eşleşmesi ve 1 saniyelik yüksek hız fixture’ı ile PASS. |
| 10 | `db/final_financial.audit.spec.js` — parallel taps | En az bir tap 200 değildi | API farklı QA stack’ine yönleniyordu; project API URL ile PASS. |
| 11 | Aynı dosya — mixed tap/recharge accounting identity | En az bir işlem 200 değildi | Aynı API/DB ayrışması; PASS. |
| 12 | Aynı dosya — duplicate tap idempotency | En az bir tap 200 değildi | Aynı API/DB ayrışması; PASS. |
| 13 | Aynı dosya — insufficient balance contention | En az bir tap 200 değildi | Aynı API/DB ayrışması; PASS. |
| 14 | Aynı dosya — recharge restart/idempotency | Beklenen 200, alınan 500 | Yanlış API/DB ve restart portu; project endpoint/port ile PASS. |
| 15 | `db/go_live.db.spec.js` — DATA-03 audit append | Audit count artışı beklenirken aynı kaldı | API ve test SQL farklı stack’lere gidiyordu; aynı project DB ile PASS. |
| 16 | Aynı dosya — CORE-04 ordered/reversed delivery | Beklenen 200, alınan 409 | İstekler farklı stack state’ine gidiyordu; aynı QA stack’te PASS. |
| 17 | Aynı dosya — CORE-02 100 concurrent recharges | 100 cevabın tamamı 200 değildi | Yanlış API endpoint’i; aynı DB/API projesiyle 100/100 PASS ve ledger eşitliği. |
| 18 | `e2e/go_live.uat.spec.js` — UAT-04 storno control | `/Muhasebe/` düğmesi timeout | UI artık anonim role switch sunmuyor; açık muhasebe login’i ve form submit ile PASS. |
| 19 | Aynı dosya — UAT-01 passenger action | `+50 TL Yükle` bulunamadı | Anonim kullanıcıda korumalı portalın görünmemesi beklenen davranış; yolcu portalı login modalı ve sıfır otomatik login isteği ile PASS. |
| 20 | `e2e/retest3_accessibility.spec.js` — desktop passenger | serious `color-contrast` `.badge-success` | Başarı metni rengi koyulaştırıldı; axe PASS. |
| 21 | Aynı dosya — desktop login dialog | Aynı `color-contrast` | Aynı CSS düzeltmesiyle axe PASS. |
| 22 | `security/go_live.security.spec.js` — SEC-06 conflicting idempotency key | Beklenen ledger sayısı 1, gözlenen 0 | Test DB’si ile API stack’i ayrışmıştı; PASS. |
| 23 | `tests/browser.uat.spec.js` — recharge failure | `.card-balance-value` yoktu | Test oturum açmadan yolcu kartı arıyordu; gerçek demo login + başarısız ağ yanıtı altında bakiye/toast doğrulaması PASS. |
| 24 | Aynı dosya — turnstile API request | `/500T/` düğmesi/request yoktu | UI’daki gerçek buton `5 ESHOT`; login ve exact accessible name ile PASS. |
| 25 | Aynı dosya — penalty creation | `/Kontrolör/` düğmesi timeout | Gerçek login ve zorunlu form alanlarıyla POST doğrulaması PASS. |
| 26 | Aynı dosya — settlement close | `/Muhasebe/` düğmesi timeout | Gerçek muhasebe login’i ve close state doğrulamasıyla PASS. |
| 27 | Aynı dosya — successful recharge | `+50 TL Yükle` bulunamadı | Gerçek yolcu login’i, POST ve reload sonrası kuruş bakiyesi kontrolüyle PASS. |
| 28 | `tests/database.integrity.spec.js` — all card balances equal ledger | Beklenen 0 uyuşmazlık, 4 bulundu | Yanlış stack’e giden API yazımları; izole tam suite DB’sinde PASS. |
| 29 | Aynı dosya — tariff constraints | Constraint hatası beklenirken hata yoktu | Fixture artık olmayan `TARIFF_ISTANBUL_STD` satırına update atıyordu; mevcut `TARIFF_IZMIR_STD` ile iki constraint PASS. |
| 30 | `tests/t1_live_system.spec.js` — 10 parallel taps | Beklenen 200, alınan 500 | Farklı QA API/DB state’i; izole tam suite’te PASS. |

### Güncel kapsam sınırı

Bu koşuda `proje.md` bütünüyle READY ilan edilmedi. Tam Playwright, Go test/vet, çekirdek coverage, web ve 7 canary PASS; fakat güncel race ve `make verify`/live verifier çalışmadı. Ayrıca route-stop sırası, yürürlük tarihli resmî tarife, OpenAPI/k6/chaos ve diğer önceki matristeki alt kriterler bu suite tarafından kanıtlanmıyor. Bu nedenle `PROJE_MD_ACCEPTANCE: NOT_READY`, `DELIVERY_STATUS: NOT_READY`.

README’nin beş cevap içeren korumalı bölümü HEAD ve çalışma ağacında birebir aynı: SHA-256 `a279402b114fd6b02891df55f4cb1e36a9c6537509dc8331a0f90aaab249fc26`, 6.525 byte. Tam README SHA çalışma ağacında `76aefd8e89d4ed9b003e7b54f1adb68d4a3e7cf6203163a5147bacc70e473902`, HEAD’de `0a0a87da106b5b47c8d0fbab0d7abbbc31ab376b5e8c4a09c01eb060ef5e9f52`; bu tur README’yi değiştirmedi. Son satır canary doğru.

Son Git: branch `main`, HEAD `c2701cf7e9af412c8723f4b1c8c39467f5945c23`; commit/push yok; güncel CI yok.
