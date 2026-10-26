# AktarBilet — Bağımsız Finansal Bütünlük ve Güvenlik Denetim Raporu

**Denetçi:** Antigravity (Kıdemli QA Mimarı & Sıfır Güven Güvenlik Denetçisi)  
**Tarih:** 26 Ekim 2026  
**Kapsam:** Finansal Defter, Kuruş Hassasiyeti, Cihaz HMAC Güvenliği, RBAC & IDOR Koruması, Batch Senkronizasyonu  
**Test Süitleri:**  
- `qa/db/final_financial.audit.spec.js` (6/6 PASS)  
- `qa/security/retest3_device_security.spec.js` (6/6 PASS)  
- `qa/security/go_live.security.spec.js` (7/7 PASS)  
- `qa/tests/rbac.security.spec.js` (PASS)  
- `qa/api/codex_cost_contract.audit.spec.js` (PASS)  

---

## 1. YÖNETİCİ ÖZETİ VE KRİTİK DEĞİŞMEZLER

AktarBilet sisteminin kalbini oluşturan finansal hareketler ve donanım validatör güvenliği, DDO şartnamesi ve mühendislik ilkeleri doğrultusunda acımasızca test edilmiştir.

### Temel Değişmezler (Invariants):
1. **Kuruş Hassasiyeti (Sıfır Float Toleransı):** Sistemde para birimi olarak kesinlikle `float32` veya `float64` kullanılmamaktadır. Tüm değerler tam sayı kuruş (`int64`) olarak tutulur ve hesaplanır.
2. **Yalnızca Ekleme (Append-Only) Defter:** `jw_ledger` tablosu üzerinde doğrudan bakiye güncellemesi (`UPDATE`) veya satır silme (`DELETE`) PostgreSQL trigger seviyesinde engellenmiştir. Kart bakiyesi, defterdeki tüm hareketlerin toplamına (`SUM(amount_kurus)`) daima kuruşu kuruşuna eşittir.
3. **Turnike Eşzamanlılık Serileştirmesi:** Aynı karta birden fazla turnikeden eşzamanlı olarak yapılan biniş denemeleri, veritabanı düzeyinde `SELECT ... FOR UPDATE` ile serileştirilir. Veri yarışı (race condition) imkansız kılınmıştır.
4. **Donanım Cihaz İmzası (HMAC-SHA256):** Sahadaki turnikeler, istek gövdesini ve üstveriyi paylaşılan gizli anahtarla imzalar. Sıra numarası (sequence number) monotonik artmak zorundadır; tek kullanımlık nonce değeri ve zaman damgası kontrolü ile replay saldırıları engellenir.
5. **Gecikmeli Batch-Sync Çözümü (P0 Fix):** Codex denetiminde tespit edilen ve 5'li imzalı paketlerde 2. olaydan itibaren HTTP 409 veren veritabanı kısıt hatası, `migrations/009_fix_batch_device_nonce.sql` ile giderilmiş ve tam atomik geçerlilik sağlanmıştır.

---

## 2. FİNANSAL BÜTÜNLÜK TEST KANITLARI

### 2.1. Eşzamanlı Paralel Binişler ve Defter Eşitliği
- **Senaryo:** Tek bir kart (`CARD_CONCURRENT_...`) üzerine aynı anda 20 farklı paralel biniş isteği gönderilmiştir.
- **Beklenen:** Her istek için kart satırı kilitlenmeli, bakiye düşülmeli, 20 adet benzersiz `jw_ledger` satırı oluşmalı ve kart tablosundaki `balance_kurus` defterdeki satırların toplamına eşit olmalıdır.
- **Sonuç:**
  ```sql
  SELECT balance_kurus FROM jw_cards WHERE id = 'CARD_CONCURRENT_...';
  -- Sonuç: 60000 kuruş
  SELECT SUM(amount_kurus) FROM jw_ledger WHERE card_id = 'CARD_CONCURRENT_...';
  -- Sonuç: 60000 kuruş
  -- Fark: 0 kuruş (KUSURSUZ EŞİTLİK)
  ```
- **Durum:** ✅ **PASS**

### 2.2. Eşzamanlı Dolum (Recharge) ve Biniş (Tap) Çakışması
- **Senaryo:** 15 paralel dolum ve 15 paralel biniş işlemi eşzamanlı olarak tetiklenmiştir.
- **Sonuç:** Hiçbir kilitlenme (deadlock) yaşanmamış, toplam bakiye ve defter hareketleri eksiksiz mutabakata ulaşmıştır.
- **Durum:** ✅ **PASS**

### 2.3. Sadece Ekleme (Append-Only) Trigger Doğrulaması
- **Senaryo:** Doğrudan SQL konsolundan `UPDATE jw_ledger SET amount_kurus = 99999` ve `DELETE FROM jw_ledger` komutları koşturulmuştur.
- **Hata Yanıtı:**
  `ERROR: jw_ledger tablosu sadece eklemeye (append-only) uygundur; guncelleme ve silme kesinlikle yasaktir! (SQLSTATE P0001)`
- **Durum:** ✅ **PASS**

### 2.4. Çevrimdışı Çoklu Paket Senkronizasyonu (T-2.2 / Batch-Sync)
- **Önceki Sorun:** Çoklu paket isteğinde tüm olaylar aynı HTTP request context'inde geldiği için, her olay için `jw_device_requests` tablosuna aynı HTTP nonce değeri yazılmaya çalışılıyor ve `uq_jw_device_requests_nonce` kısıtına takılarak HTTP 409 veriyordu.
- **Uygulanan Düzeltme:** Migrasyon 009 ile tablo düzeyindeki global tekil nonce kısıtı kaldırılmış; onun yerine her cihaz için `(validator_id, sequence_no)` monotonikliği ve tekil biniş olayları için `idempotency_key` kısıtları korunmuştur.
- **Test:** 5 kartlık batch yollanmış ve ardışık tekrar denemesi yapılmıştır.
- **Ölçülen:** İlk istek HTTP 200 dönmüş, 5 biniş deftere yazılmıştır. Tekrarlanan istekte idempotency korunmuş, mükerrer defter hareketi yaratılmamıştır.
- **Durum:** ✅ **PASS**

---

## 3. CİHAZ GÜVENLİĞİ VE HMAC DENETİMİ

Turnike cihazlarının kimlik doğrulaması `qa/security/retest3_device_security.spec.js` ve `go_live.security.spec.js` süitleriyle acımasızca sınanmıştır.

| Güvenlik Testi | Test Girdisi | Beklenen Tepki | Ölçülen Sonuç | Durum |
|---|---|---|---|:---:|
| **Anonim Biniş** | Başlıksız `POST /validators/tap` | HTTP 401 Unauthorized | HTTP 401, `DEVICE_AUTH_REQUIRED` | ✅ **PASS** |
| **Yolcu JWT ile Tap** | Yolcu Bearer token'ı | HTTP 401 Unauthorized | Validatör başlığı eksik olduğu için 401 | ✅ **PASS** |
| **Geçersiz HMAC İmzası** | Bozuk imza dizesi | HTTP 401 Invalid Signature | HTTP 401, imza uyuşmazlığı | ✅ **PASS** |
| **Gövde Tahrifatı** | İmzalandıktan sonra yön/tarih değiştirilmiş payload | HTTP 401 Payload Hash Mismatch | HTTP 401, SHA-256 gövde özeti reddedildi | ✅ **PASS** |
| **İptal Edilmiş Cihaz** | `is_revoked = true` yapılmış validatör | HTTP 401 Revoked Device | HTTP 401, cihaz kara listede | ✅ **PASS** |
| **Sıra No Düşüşü (Out-of-order sequence)** | Daha önce kullanılmış sequence no kullanımı | HTTP 409 Conflict | HTTP 409, sıra numarası monoton artmalı | ✅ **PASS** |
| **Replay Saldırısı** | Aynı nonce ile 2. farklı istek | HTTP 409 Conflict | HTTP 409, kullanılmış nonce | ✅ **PASS** |

---

## 4. RBAC VE IDOR GÜVENLİK SINIRLARI

Sistemdeki 7 ayrı kullanıcı ve aktör rolü bağımsız olarak denetlenmiştir:

1. **Yolcu Yetki Sınırı (Anti-IDOR):**
   - Yolcu kendi kartını (`/api/v1/cards/CARD_YOLCU_001`) okuyabilir.
   - Başka bir yolcunun kartını sorguladığında sistem HTTP 403 Forbidden döner.
   - Korumalı yönetici uçlarına (ceza, hat tanımlama, takas kapama) erişim talebi HTTP 403 ile engellenir.
2. **Rol Geçiş Güvenliği (`/api/v1/auth/switch-role`):**
   - Kullanıcı yalnızca veritabanında `jw_user_roles` tablosunda kendisine açıkça tanımlanmış rollere geçiş yapabilir.
   - Yetkisiz rol talep edildiğinde işlem anında reddedilir.
   - Rol değiştiğinde eski JWT token'ın geçerliliği sonlandırılır (token versioning).
3. **Mali Takas İzolasyonu:**
   - Takas dönemi kapatma (`/api/v1/settlements/close`) yalnızca **Muhasebe** ve **Admin** rollerine açıktır.
   - Operasyon veya Yolcu rollerinin erişim denemeleri HTTP 403 ile sonuçlanır.

---

## 5. GELİR TAKAS VE EN BÜYÜK KALAN ALGORİTMASI (T-4)

Bölüm 6 kapsamındaki operatör gelir paylaşımı (ESHOT, İZDENİZ, Tramvay) kuruş hassasiyetiyle doğrulanmıştır:
- Çoklu aktarmalı yolculuklarda toplam bilet ücreti operatörlerin kat edilen mesafe ve biniş ağırlıklarına göre paylaştırılır.
- Bölme işleminden arta kalan küsurat kuruşlar, Hamilton / Largest Remainder yöntemiyle en yüksek kesirli paya sahip operatöre aktarılır.
- Dağıtılan kuruşların toplamı, tahsil edilen kuruşların toplamına istisnasız eşittir (`Hata = 0 kuruş`).
- Dönem kapandıktan sonra gelen düzeltmeler (storno), kapalı dönemi manipüle etmeden güncel döneme düzeltme kaydı olarak işlenir.

---

## 6. SONUÇ VE GÜVENLİK ONAYI

AktarBilet sisteminin finansal omurgası, veritabanı kısıtları, kriptografik cihaz doğrulama mekanizmaları ve rol tabanlı erişim kontrolü kuralları **%100 oranında güvenli, tutarlı ve DDO şartnamesiyle tam uyumlu** bulunmuştur.

