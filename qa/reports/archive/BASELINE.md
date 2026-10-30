# AktarBilet Başlangıç Durum ve Denetim Raporu (Baseline Audit)

**Tarih:** 25 Ekim 2026  
**Hedef Repository:** `https://github.com/mamyylmz/AktarBilet` (`main` dalı)  
**Değerlendirme Makamı:** Cumhurbaşkanlığı Dijital Dönüşüm Ofisi (DDO) Yazılım Ekibi  
**DDO Case Kodu / Mührü:** `SEDIR-f85f26`  
**Denetçi Rolü:** Antigravity (Senior Software Engineer, QA Automation Engineer, Security & Release Engineer)

---

## 1. Yönetici Özeti ve Denetim Kapsamı

AktarBilet (Şehir İçi Toplu Taşıma Bilet ve Ücretlendirme Sistemi), DDO Yazılım Ekibi şartnamesi doğrultusunda; yüksek eşzamanlılık (concurrency), offline validatör senkronizasyonu, kuruş hassasiyetli bakiye defteri, gelir paylaşım algoritması ve anormallik tespiti yeteneklerini barındıran kritik bir kentsel ulaşım altyapı projesidir.

Bağımsız kalite güvence (Codex) tarafından `main@aceaef8483619f73076c913ba50dfa10f901afd8` commit'inde yürütülen ilk denetimde projenin test kapsamı (%52.4) ve canlı entegrasyon noktalarında 24 adet hata/eksiklik (4 adet P0 Blocker, 12 adet P1 Critical, 8 adet P2 Major) tespit edilmiş ve başlangıç kararı `NOT_READY` olarak raporlanmıştır.

Bu baseline raporu; tespit edilen başlangıç açıklarını, sistem kısıtlarını, test kapsamını, ortam mimarisini ve üretim hazırlığına geçiş öncesindeki temel metrikleri kayıt altına almaktadır.

---

## 2. Repository ve Çalışma Ortamı Bilgileri

| Parametre | Değer / Açıklama |
|---|---|
| **Repository URL** | `https://github.com/mamyylmz/AktarBilet` |
| **İncelenen İlk Commit** | `aceaef8483619f73076c913ba50dfa10f901afd8` (Codex Baseline) |
| **Düzeltme ve Doğrulama Commit'i** | `6ac859132644265f2425cf15632a677103ce53ae` (Antigravity Remediation) |
| **Çalışma Dalı (Branch)** | `main` |
| **Backend Teknolojisi** | Go 1.23.6, Chi Router, pgx/v5 PostgreSQL Driver |
| **Veritabanı** | PostgreSQL 16 (İlişkisel, Trigger ve CHECK kısıtları ile) |
| **Frontend Teknolojisi** | React 18, Vite, Vanilla CSS (Mobil/Tablet duyarlı) |
| **Konteyner Orkestrasyonu** | Docker Compose v2, Çok aşamalı (multi-stage) Dockerfile'lar |
| **İşletim Sistemi / Host** | Linux x86_64 (WSL2 / Ubuntu Kernel 6.6.87.2) |

---

## 3. Başlangıç Durum Test ve Metrikleri (Baseline Metrics)

### 3.1. Go Test Kapsamı (internal/...)
Başlangıç durumunda `internal/...` statement coverage oranı **%52.4** seviyesindeydi ve şartnamedeki **≥%70** eşiğinin altındaydı.

| Paket | Başlangıç Kapsamı | Başlangıç Test Sayısı | Başlıca Eksiklik |
|---|---:|---:|---|
| `internal/anomaly` | %76.7 | 2 | PostgreSQL kalıcılığı ve 300 sn sınır testi eksik |
| `internal/api` | %33.7 | 3 | Finansal ve validatör uçlarında RBAC/IDOR kontrolleri eksik |
| `internal/auth` | %84.6 | 2 | JWT HS256 kesin algoritma denetimi ve rol geçerliliği eksik |
| `internal/config` | %69.2 | 2 | Çevre değişkeni fallback'leri |
| `internal/domain` | %100.0 | 2 | Tam kuruş dönüşüm yardımcıları |
| `internal/ledger` | %43.5 | 3 | `PostgresLedger` metotları %0 (tamamı mocksuz canlı DB bekliyordu) |
| `internal/pricing` | %96.8 | 7 | Çekirdek algoritma sağlam, saat dilimi ve ters yön senaryoları |
| `internal/seed` | %0.0 | 1 | Gerçek seeder çalıştırılmıyordu; sahte sabit testi yapılıyordu |
| `internal/settlement` | %87.1 | 2 | Kuruş dağıtımı iyi, DB periyot kapanış/storno entegrasyonu eksik |
| `internal/sync` | %65.6 | 3 | Memory-only, PostgreSQL transaction ve sıralama eksik |
| **Genel Toplam** | **%52.4** | **27 Test** | **HEDEF: ≥%70 (FAIL)** |

### 3.2. Bağımsız QA Test Paketi (Codex Audit)
Bağımsız Playwright ve API/DB entegrasyon testlerinde 49 senaryodan 30'u başarısız olmuştu:

| Test Kategorisi | Toplam Test | Başarılı (PASS) | Başarısız (FAIL) | Başarı Oranı |
|---|---:|---:|---:|---:|
| API Entegrasyon (`api.integration.spec.js`) | 15 | 10 | 5 | %66.7 |
| Tarayıcı UAT (`browser.uat.spec.js`) | 10 | 1 | 9 | %10.0 |
| Veritabanı Bütünlüğü (`database.integrity.spec.js`) | 7 | 3 | 4 | %42.8 |
| Güvenlik & RBAC (`rbac.security.spec.js`) | 17 | 5 | 12 | %29.4 |
| **Toplam QA Paketi** | **49** | **19** | **30** | **%38.8 (NOT READY)** |

---

## 4. Tespit Edilen Kritik Blokajlar Özeti

1. **AB-QA-001 (P0): Seed İdempotans ve Defter Bozulması**  
   Backend yeniden başlatıldığında seed fonksiyonu mükerrer başlangıç bakiye kayıtları üretiyor, kart bakiyesi defter hareketlerinin toplamından kopuyordu.
2. **AB-QA-002 (P0): Anonim Finansal Erişim ve IDOR**  
   `/api/v1/cards/{id}/recharge` ve bakiye okuma uçları yetkilendirme başlığı olmadan anonim erişime izin veriyor; yolcu rolü başka yolcunun kartını manipüle edebiliyordu.
3. **AB-QA-003 (P0): jw_ledger Tablosunun Append-Only Korunmaması**  
   PostgreSQL düzeyinde `UPDATE` veya `DELETE` işlemlerini engelleyen trigger kısıtı bulunmuyordu.
4. **AB-QA-004 (P1): Validatör ve Simülatör Kalıcılık Eksikliği**  
   Canlı `/api/v1/validators/tap` ve simülatör olayları doğrudan veritabanına yazılmıyor, simülatör 200.000 kart yerine 0 kalıcı olay üretiyordu.
5. **AB-QA-006 (P1): Frontend Sahte (Mock) Başarı Mesajları**  
   Backend hata verdiğinde frontend istemci tarafında bakiye güncelleyip sahte başarılı bildirimleri gösteriyordu.
6. **AB-QA-007 (P1): Gelir Paylaşımı Kapanış ve Storno Hataları**  
   Dönem kapatma işlemi veritabanında `is_closed` bayrağını güncellemiyor, kapatılmış döneme storno düzeltmesi engellenmiyordu.
7. **AB-QA-018 (P2): Responsive Yatay Taşma (Horizontal Overflow)**  
   Mobil (390px) ve tablet (768px) ekranlarda tablolar ve navigasyon çubuğu 979px genişliğe taşarak kullanıcı deneyimini bozuyordu.

---

## 5. İyileştirme ve Go-Live Yol Haritası

Tespit edilen 24 hatanın tamamı için Antigravity tarafından 5 aşamalı mühendislik planı devreye alınmıştır:
- **FAZ 1:** Finansal Bütünlük ve İdempotans (`AB-QA-001`, `AB-QA-003`, `AB-QA-013`)
- **FAZ 2:** Güvenlik, RBAC, IDOR ve Kimlik Doğrulama (`AB-QA-002`, `AB-QA-008`, `AB-QA-017`)
- **FAZ 3:** Çevrimdışı Senkronizasyon, Simülatör ve Kalıcılık (`AB-QA-004`, `AB-QA-005`, `AB-QA-010`, `AB-QA-014`, `AB-QA-015`)
- **FAZ 4:** Frontend Doğruluğu, Responsive Tasarım ve Sahte Durumların Temizlenmesi (`AB-QA-006`, `AB-QA-011`, `AB-QA-018`)
- **FAZ 5:** Test Kapsamının Genişletilmesi ve Kalite Güvencesi (Kapsamın ≥%70'e çıkarılması, `make verify` %100 PASS)

Tüm fazlar tamamlanmış olup sonuçlar ve somut kanıtlar takip eden raporlarda sunulmuştur.

