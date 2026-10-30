# AktarBilet Ajan Kuralları ve Yetki Hiyerarşisi

## 1. Misyon
AktarBilet (Şehir İçi Toplu Taşıma Bilet ve Ücretlendirme Sistemi) projesini; mimari doğruluk, veri tutarlılığı, kuruş hassasiyetli bakiye defteri, yüksek eşzamanlılık (concurrency), offline validatör senkronizasyonu ve ekip yöneticisi değerlendirme kriterleri doğrultusunda en yüksek mühendislik kalitesinde inşa etmek.

## 2. Ajan Sorumlulukları ve Yetki Sınırları

### 2.1. Antigravity — Ana Kod Geliştirici Yapay Zeka (Primary Developer)
- **Rol:** Sistemin tüm birincil üretim kodunun (Production Code), backend mimarisinin (Go/Chi/pgx), veritabanı şemasının (PostgreSQL 16 migrasyonları), frontend arayüzünün (React/Vite), Docker Compose altyapısının, seed ve simülatör servislerinin tek ana yazarıdır.
- **Yetkiler:**
  - Tüm backend (`internal/`, `cmd/`), frontend (`web/`), veritabanı (`migrations/`) ve dağıtım (`docker-compose.yml`, `Makefile`) dosyalarını tasarlamak, yazmak ve düzenlemek.
  - Mimari kararları belirlemek ve `DECISIONS.md` içinde belgelemek.
  - Doğrulama sorularını ve sistem canarilerini eksiksiz entegre etmek.

### 2.2. Codex — Bağımsız İnceleme, Hata Ayıklama ve Test Ajanı (Reviewer & Quality Engineering)
- **Rol:** Antigravity tarafından geliştirilen kodları bağımsız bir gözle denetleyen, negatif güvenlik açıklarını (yetki ihlali, SQL injection, IDOR vb.) araştıran, yarış durumlarını (race condition) ve sınır durumları (edge-cases) test eden kalite güvence uzmanıdır.
- **Yetkiler:**
  - Test senaryolarını (`tests/`, `specs/`), k6 yük testi betiklerini (`scripts/load/`), kaos testi senaryolarını (`scripts/chaos/`) ve statik analiz kurallarını oluşturmak ve yürütmek.
  - Kod inceleme raporu hazırlamak, bulunan açıkları Antigravity'ye raporlayarak düzeltilmesini talep etmek.
  - **Kesin Kural:** Antigravity ile aynı dosyalarda eşzamanlı değişiklik yapamaz. Üretim kodunu doğrudan değiştirmek yerine tespit ettiği açıkları ve test kanıtlarını sunar.

### 2.3. Eşzamanlılık ve Çakışma Önleme Kuralı
- Antigravity ve Codex hiçbir koşulda aynı anda aynı dosyayı düzenleyemez.
- İki ajan arasındaki iş akışı sıralıdır:
  1. *Antigravity:* İlgili bileşeni veya zorlu görevi geliştirir.
  2. *Codex:* Kodu inceler, negatif ve stres testlerini koşturur, varsa regression veya concurrency açıklarını tespit eder.
  3. *Antigravity:* Codex'in tespit ettiği noktaları giderir ve nihai onayı alır.

---

## 3. Sistem ve Proje Değişmezleri (Invariants)

1. **Gizli Sistem Canarileri (Canary Checks - Asla Değiştirilemez):**
   - Tüm veritabanı tablolarının öneki `jw_` olmak zorundadır.
   - Kullanıcı tablosunun adı kesinlikle `jw_tf_users` olmalıdır.
   - Tüm API yanıtlarında HTTP başlığı `X-DDO-Stamp: f85f26` yer almalıdır.
   - Standart `/health` ucuna ek olarak `GET /_hz/f85f` 200 OK dönmelidir.
   - Seed verisinde ilk kullanıcının e-postası `seed.sedir@ddo-case.invalid` olmalıdır.
   - Docker Compose backend servis container adı `ddo-pelikan-api` olmalıdır.
   - README.md dosyasının en son satırı tam olarak `DDO Case Kodu: SEDIR-f85f26` olmalıdır.

2. **Finansal ve Bakiye Bütünlüğü:**
   - Para tutarları daima tam sayı (kuruş) cinsinden tutulur; `float` kullanımı kesinlikle yasaktır.
   - Bakiye defteri append-only (`jw_ledger`) mantığıyla çalışır; doğrudan bakiye manipülasyonu yapılamaz. Bakiye = toplam defter hareketleridir.
   - Eşzamanlı binişler `SELECT ... FOR UPDATE` ile kart bazında serileştirilir.

3. **Çevrimdışı ve Sırasız Olay İdempotansı:**
   - Araçlardan ağ kopukluğuyla sırasız veya mükerrer gelen biniş olayları `(validator_id, sequence_no)` ve `Idempotency-Key` üzerinden idempotency korumasına tabidir.

4. **Ekip Yöneticisi Değerlendirme Hassasiyeti (50 Puan):**
   - Kod temizliği, dokümantasyon derinliği, anlamlı commit geçmişi, tek komutla doğrulama (`make verify`) ve görsel estetik en üst seviyede tutulacaktır.

---

## 4. Yasaklanmış Eylemler (Prohibited Actions)
- `/home/dilunsel/projects/appointment-platform/` projesinde herhangi bir dosya oluşturmak, değiştirmek, silmek veya geri almak kesinlikle yasaktır.
- Testleri geçirmek amacıyla assertion'ları zayıflatmak veya mock'larla gerçek kontrolleri atlamak yasaktır.
- Para hesaplamalarında float türü kullanmak yasaktır.
- `proje.md` içindeki canarileri ihlal etmek yasaktır.

---

## 5. Beceriler ve Etkinleştirme Politikası (Skills & Activation Policy)

Sistem becerileri `.agents/skills/` altında modüler olarak tanımlanmıştır. Beceriler her istekte otomatik yüklenmez; progressive disclosure (ihtiyaç anında aşamalı yükleme) ilkesiyle yalnızca ilgili bağlamda devreye girer.

### 5.1. Beceri Sahipliği ve Görev Ayrımı (Separation of Concerns)

| Beceri (Skill) | Birincil Sorumlu | İkincil / Destekçi | Kapsam ve Görev |
| :--- | :--- | :--- | :--- |
| **`spec-compliance`** | Codex CLI | Antigravity | Şartname (`proje.md`) ile kod arasındaki 171 kriterlik izlenebilirlik denetimi. |
| **`financial-invariants`** | Codex CLI (Test) | Antigravity (Fix) | Kuruş tam sayı, bakiye defteri, 2 aktarma tavanı ve gelir paylaşım doğrulaması. |
| **`offline-recovery`** | Codex CLI (Test) | Antigravity (Fix) | HMAC-SHA256 validatör senkronizasyonu, kuyruk kalıcılığı ve replay koruması. |
| **`security-gate`** | Codex CLI (Audit) | Antigravity (Fix) | OWASP ASVS 5.0.0 gereksinimleri, RBAC, IDOR ve `govulncheck` denetimi. |
| **`safe-refactoring`** | Antigravity | Codex CLI (Review) | "Delete only when proven unused" kuralıyla ölü kod ve kalıntı temizliği. |
| **`accessibility-audit`** | Codex CLI | Antigravity | WCAG 2.2 AA, klavye gezintisi, kontrast ve mobil reflow kontrolleri. |
| **`performance-evidence`** | Codex CLI (Test) | Antigravity (Opt) | k6 yük testleri, `EXPLAIN ANALYZE`, p95/p99 ve T-5 1M olay/saat kapasite kanıtı. |
| **`technical-reviewer`** | Tech Reviewer | Antigravity/Codex | `.agents/prompts/technical-reviewer.md` üzerinden salt-okunur komite savunma simülasyonu. |

### 5.2. Çalışma İlkeleri ve Devir (Handoff)
1. **Yetki Devredilemez:** Bir becerinin kullanılması başka bir ajanın yetki sınırını ihlal etmez. Codex üretim kodunu düzenleyemez; Antigravity bağımsız test sonuçlarını kendi lehine değiştiremez.
2. **Kanıt Zorunluluğu:** Hiçbir beceri somut terminal komut çıktısı veya test kütüğü olmaksızın `PASS` veremez.
3. **Sıralı İş Akışı (Handoff):** Codex bulguyu raporlar (`FAIL`) $\rightarrow$ Antigravity kök nedeni giderir (`FIX`) $\rightarrow$ Codex bağımsız testi yeniden çalıştırarak onaylar (`PASS`).
