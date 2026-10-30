# AktarBilet — Kontrollü Dosya Temizliği ve Envanter Raporu (AGY Controlled Cleanup)

**Tarih:** 30 Ekim 2026  
**Yazar:** Antigravity (Ana Geliştirici Yapay Zeka)  
**Kapsam:** Güvenli dosya temizliği, ölü kod arındırma, sprint taslaklarının arşivlenmesi ve dosya bütünlüğünün korunması.

---

## 1. Temizlik İlkeleri ve Korunan Varlıklar

Temizlik çalışması şartnamede belirtilen **"Brutal Deletion Yapmadan Temizlik"** ilkesi doğrultusunda titizlikle yürütülmüştür:
1. **Canariler ve Proje Değişmezleri:** Hiçbir canari veya kuruş hassasiyetli finansal mantık etkilenmemiştir (`jw_` öneki, `jw_tf_users`, `X-DDO-Stamp: f85f26`, `GET /_hz/f85f`, `seed.sedir@ddo-case.invalid`, `ddo-pelikan-api`, `DDO Case Kodu: SEDIR-f85f26` eksiksiz korunmaktadır).
2. **README ve Şartname Atıfları:** `README.md` içinde referans verilen `T1_INITIAL_FAILURES.md`, `CODEX_FINAL_PROJE_MD_ACCEPTANCE.md` ve `AGY_FINAL_PROJE_MD_ACCEPTANCE.md` raporları ana `qa/reports/` dizininde aynen korunmuştur.
3. **Canlı Test ve Üretim Kodu:** Tüm backend API uçları, Go testleri, SQL migrasyonları ve React frontend bileşenleri sağlam ve işler durumdadır.

---

## 2. Arındırılan Ölü Kodlar ve Korunan Yardımcılar

| Dosya Yolu | Bileşen / Sembol | Tür | Gerekçe | Durum |
| :--- | :--- | :--- | :--- | :--- |
| `internal/sync/service.go` | `ErrDuplicateEvent` | Hata Değişkeni | İdempotent olaylarda hata dönmek yerine mevcut olayı döndüren sözleşme nedeniyle referanssız kalan hata tanımı. | **Kaldırıldı** |
| `internal/config/config.go` | `getEnvInt(key, defaultVal)` | Fonksiyon | Ortam değişkeni ayrıştırma yardımcısı; `TestConfig_GetEnvInt` birim testi ile tam kapsam dahilinde korundu. | **Korundu** |

---

## 3. Kaldırılan Geçici Script ve Kalıntılar

| Dosya Yolu | Tür | Gerekçe | Durum |
| :--- | :--- | :--- | :--- |
| `qa/scripts/capture_final_product_screenshots.js` | Script | Eski manuel ekran görüntüsü alma betiği (hiçbir CI/test referansı yok). | **Kaldırıldı** |
| `qa/scripts/capture_footer_screenshots.mjs` | Script | Eski footer ekran görüntüsü alma betiği. | **Kaldırıldı** |
| `qa/scripts/capture_screenshots.js` | Script | Eski genel screenshot betiği. | **Kaldırıldı** |
| `qa/artifacts/*.log` | Log | Önceki test koşularından kalan geçici loglar (`playwright.log`, `go-tests.log`, vb.). | **Temizlendi** |

---

## 4. Arşivlenen Ara/Eski Taslak Raporlar (`qa/reports/archive/`)

Sprint başlangıcındaki ön taslaklar ve son kabul raporları (`CODEX_FINAL_PROJE_MD_ACCEPTANCE.md` ve `AGY_FINAL_PROJE_MD_ACCEPTANCE.md`) ile yerini nihai sürümlere bırakmış ara dokümanlar, tarihsel denetim izini kaybetmemek adına `qa/reports/archive/` altına güvenle taşınmıştır:

- `BASELINE.md`, `BUGS.md`, `CI_DIAGNOSIS.md`
- `CODEX_BUGS.md`, `CODEX_FINAL_AUDIT.md`, `CODEX_RBAC_AUDIT.md`, `CODEX_VERIFY_AUDIT.md`
- `DELETION_CANDIDATES.md`, `FIX_LOG.md`, `GO_LIVE_CHECKLIST.md`
- `PERFORMANCE.md`, `RBAC_MATRIX.md`, `RELEASE_READINESS.md`
- `REQUIREMENTS_TRACEABILITY.md`, `TEST_AUDIT.md`, `UAT_RESULTS.md`
- `AGY_ACCEPTANCE_MATRIX.md`, `AGY_CODEX_HANDOFF.md`, `AGY_FINAL_IMPLEMENTATION.md`
- `AGY_PERFORMANCE_FINAL.md`, `AGY_SECURITY_FINAL.md`, `AGY_UX_ACCESSIBILITY_FINAL.md`

Bu işlem sonucunda `qa/reports/` dizini sadeleştirilmiş, yalnızca aktif kabul matrisleri, T-1 analizi ve nihai devir raporları ön planda kalmıştır.
