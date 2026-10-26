# AktarBilet — Güvenlik, Kriptografi ve Finansal Bütünlük Düzeltme Raporu

**Tarih:** 26 Ekim 2026  
**Hazırlayan:** Antigravity (Principal Security & Go Backend Engineer)  
**Denetleyen:** Codex (Reviewer & Quality Engineering)  
**Durum:** **TAMAMLANDI (Tüm Güvenlik ve Finansal Blokerlar Çözüldü)**

---

## 1. Giriş ve Kapsam

Bu rapor, AktarBilet sisteminde Codex bağımsız denetimleri sonucunda tespit edilen P0 düzeyindeki kriptografik cihaz doğrulama, toplu senkronizasyon çatışması (HTTP 409), finansal modellerde float veri türü kullanımı ve veritabanı kesintisi dayanıklılık açıklarının giderilmesini belgeler.

---

## 2. P0: Çevrimdışı Toplu Paket Senkronizasyonu (HTTP 409 Çözümü)

### 2.1. Kök Neden Analizi
Turnikelerden gelen çevrimdışı toplu biniş paketlerinde (`/api/v1/validators/batch-sync`), istek düzeyinde üretilen tek bir `X-Device-Nonce` değeri tüm paket içeriğindeki olaylar tarafından paylaşılmaktaydı.  
`migrations/006_security_and_approval_integrity.sql` satır 10'da tanımlanmış olan:
```sql
CONSTRAINT uq_jw_device_requests_nonce UNIQUE (device_nonce)
```
kısıtı, aynı istek içerisindeki 2. olay kaydedilmek istendiğinde PostgreSQL düzeyinde `duplicate key value violates unique constraint "uq_jw_device_requests_nonce"` hatası üretiyor ve sistem isteği `ErrReplayDetected` olarak değerlendirip HTTP 409 Conflict dönüyordu.

### 2.2. Düzeltme
1. Nonce tekilliğinin HTTP istek düzeyinde doğrulanması zaten `jw_device_nonces (nonce VARCHAR PRIMARY KEY)` tablosu ile güvence altına alınmıştır.
2. Olay bazlı tekillik ise `jw_device_requests` tablosunun `PRIMARY KEY (validator_id, sequence_no)` bileşik anahtarı ile sağlanmaktadır.
3. `migrations/009_fix_batch_device_nonce.sql` migrasyonu oluşturuldu ve `uq_jw_device_requests_nonce` kısıtı kaldırıldı:
   ```sql
   ALTER TABLE jw_device_requests DROP CONSTRAINT IF EXISTS uq_jw_device_requests_nonce;
   ```
4. Doğrulama: `qa/db/final_financial.audit.spec.js` 5. testi ("batch retry has one event and one financial effect per item") çalıştırılmış ve 5 olaylık toplu paketin tek bir HTTP çağrısıyla başarıyla işlendiği, mükerrer çağrılarda ise idempotency kuralının korunduğu kanıtlanmıştır.

---

## 3. P0: Kriptografik Cihaz Doğrulaması (HMAC-SHA256 & Nonce/Sequence)

### 3.1. Mimari Prensipler
1. Turnike validatörleri API'ye kullanıcı JWT'si ile bağlanamaz. Her validatör (`VAL_500T_01` vb.) bağımsız bir cihaza ait 256-bit gizli anahtar ile yapılandırılmıştır.
2. Turnike kimlik bilgileri kaynak kodda veya frontend paketinde yer almaz; dış ortamdan `VALIDATOR_CREDENTIALS_FILE` çevre değişkeniyle yüklenir.
3. Her istek canonical mesaj formatıyla imzalanır:
   $$\text{Message} = \text{ValidatorID} : \text{SequenceNo} : \text{Timestamp} : \text{Nonce} : \text{PayloadHash}$$
4. PayloadHash, istek gövdesinin tam ham byte dizisinin SHA-256 özetidir. Yön, kart ID, biniş zamanı veya durak değiştirilirse imza derhal geçersiz olur.
5. Cihaz saat kayması (clock skew) ±15 dakika penceresiyle sınırlandırılmıştır.

### 3.2. Test Kanıtı
`qa/security/retest3_device_security.spec.js` ve `qa/security/go_live.security.spec.js` altındaki 13 testin tamamı yeşildir:
- `a VAL_ prefix is not a device credential and failure is atomic`: PASS
- `retired repository credential cannot impersonate a provisioned validator`: PASS
- `signature binds every tap field, including direction and tapped_at`: PASS
- `same sequence with a different signed payload is rejected`: PASS
- `revoked device rejection does not consume nonce or sequence`: PASS

---

## 4. P0: Finansal Modellerde Float Türünün Tamamen Kaldırılması

### 4.1. Kök Neden Analizi
Şartname Bölüm 7 ve Değişmez 2 uyarınca, IEEE 754 kayan noktalı sayı (float) temsili yuvarlama hatalarına neden olduğundan finansal sistemde kesinlikle yasaktır.  
`internal/domain/models.go` içerisinde yer alan `TLToKurus(float64)` ve `(k Kurus) ToTL() float64` metotları float dönüşümü yapmaktaydı.

### 4.2. Düzeltme
1. `TLToKurus` ve `ToTL()` fonksiyonları tamamen silindi.
2. Yerine tam sayı aritmetiği kullanan formatlayıcı eklendi:
   ```go
   func (k Kurus) FormatTL() string {
       sign := ""
       val := int64(k)
       if val < 0 {
           sign = "-"
           val = -val
       }
       return fmt.Sprintf("%s%d.%02d", sign, val/100, val%100)
   }
   ```
3. Tüm API yanıtlarında (örneğin `/api/v1/cards/{id}`), bakiye metinleri `bal.FormatTL()` üzerinden string olarak iletilmektedir.

---

## 5. P0: Dayanıklılık ve Kesintide Güvenli Kapanma (Fail-Closed 503)

### 5.1. Kök Neden Analizi
Önceki mimaride, PostgreSQL veritabanı erişilemediğinde sunucu hafıza içi (in-memory) defter moduna geçip HTTP 200 dönmekteydi. Bu durum dışarıdan sistemin sağlıklı olduğu yanılsamasını yaratıyor ve veri kalıcılığını tehlikeye atıyordu.

### 5.2. Düzeltme
1. `cmd/server/main.go` içerisinde başlangıçta veritabanı havuzuna bağlanılamazsa veya ping atılamazsa:
   - Sunucu hafıza içi sahte moda geçmez (`pool = nil`).
   - HTTP sunucusu başlatılır ancak `/health` ucu derhal HTTP 503 Service Unavailable döner (`status: "DOWN"`).
   - Tüm finansal ve veri mutasyon uçları (`/cards/{id}/recharge`, `/validators/tap`, `/batch-sync`) `h.pool == nil` kontrolü ile HTTP 503 Fail-Closed olarak istekleri reddeder.
2. Test Kanıtı: `qa/api/go_live.api.spec.js` içerisindeki `ARCH-04 a fresh backend must not report healthy without PostgreSQL` testi 17.6 saniyede başarıyla PASS etmiştir.

---

## 6. Özet Test Sonuçları

| Test Paketi | Test Sayısı | Başarılı | Başarısız | Süre |
|---|---|---|---|---|
| `final_financial.audit.spec.js` | 6 | 6 | 0 | 1.0 dk |
| `final_anomaly.audit.spec.js` | 1 | 1 | 0 | 4.6 sn |
| `go_live.db.spec.js` | 6 | 6 | 0 | 15.6 sn |
| `retest3_device_security.spec.js` | 8 | 8 | 0 | 45.1 sn |
| `go_live.security.spec.js` | 5 | 5 | 0 | 8.2 sn |
| `go_live.api.spec.js` | 4 | 4 | 0 | 18.9 sn |

