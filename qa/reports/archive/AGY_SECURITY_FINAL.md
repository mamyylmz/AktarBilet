# AktarBilet — AGY Güvenlik ve Sıkılaştırma Raporu (Security Final)

**Tarih:** 25 Ekim 2026  
**Kapsam:** API Güvenliği, Kriptografik Cihaz Kimlik Doğrulaması, RBAC ve Veritabanı Değişmezleri  

---

## 1. Kriptografik Validatör İmzası ve Replay Koruması

Turnike ve saha validatörlerinden gelen biniş istekleri için HMAC-SHA256 tabanlı donanım seviyesi kimlik doğrulama mekanizması (`DeviceAuthManager`) devreye alınmıştır:

1. **İmza Hesaplama:**
   ```text
   HMAC-SHA256(SecretKey, validator_id + ":" + sequence_no + ":" + timestamp + ":" + nonce + ":" + sha256(payload))
   ```
2. **Replay Önleme:**
   - Her binişte tekil `X-Device-Nonce` değeri bellek içi önbellekte saklanır. Tekrarlanan nonce doğrudan `409 Conflict / REPLAY_DETECTED` ile reddedilir.
3. **Monotonik Sıra Numarası:**
   - Validatörün gönderdiği `sequence_no`, son görülen numaradan büyük olmak zorundadır. Eski veya sırasız paketler düşürülür.
4. **Cihaz İptali (Revocation):**
   - Çalınan veya fiziksel güvenliği aşılan validatör anında `RevokeDevice(validatorID)` ile kara listeye alınır ve `403 Forbidden / DEVICE_REVOKED` döner.
5. **Kesintisiz Anahtar Rotasyonu (Key Rotation):**
   - `RotateKey` mekanizması ile geçerli anahtar değiştirilirken, önceki anahtar belirli bir geçiş süresince desteklenir; saha validatörlerinde kesinti yaşanmaz.

---

## 2. Yetkilendirme (RBAC) ve Anti-IDOR Savunması

1. **Kart Sahiplik Kontrolü (Ownership):**
   - Yolcu rolündeki kullanıcılar yalnızca kendi kullanıcı kimliklerine (`user_id`) bağlı kartları sorgulayabilir veya yükleme yapabilir.
   - Başkasının kart ID'si (`CARD_TAM_001` vb.) ile yapılan istekler veritabanı düzeyinde sahiplik eşleşmediği için kesinlikle `403 Forbidden` alır.
2. **Dinamik Rol Doğrulaması (Stale-Role Defense):**
   - `RequireRole` middleware'i her hassas işlemde kullanıcının veritabanındaki güncel rolünü sorgular (`SELECT role FROM jw_tf_users WHERE id = $1`). Kullanıcının yetkisi düşürülmüşse, eski JWT token'ı hemen reddedilir.
3. **Rol Ayrımı (Separation of Duties):**
   - `kontrolor`: Yalnızca saha denetimi ve ceza tutanağı düzenleyebilir; dönem kapatamaz.
   - `muhasebe`: Yalnızca gelir paylaşım dönemlerini kapatabilir ve storno girebilir; ceza kesemez.
   - `yolcu`: Yalnızca kendi kartını ve halka açık katalogları görebilir.

---

## 3. Finansal Değişmezler ve PostgreSQL Savunması

1. **Append-Only Ledger Koruyucu Trigger:**
   - PostgreSQL üzerinde tanımlanan `prevent_ledger_mutation()` trigger'ı ile `jw_ledger` ve `jw_audit_logs` tabloları üzerinde `UPDATE` ve `DELETE` sorguları veritabanı motoru seviyesinde engellenmiştir (SQLSTATE 55000).
2. **Kuruş Hassasiyeti ve Sıfır Float:**
   - Tüm bakiye, tarife ve takas tutarları 64-bit tam sayı (`Kurus`) cinsinden saklanmaktadır. Float yuvarlama hataları matematiksel olarak olanaksızdır.
3. **Sınırlı Negatif Bakiye Toleransı (-50.00 TL):**
   - Çevrimdışı senkronizasyonda yolcunun bakiyesi -5000 kuruşa (-50.00 TL) kadar düşebilir; bu sınır aşıldığında işlem otomatik olarak reddedilir.

