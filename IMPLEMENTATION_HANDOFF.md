# IMPLEMENTATION_HANDOFF.md

## Proje: AktarBilet — Geliştirme Checkpoint

**Son Güncelleme:** 2026-10-30 (UI/data consistency + Android backend reconnect)  
**Git HEAD:** c2701cf7e9af412c8723f4b1c8c39467f5945c23  
**Git Çalışma Ağacı:** Temiz değil — mevcut kullanıcı/önceki ajan değişiklikleri korunuyor; yeni uygulama değişiklikleri de commit edilmedi.  

---

## ✅ TAMAMLANAN AŞAMALAR

### FAZ 1 — Backend / Finansal Kritik Düzeltmeler

| Görev | Dosya | Durum |
|-------|-------|-------|
| JWT secret fatal | `cmd/server/main.go` | ✅ DONE |
| Settlement audit atomikliği (TX rollback) | `internal/api/handlers.go` | ✅ DONE |
| RBAC fail-closed | `internal/api/middleware.go` | ✅ DONE |
| OOO-REPRICE reference formatı | `internal/sync/service.go` | ✅ DONE |
| Keyset pagination (anomalies) | `internal/api/handlers.go` | ✅ DONE |
| Live verify T-1 ve T-4 | `cmd/verify/main.go` | ✅ DONE |
| **ListSettlements — gerçek veri** | `internal/api/handlers.go` | ✅ DONE (bu tur) |
| **ReopenSettlementPeriod — dinamik dönem** | `internal/api/handlers.go` | ✅ DONE (bu tur) |
| **Migration: idempotency_key, penalty enum, reopen fields** | `migrations/010_settlement_ledger_idempotency.sql` | ✅ DONE (bu tur) |
| **Test assertion güncelleme (reopen dönem kısıtı)** | `internal/api/handlers_test.go` | ✅ DONE (bu tur) |
| Settlement migration runner kayıtlaması (010) | `cmd/server/main.go` | ✅ DONE; clean QA DB applied 001–010 |
| Settlement reopen/storno audit JSON tipleri | `internal/api/handlers.go` | ✅ DONE; PostgreSQL rollback/concurrency/reopen PASS; signed storno lifecycle PASS |
| Settlement raporunun dönem olayları ve signed adjustments kaynağı | `internal/api/handlers.go` | ✅ DONE; persisted fare event and signed adjustment integration PASS |
| Current/open period calendar bounds | `internal/seed/seeder.go` | ✅ DONE; current month uses calendar start/end |
| Keyset pagination: malformed/stale cursor + exact page boundary | `internal/api/handlers.go` | ✅ DONE; DB keyset regression PASS |
| T-1 contradictory CORE-04/T1-08 fixtures | — | ✅ RECHECKED; both PASS on same isolated clean QA stack, no source correction needed |
| Validator altered-payload sequence + exact signed retry | — | ✅ RECHECKED; both PASS on isolated clean QA stack |
| Signed T-5 batch ingestion/report reconciliation | `qa/load/codex-signed-t5-capacity.mjs` | ✅ PASS on repeat: 18,000 submitted/accepted/persisted, 18,000 ledger debits, 36,000,000 kuruş fare delta exactly matched report |
| Admin user/role directory | `internal/api/handlers.go`, `internal/api/router.go` | ✅ DONE; real DB roles via admin-only endpoint |
| Settlement close/reopen/storno audit history | `internal/api/handlers.go`, `web/src/features/accounting/AccountingPortal.jsx` | ✅ DONE; real timestamps, actor, reason and signed amount |
| Aktarma sübvansiyonu ayrı settlement adjustment olarak | `migrations/011_settlement_adjustments.sql`, `internal/api/handlers.go`, `internal/api/router.go`, `web/src/services/api.js`, `web/src/features/accounting/AccountingPortal.jsx`, `web/src/App.jsx` | ✅ DONE; integer kuruş, append-only, dönem/gerekçe/tarih/actor, atomik audit ve idempotency |

### FAZ 2 — Yolcu Portalı

| Görev | Dosya | Durum |
|-------|-------|-------|
| **PassengerPortal: 10'lu sayfalama** | `web/src/features/passenger/PassengerPortal.jsx` | ✅ DONE (bu tur) |
| **PassengerPortal: kullanıcı dostu açıklamalar** | `web/src/features/passenger/PassengerPortal.jsx` | ✅ DONE (bu tur) |
| **PassengerPortal: kiosk başlığı sadeleştirildi** | `web/src/features/passenger/PassengerPortal.jsx` | ✅ DONE (bu tur) |
| **InspectorPortal: bakiye fallback kaldırıldı** | `web/src/features/inspector/InspectorPortal.jsx` | ✅ DONE (bu tur) |
| **InspectorPortal: ihlal enum backend ile hizalandı** | `web/src/features/inspector/InspectorPortal.jsx` | ✅ DONE (bu tur) |
| **Navbar: emoji etiketleri kaldırıldı** | `web/src/components/Navbar.jsx` | ✅ DONE (bu tur) |

### FAZ 3 — Kontrolör Portalı

Inspector balance fallback, penalty enum alignment, and removal of penalty-free card balance deduction are complete. Penalty issuance is separate from card charging.

### FAZ 6 — Genel UI Temizliği

| Görev | Dosya | Durum |
|-------|-------|-------|
| **HomePage: Demo Kiosk, Heatmap, CTA blokları kaldırıldı** | `web/src/features/public/HomePage.jsx` | ✅ DONE (bu tur) |

---

## ⏳ AÇIK KALAN AŞAMALAR

### FAZ 1 — Finansal / Backend (Codex ilerlemesi)
- [ ] Signed T-5 batch sonrası rapor tutarlılığını validator batch fixture ile doğrula.
- [ ] Açık teknik risk: QA helper'ın settlement report regression testi storno öncesi dönemi kapatmıyor; doğru OPEN→CLOSED→STORNO→REOPEN akışı Codex smoke ile geçti.

### FAZ 4 — Harita, Isı Haritası ve Operasyon
- [x] Operations/Map girişinde ESHOT 800 initial fetch; geç yanıt yeni hat seçimini ezemez
- [x] OperationsPortal içindeki tekrarlanan Isı Haritası alt sekmesi kaldırıldı; navbar/yolcu erişimi korundu
- [x] Isı haritası yenileme loading/error/son güncelleme durumuyla gerçek API çağırır
- [x] Validator/WAL telemetri kartları dar ekranlarda okunur ve taşmaz
- [x] Canvas haritası responsive ölçülendirme ve device-pixel-ratio çizimiyle keskinleştirildi; MapLibre/PMTiles altyapısı korundu

### FAZ 5 — Muhasebe, Admin ve Rol Yönetimi
- [x] Muhasebe dönem geliri/biniş özeti persisted settlement API'sinden; sahte dolum/destek rakamları kaldırıldı
- [x] Dönem dolum toplamı ve işlem sayısı gerçek `jw_ledger` dolum kayıtlarından
- [x] Aktarma desteği/sübvansiyon tutarı: ayrı `SUBSIDY` settlement adjustment; yalnız muhasebe kullanıcısının açıkça girdiği pozitif tam sayı kuruş kapalı döneme yazılır, gerekçe/actor/tarih audit edilir. Migration geçmiş tutarları değiştirmeden mevcut ledger satırlarını sınıflandırır; tutar uydurulmaz.
- [x] Dönem close/reopen/storno geçmişi gerçek audit kaydından gösterilir
- [x] Admin varsayılanı login akışında Sistem Yönetimi; navbar'daki Muhasebe Denetimi tekrarı kaldırıldı
- [x] Sistem sağlık kartları `/health` yanıtı ve gerçek timestamp/database durumunu gösterir; sabit 7/7 iddiası kaldırıldı
- [x] Kullanıcı/çoklu rol matrisi admin-only `/api/v1/admin/users` endpoint'inden beslenir
- [x] Tek native role preview seçicisi; önizlemenin yetki vermediği açıkça belirtilir

### FAZ 6 — Genel UI Temizliği
- [x] Toast bildirimleri masaüstü ve mobilde navbar altı sol üstte
- [x] Select/textbox renkleri tema değişkenleriyle eşitlendi; davranışları değişmedi
- [x] Ana sayfada istenen tanıtım/demo bölümleri yok; navbar metni İzmir ile tutarlı

### FAZ 7 — Android APK
- [x] Capacitor 8 kurulumu ve `web/capacitor.config.json`
- [x] Adaptive AktarBilet ticket icon, splash theme, debug-only cleartext yapılandırması ve `adb reverse` uyumlu `localhost:8080` API adresi
- [x] Android debug APK üretildi, v2 imzası doğrulandı ve `C:\Users\asus\Downloads\AktarBilet-debug.apk` konumuna kopyalandı (4.1 MB; SHA-256 `a552de43f27ccb97fb92ecb4232e70e1e49e75028bcaeaf0c3acda715d0e78cc`)
- [ ] USB cihaz testi: Windows ADB `Redmi Note 8` cihazını görüyor ve `tcp:8080 -> tcp:18082` reverse kuralı kuruldu; APK yüklemesi MIUI `INSTALL_FAILED_USER_RESTRICTED` ile reddedildi. Uygulama backend testi henüz yapılmadı/PASS değil.

---

## Doğrulama Sonuçları (Önceki Checkpoint)

```
go test ./internal/... → tüm paketler OK (11/11 PASS)
npm run build          → başarılı (4.43s, 53 modül)
go build ./...         → temiz (0 hata)
```

Codex devam turu: `go test ./internal/api ./internal/seed ./cmd/server ./cmd/verify ./internal/transit` PASS. İzole PostgreSQL 16/API: `financial_report`, `settlement_atomicity`, signed close/storno/idempotent retry/report delta/reopen, `CORE-04`, `T1-08`, keyset, altered-sequence ve exact retry senaryoları PASS. Güncel imajda `/app/verify` tam çalıştı: SYS-1/2, F-1..F-7 ve T-1..T-5 PASS. ESHOT dış servis yanıt vermezken bounded stale fallback 4s içinde döndü. Frontend `npm run lint` ve `npm run build` PASS; headless browser smoke 390px: initial 800 fetch, hızlı 470→35 seçim, navbar heatmap erişimi, yatay taşma yok. `codex_posthardening_settlement_report` testi dönemi kapatmadan storno istediği için 400 aldı (beklenen invariant); doğru lifecycle Codex smoke ile geçti. Mevcut `aktarbilet-codex-final` QA servislerine dokunulmadı.

Bu checkpoint'te değiştirilen dosyalar: `internal/api/handlers.go`, `internal/api/router.go`, `web/src/features/accounting/AccountingPortal.jsx`, `web/src/features/admin/AdminPortal.jsx`, `web/src/components/Navbar.jsx`, `web/src/index.css`, `web/src/services/api.js`, `web/src/map/MapLibreView.jsx`, `web/package.json`, `web/pnpm-lock.yaml`, `web/pnpm-workspace.yaml`, `web/capacitor.config.json`, `web/.env.android-debug` ve `web/android/` Capacitor platform dosyaları.

Bu checkpoint'te ek doğrulama: `go test ./internal/api ./cmd/server`, `go vet ./internal/api ./cmd/server`, web lint/build, Android `assembleDebug` ve APK v2 signature verification PASS. Rebuilt isolated API integration: kullanıcı listesi anonime 401, muhasebe rolüne 403, admin'e 200; 6 DB kullanıcısı listelendi. Dolum raporu 36,017 işlem / 3,600,270,000 kuruş döndürdü ve aynı dönem için PostgreSQL `jw_ledger` sorgusuyla tam eşleşti. Signed T-5 fixture ilk örnekte 275.62 event/s ölçtü; tekrarda 299.68 event/s (1,078,848/saat), 0 hata, 18,000 event ve ledger debit, tam 36,000,000 kuruş rapor delta ile `T5_CAPACITY_VERIFIED` verdi. İki fixture sonucu `artifacts/codex-signed-t5-*.json` altında.

## Kritik Notlar

- Bu devam görevinde değiştirilen/eklenen dosyalar: `migrations/011_settlement_adjustments.sql`, `cmd/server/main.go`, `internal/api/handlers.go`, `internal/api/router.go`, `web/src/services/api.js`, `web/src/features/accounting/AccountingPortal.jsx`, `web/src/App.jsx`, `IMPLEMENTATION_HANDOFF.md`.
- 2026-10-30 devam turu: migration 011 izole QA DB'de uygulandı. `go test ./internal/api ./internal/settlement ./cmd/server`, `go vet ./internal/api ./cmd/server`, `npm run lint`, `npm run build` ve `git diff --check` PASS.
- İzole QA API smoke: ilk subsidy kaydı 201; aynı Idempotency-Key retry 200/idempotent; rapor toplamı 12.345 kuruş/adet 1; audit action ve gerekçe mevcut. Bu değer sadece QA fixture'ıdır, gerçek/üretim subvansiyonu değildir.
- İlk WSL ADB kontrolünde cihaz görünmüyordu; Windows ADB kontrolü sonraki adımda telefonu buldu. O ilk kontrolde APK cihaz testi yapılmadı. QA testleri değiştirilmedi, commit/push yapılmadı.
- 2026-10-30 Windows 11 + WSL devamı: WSL ADB daemon sandbox socket hatası verdi; Windows SDK `adb.exe` kullanıldı. `adb devices -l`: Redmi Note 8 (`169ddad3`, Android 11 / MIUI V125) `device`. Reverse `tcp:8080 tcp:18082` kuruldu ve Windows'tan QA API health erişimi sağlıklı. Windows Downloads APK'sı repo APK'sıyla SHA-256 eşleşiyor (`a552de43…d0e78cc`). `adb install -r` MIUI'den `INSTALL_FAILED_USER_RESTRICTED: Install canceled by user` aldı; APK açılmadı, uygulama testi PASS değil. Kullanıcıdan MIUI USB kurulum iznini açması/onaylaması bekleniyor.
- Yeniden deneme: Windows ADB cihazı ve reverse kuralı hâlâ mevcut; hem `adb install -r` hem `adb install --no-streaming -r` aynı `INSTALL_FAILED_USER_RESTRICTED` ile reddedildi. Kurulum/açılış/backend uygulama testi hâlâ yapılmadı ve PASS değil.
- Sonraki tekrar: telefon `device` durumunda yeniden algılandı (transport_id değişti), ancak `adb install -r` yine `INSTALL_FAILED_USER_RESTRICTED: Install canceled by user` döndürdü; kurulum ve uygulama testi PASS değil.
- Kullanıcının isteğiyle APK kurulum yapılmadan telefona `/sdcard/Download/AktarBilet-debug.apk` olarak kopyalandı; cihazdaki SHA-256 `a552de43f27ccb97fb92ecb4232e70e1e49e75028bcaeaf0c3acda715d0e78cc` ile kaynak APK eşleşti. Manuel yükleme ve uygulama/backend testi henüz yapılmadı.
- Kullanıcı APK'yı telefona manuel yükleyip açtı. Windows ADB'de paket `com.aktarbilet.app` kurulu/çalışan ve `adb reverse tcp:8080 tcp:18082` aktif doğrulandı. Telefonda canlı araç ekranı 17 aktif araç gösterdi; backend logunda telefon kaynaklı `OPTIONS` ve `GET /api/v1/transit/live-buses` (192.168.160.1, GET 734 ms) görüldü. Bu, mevcut APK'nın gerçek yerel backend'e eriştiğini doğrulayan cihaz smoke testidir — PASS. Uygulama içi login/finansal işlem akışı bu smoke kapsamına dahil değildir.
- Kullanıcının yeniden bağlantı isteğinde Windows ADB transport_id 8 ile telefonu buldu; reverse kuralı tekrarlandı ve uygulama yeniden açıldı. Backend logunda telefondan `/health`, `/api/v1/anomalies` ve `/api/v1/admin/users` istekleri; Windows health check'te `database: healthy` görüldü.

- `make verify` çalıştırmak için `VALIDATOR_CREDENTIALS_FILE` env değişkeni gerekiyor
- `migrations/010_settlement_ledger_idempotency.sql` ve `011_settlement_adjustments.sql` isolated QA DB'de uygulandı; production migration durumu bu checkpoint kapsamında değiştirilmedi
- `ListSettlements` gerçek `jw_validator_events`, `jw_ledger` dolumları ve açıkça girilmiş `SUBSIDY` adjustment kayıtlarını okur; kaynak olmayan sübvansiyon tutarı oluşturulmaz
- `ReopenSettlementPeriod` artık tüm dönem adlarını kabul eder, otomatik oluşturur

## Bir Sonraki Görev

Sonraki adım: bağımsız delta QA. T-5 rapor/kapasite fixture'ı tekrarlı ölçümle PASS.

Son tam verifier çalıştırması güncel QA imajında PASS oldu; bu çalışma bağımsız nihai QA yerine geçmez.  

## 2026-10-30 — Nihai teslim kabul denemesi (Codex)

- `prompts/current-task.md` bu turda baştan sona okundu ve görev kapsamındaki mevcut çalışma devam ettirildi. İptal edilmiş son işlem salt-okunur PostgreSQL kabul sorgusuydu; tekrar çalıştırıldı, veri değiştirmedi.
- PostgreSQL doğrulaması: 450 aktif hat (446 ESHOT otobüs, 3 tramvay koridoru, 1 İZDENİZ hattı), 11.010 aktif durak, 200.008 kart, 76 ledger hareketi; bakiye/ledger uyuşmazlığı 0. İzmir seed hatlarının içinde İstanbul geçmiş ID'leriyle çakışanlar `LINE_ESHOT_*` kimliği alır. Tarihsel İstanbul satırları korunur/pasiftir; 40 eski İstanbul seed ID'sinin aktif sayısı 0. Tekrarlı backend seed sonrası sayımlar ve ledger neti (220.000 kuruş) değişmedi.
- Demo hesap sorgusu: dört ayrı yolcu hesabının her birinde tek örnek kart; tam 90.000, öğrenci 25.000, yaşlı 15.000 ve ücretsiz 0 kuruş. Her kart bakiyesi defter toplamıyla eşleşti. Canlı verifier hesap/kart sahipliği ve çapraz kullanıcı erişim reddini doğruladı.
- Uygulama imajında migration 012 ve seed ile ayrı PostgreSQL 16 kabul veritabanı oluşturuldu; gerçek HTTP sunucusu ile `/app/verify` senaryoları çalıştırıldı. Verifier'ın tanımlı SYS/F/T senaryoları PASS; bu çıktı şartnamenin tüm alt maddelerini tek başına kanıtlamaz.
- `go test ./...`, `go vet ./...`, web lint/build ve `git diff --check` PASS. Önceki başarılı Android cihaz smoke kanıtı handoff'ta kayıtlı; bu turda APK değiştirilmedi.
- Race test `CGO_ENABLED=1 go test -race ./...` gcc bulunmadığı için çalıştırılamadı; gcc kurulum girişimi ortamın root yetkisi olmadığı için engellendi. `make verify` için `VALIDATOR_CREDENTIALS_FILE` gerekir. Ağ/DNS nedeniyle GitHub'a erişim ve güncel CI sonucu doğrulanmadı.
- Kapsam açıkları: resmi ESHOT snapshot'ı güzergâhların durak sırasını vermediğinden route-stop sırası/bağlantısı uydurulmadı; tüm her-hesap ayrı giriş senaryoları ve `proje.md` alt maddelerinin tamamı canlı koşuda yürütülmedi. Seed'deki örnek tarife resmi 01.04.2026 tarifesiyle aynı değildir; resmi güncel tabloda merkez tam/genç/60 yaş ücretleri 35/17,50/29 TL'dir. Bu sebeple hat kataloğunun sayısal şartları PASS olsa da uçtan uca `proje.md` kabulü ve teslim NOT_READY'dir.
- README'nin beş soru-cevap bölümü git tabanına göre birebir korundu; son satır canary doğru. Nihai matris ve kanıt sınırları `qa/reports/CODEX_FINAL_DELIVERY_ACCEPTANCE.md` dosyasındadır.
- Blokerler giderilmediği için commit/push yapılmadı; kullanıcı çalışması ve mevcut Docker/QA verileri korunmuştur.
## 2026-10-30 — UI/UX, veri tutarlılığı ve cihaz bağlantısı

- **FIXED:** Yolcu kartı ve hareketleri tek snapshot API'sinden gelir; bakiye ledger toplamıdır, kart kimliği kullanıcıdan alınır, sahte bakiye/kart fallback'leri ve eski yanıtların state'i ezmesi kaldırıldı. Biniş/dolum sonrası gerçek snapshot yenilenir.
- **FIXED:** Admin audit zamanı doğru API timestamp'inden Türkçe `Europe/Istanbul` biçiminde gösterilir; kaynağı olmayan zaman açıkça belirtilir. Sağlık kartları ölçülmeyen kontroller için PASS üretmez.
- **FIXED:** Navbar çoklu rol buton grubu ve “İzmir Akıllı Ulaşım” metni kaldırıldı; admin Sistem Yönetimi bağlantısı korundu. Toast navbar altında sol üstte. Settlement geçmişi dönem/audit durumu ve mobil kaydırma içerir.
- **FIXED:** Backend'de gerçek validator telemetrisi olmadığı için statik cihaz/WAL örnekleri canlı veri gibi gösterilmez. Ledger mutasyonları kartı kilitler ve başlangıç bakiyesini append-only ledger toplamından okur.
- **Gerçek cihaz bağlantısı:** Windows ADB Redmi Note 8'i (`169ddad3`) gördü. `adb reverse tcp:8080 tcp:8080` etkin. Telefonda `curl http://127.0.0.1:8080/health` → `200 OK`, `database: healthy`, `X-DDO-Stamp: f85f26`. Uygulama yeniden yüklenmedi/kapatılmadı.
- **Değişen kaynak dosyaları:** `internal/api/handlers.go`, `internal/api/router.go`, `internal/ledger/ledger.go`, `web/src/App.jsx`, `web/src/components/Navbar.jsx`, `web/src/components/ui/Toast.jsx`, `web/src/features/accounting/AccountingPortal.jsx`, `web/src/features/admin/AdminPortal.jsx`, `web/src/features/inspector/InspectorPortal.jsx`, `web/src/features/operations/OperationsPortal.jsx`, `web/src/features/passenger/PassengerPortal.jsx`, `web/src/index.css`, `web/src/lib/formatters.js`, `web/src/services/api.js`.
- **Minimum doğrulama:** `go test ./internal/api ./internal/ledger ./internal/sync` PASS; `go vet ./internal/api ./internal/ledger ./internal/sync` PASS; `npm run lint` PASS; `npm run build` PASS; Android `npm run android:debug` PASS; APK v2 imza doğrulaması PASS; APK içindeki JS bundle yeni `getCardSnapshot` istemcisini içeriyor; `git diff --check` PASS.
- **APK:** `C:\Users\asus\Downloads\AktarBilet-debug.apk`; önceki SHA-256 `a552de43f27ccb97fb92ecb4232e70e1e49e75028bcaeaf0c3acda715d0e78cc`; yeni SHA-256 `0ec831facd14746f21dd5f325e1ceab4ec4119df6fb3a16c275ab4893cb61a91` (indirilen kopya ve build çıktısı eşleşiyor). Telefona kurulum yapılmadı.
- **Bağımsız QA:** Aynı kullanıcı/kart için web ve Android bakiye/hareket eşleşmesi; kabul edilen ve reddedilen biniş sonrası hareket; login/rol değişiminde stale yanıt; audit timestamp ve zaman dilimi; admin navbar/sağlık; dar ekran settlement tablosu; telemetri yokken cihaz durumu; yeni APK'nın manuel kurulum smoke testi.

Durum: `UIUX_IMPLEMENTATION_COMPLETE — READY_FOR_INDEPENDENT_QA`.

## 2026-10-30 — Final acceptance blocker remediation (Codex)

- `prompts/current-task.md` eksiksiz okundu. Race detector hostta `CGO_ENABLED=0` ve `gcc` yokluğu nedeniyle kullanılamadı; güncel kaynak için Go 1.23 Docker ortamında `go test -race ./...` tüm paketlerde PASS. `go vet ./...`, `go test ./...` ve PostgreSQL-backed internal race testleri PASS. Çekirdek domain coverage 81.9% (607/741); tüm internal coverage 63.1%.
- Temiz PostgreSQL 16 QA DB’de migration 001–012, seed ve HTTP `/app/verify` SYS-1/SYS-2, F-1..F-7 ve T-1..T-5 için tanımlı senaryolarda PASS; komut 15 dakika sınırının altında. Bu sınırlı canlı matris, şartnamenin bütün alt maddelerini kanıtlamaz.
- Tam Playwright suite 157 test: **127 PASS / 30 FAIL**. Sonuç temiz ve ayrı QA DB’den alınmıştır. En az bir hata, optimizasyon onayı testinin `jw_optimization_approvals` tablosunu görememesi; concurrency, UAT ve accessibility kabul hataları da mevcut. Ayrıntı ve trace/context dosyaları `/tmp/aktarbilet-finalaccept4-playwright` altındadır. Suite PASS olmadığı için kabul engeli sürüyor.
- QA Compose düzeltmesi: init konteynerleri `network_mode: none`; backend/frontend host portları `QA_BACKEND_PORT` ve `QA_FRONTEND_PORT` ile seçilebilir; gereksiz default ağ kullanımına karşı built-in `bridge` tanımı var. `qa/api/go_live.api.spec.js`, `QA_COMPOSE_OVERRIDE` verilirse outage testindeki Compose restart işlemlerinde aynı override’ı uygular. Test assertion’ları değiştirilmedi. Yeni imajlar build edildi, temiz DB stack’i sağlıklı başladı.
- Tarife: seed `AktarBilet Simülasyon Tarifesi` olarak ayrıdır; resmî tarife ile karıştırılmadı. Yaşlı 60+, öğrenci/genç ve ücretsiz sınıfları ayrıdır. Tarihsel finansal kayıtlar değiştirilmedi, kaynağı olmayan tutar oluşturulmadı. Resmî tarifeli uçtan uca kabul kanıtı eksik.
- Ana DB önceki salt-okunur ölçümü: 450 aktif İzmir hattı, 11.010 durak, 200.008 kart, ledger/bakiye farkı 0; dört demo hesabı/kartı tutarlı. ESHOT verisinde sıralı route-stop dizisi olmadığından hat-durak sırası uydurulmadı.
- README’nin korunan beş soru-cevap bölümü HEAD ve çalışma ağacında birebir aynı, SHA-256 `89352298af1f3270fe9ca856adba93e46df0e97bd5a7bf9115885024827721a6` (3.295 byte); son satır canary doğru. `git diff --check` PASS.
- `gh auth status` PASS. En son `main` CI run `41056498730` başarılıdır fakat bu kirli çalışma ağacını doğrulamaz. Yerel zorunlu koşullar başarısız olduğundan commit/push yapılmadı; güncel CI run yok.
- Güncel tam matris ve kanıt sınırları: `qa/reports/CODEX_FINAL_DELIVERY_ACCEPTANCE.md`. Karar: `PROJE_MD_ACCEPTANCE: NOT_READY`, `DELIVERY_STATUS: NOT_READY`.

## 2026-10-30 — Playwright ve kabul toparlama

- Önceki tam suite sonucu 127 PASS / 30 FAIL idi; en eski acceptance raporunda 138/19 vardı. Hata context’leri tek tek sınıflandı ve `qa/reports/CODEX_FINAL_DELIVERY_ACCEPTANCE.md` içinde 30 test adı/mesajı ile düzeltme kanıtı listelendi.
- QA Compose servis DNS alias’ları proje bazında ayrıldı; frontend/API testleri aynı projeye yöneltildi, outage/restart komutlarına override ve port ortamı verildi. Temiz `aktarbilet-finalaccept9` DB’de tam Playwright **157 PASS / 0 FAIL / 0 SKIP / 0 FLAKY**, 217 saniye. JSON kanıt `/tmp/aktarbilet-playwright-final-clean.json`.
- Düzeltmeler: fare report fixture gerçek validator atamasına hizalandı; storno report QA dönemini kapatıyor; anomaly fixture geçerli validator ataması ve imkânsız hız kullanıyor; eski İzmir tariff ID kullanılıyor; lifecycle testi dinamik dönem açılışını ve kapalı→açık sırasını doğruluyor. Browser UAT gerçek rol login’i/form akışını izliyor. Light success rengi axe kontrast bulgusunu gideriyor. Assertion’lar gevşetilmedi.
- Son testler: `go test -count=1 ./...` PASS; `go vet ./...` PASS; domain kapsamı 592/764 = **77.5%** (proje alt sınırı ≥70%); frontend lint 0 error/0 warning ve build 53 modül PASS; `git diff --check` PASS. Yedi zorunlu canary Playwright kontrolü PASS.
- Race bu son kaynak durumu için doğrulanmadı: hostta gcc yok; Go Docker testleri eşzamanlı çalıştırılınca Docker modül-cache I/O/SIGBUS aldı ve WSL Docker CLI bağlantısı sonrasında kullanılamadı. `make` de WSL’de kurulu değil. `go run ./cmd/verify` eşdeğeri geçerli QA portuna yönlendirildi fakat tam suite sonunda QA backend kapalı olduğundan connection refused aldı; F/T live matrisi bu turda PASS sayılmıyor.
- Korumalı README soru/cevap bölümü HEAD ile birebir aynı, SHA-256 `a279402b114fd6b02891df55f4cb1e36a9c6537509dc8331a0f90aaab249fc26` (6.525 byte). Bu tur README’ye dokunulmadı; final canary doğru.
- Açık kabul: route-stop sırası ve resmî yürürlük tarihli tarife kanıtı, güncel OpenAPI/k6/chaos ve tam F/T alt maddeleri suite ile kapanmadı. Bu nedenle **PROJE_MD_ACCEPTANCE: NOT_READY**, **DELIVERY_STATUS: NOT_READY**. Commit/push ve güncel CI yapılmadı; kullanıcı değişiklikleri korunuyor.

---

*Antigravity — Principal Developer*  
*DDO Case Kodu: SEDIR-f85f26*
