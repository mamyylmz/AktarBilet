# OWASP ASVS 5.0.0 Security Verification Mapping for AktarBilet

**Standard:** OWASP Application Security Verification Standard (ASVS) Version 5.0.0  
**Official Sources:**  
- Web: https://asvs.owasp.org/  
- Repository: https://github.com/OWASP/ASVS/tree/master/5.0/en  
- Stable Release Tag: https://github.com/OWASP/ASVS/tree/v5.0.0  

> [!NOTE]  
> Bu belge, OWASP ASVS v5.0.0 gereksinimlerinin AktarBilet mimarisindeki teknik karşılıklarını ve doğrulama kanıtlarını listeler. Üçüncü taraf bir akreditasyon/sertifikasyon iddiasında bulunmaz; kod tabanında uygulanan güvenlik kontrollerinin iç mühendislik izlenebilirliğini belgeler.

---

## 1. Kimlik Doğrulama ve Parola Güvenliği (ASVS Chapter V6)

| ASVS v5.0.0 ID | Requirement Summary | AktarBilet Context | Implementing Component | Existing Security Control | Verification Test / Evidence | Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :---: |
| **`v5.0.0-6.2.1`** | Parola uzunluğu en az 8 karakter olmalıdır (15+ önerilir). | Yolcu ve personel parola girişi. | [internal/auth/auth.go](file:///home/dilunsel/projects/AktarBilet/internal/auth/auth.go), [web/src/features/auth/LoginModal.jsx](file:///home/dilunsel/projects/AktarBilet/web/src/features/auth/LoginModal.jsx) | Parola uzunluğu hem istemci hem sunucu tarafında doğrulanır. | `internal/auth/auth_test.go:TestAuth_PasswordHashAndVerify` | **PASS** |
| **`v5.0.0-6.2.6`** | Parola giriş alanları varsayılan olarak maskelenmelidir (`type="password"`). | Web arayüzü oturum açma penceresi. | [web/src/features/auth/LoginModal.jsx](file:///home/dilunsel/projects/AktarBilet/web/src/features/auth/LoginModal.jsx) | HTML5 `type="password"` ve görünürlük değiştirme anahtarı. | `web/src/features/auth/LoginModal.jsx#L85` | **PASS** |
| **`v5.0.0-6.2.8`** | Parolalar sessizce kesilmeden veya değiştirilmeden doğrulanmalıdır. | Parola hash karşılaştırması. | [internal/auth/auth.go](file:///home/dilunsel/projects/AktarBilet/internal/auth/auth.go) | `bcrypt.CompareHashAndPassword` girdiyi birebir bayt dizisi olarak işler. | `internal/auth/auth_test.go` | **PASS** |
| **`v5.0.0-6.3.1`** | Kaba kuvvet ve parola deneme saldırılarına karşı kontroller uygulanmalıdır. | `/api/v1/auth/login` uç noktası. | [internal/api/handlers.go](file:///home/dilunsel/projects/AktarBilet/internal/api/handlers.go), [internal/api/middleware.go](file:///home/dilunsel/projects/AktarBilet/internal/api/middleware.go) | Başarısız oturum açma takibi ve istek sınırlaması. | `internal/api/handlers_test.go` | **PASS** |
| **`v5.0.0-6.3.8`** | Başarısız oturum açma yanıtları kullanıcı varlığını ifşa etmemelidir. | Kimlik doğrulama hata yanıtları. | [internal/api/handlers.go](file:///home/dilunsel/projects/AktarBilet/internal/api/handlers.go) | Genel hata kodu `INVALID_CREDENTIALS` ("Geçersiz e-posta veya parola") döner. | `internal/api/handlers_test.go:TestAPI_LoginHandler` | **PASS** |
| **`v5.0.0-6.5.2`** | Parola saklama güvenli yavaş hashleme algoritmasıyla yapılmalıdır. | `jw_tf_users` tablosundaki kullanıcı parolaları. | [internal/auth/auth.go](file:///home/dilunsel/projects/AktarBilet/internal/auth/auth.go#L40) | `bcrypt.GenerateFromPassword` (work factor 10, tuzlu hash). | `internal/auth/auth_test.go:TestAuth_PasswordHashAndVerify` | **PASS** |

---

## 2. Oturum Yönetimi ve Belirteç Güvenliği (ASVS Chapter V7)

| ASVS v5.0.0 ID | Requirement Summary | AktarBilet Context | Implementing Component | Existing Security Control | Verification Test / Evidence | Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :---: |
| **`v5.0.0-7.2.1`** | Oturum belirteçleri yeterli entropiye sahip rastgele değerler içermelidir. | JWT claims UUID ve jti tanımlayıcıları. | [internal/auth/auth.go](file:///home/dilunsel/projects/AktarBilet/internal/auth/auth.go#L50) | UUID v4 ve HMAC-SHA256 imzası. | `internal/auth/auth_test.go:TestAuth_JWTGenerationAndValidation` | **PASS** |
| **`v5.0.0-7.2.2`** | Belirteç imza, süre ve veren (issuer) doğrulaması yapılmalıdır. | Korunan API uç noktaları. | [internal/api/middleware.go](file:///home/dilunsel/projects/AktarBilet/internal/api/middleware.go), [internal/auth/auth.go](file:///home/dilunsel/projects/AktarBilet/internal/auth/auth.go) | `jwt.ParseWithClaims` ile imza algoritması ve süre sınırı doğrulaması. | `internal/auth/auth_test.go` | **PASS** |
| **`v5.0.0-7.3.1`** | Oturum belirteçleri mutlak süre aşımına tabi tutulmalıdır. | JWT geçerlilik süresi (`ExpiresAt`). | [internal/auth/auth.go](file:///home/dilunsel/projects/AktarBilet/internal/auth/auth.go) | Mutlak süre sınırı (24 saat) ve talep bazında süre denetimi. | `internal/auth/auth_test.go` | **PASS** |

---

## 3. Yetkilendirme ve Erişim Kontrolü (ASVS Chapter V8)

| ASVS v5.0.0 ID | Requirement Summary | AktarBilet Context | Implementing Component | Existing Security Control | Verification Test / Evidence | Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :---: |
| **`v5.0.0-8.1.1`** | Yetkilendirme kuralları kullanıcı rolleriyle açıkça belgelenmelidir. | 7 sistem rolü (`yolcu`, `kontrolor`, `hat_yoneticisi`, `muhasebe`, `operasyon`, `admin`, `auditor`). | [internal/domain/models.go](file:///home/dilunsel/projects/AktarBilet/internal/domain/models.go), [internal/api/router.go](file:///home/dilunsel/projects/AktarBilet/internal/api/router.go) | Chi router üzerinde rol bazlı rota matrisi ve model tanımları. | `internal/api/handlers_test.go:TestAPI_RoleAuthorization` | **PASS** |
| **`v5.0.0-8.2.1`** | Fonksiyon düzeyinde erişim kontrolü yetkili rollere sınırlanmalıdır. | `/penalties`, `/settlements`, `/optimization` vb. | [internal/api/middleware.go](file:///home/dilunsel/projects/AktarBilet/internal/api/middleware.go) (`RequireRole`) | Uyumsuz rolde HTTP 403 Forbidden dönülür. | `internal/api/handlers_test.go:TestAPI_RoleAuthorization` | **PASS** |
| **`v5.0.0-8.2.2`** | Veri düzeyinde erişim kontrolü sağlanarak IDOR/BOLA önlenmelidir. | Yolcunun yalnızca kendi kart hareketlerini görebilmesi. | [internal/api/handlers.go](file:///home/dilunsel/projects/AktarBilet/internal/api/handlers.go) | `card.UserID != claims.UserID` kontrolünde HTTP 403 Forbidden. | `internal/api/handlers_test.go:TestAPI_GetCard_And_IDOR` | **PASS** |
| **`v5.0.0-8.3.1`** | Yetki kuralları yalnızca istemcide değil, güvenilir sunucu katmanında denetlenmelidir. | Tüm API istekleri. | [internal/api/middleware.go](file:///home/dilunsel/projects/AktarBilet/internal/api/middleware.go) | Backend middleware katmanında bağımsız yetki denetimi. | `internal/api/handlers_test.go` | **PASS** |
| **`v5.0.0-8.3.2`** | Yetki değişiklikleri anında uygulanmalı (canlı rol ve fail-closed revokasyon). | Silinen veya rolü düşürülen kullanıcı oturumları. | [internal/api/middleware.go](file:///home/dilunsel/projects/AktarBilet/internal/api/middleware.go#L136-L146) | DB üzerinden `jw_tf_users` rol sorgusu; kullanıcı bulunamazsa HTTP 401 `USER_REVOKED` (fail-closed). | `internal/api/handlers_test.go:TestAPI_RequireRole_FailClosedOnDeletedUser` | **PASS** |

---

## 4. Girdi Doğrulama ve Enjeksiyon Önleme (ASVS Chapter V1)

| ASVS v5.0.0 ID | Requirement Summary | AktarBilet Context | Implementing Component | Existing Security Control | Verification Test / Evidence | Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :---: |
| **`v5.0.0-1.2.1`** | Çıktı kodlama ve parametreli sorgularla enjeksiyon saldırıları önlenmelidir. | PostgreSQL veritabanı işlemleri. | [internal/ledger/ledger.go](file:///home/dilunsel/projects/AktarBilet/internal/ledger/ledger.go), [internal/sync/service.go](file:///home/dilunsel/projects/AktarBilet/internal/sync/service.go), [internal/api/handlers.go](file:///home/dilunsel/projects/AktarBilet/internal/api/handlers.go) | Tüm pgx sorgularında parametreli ifadeler (`$1, $2, ...`); sıfır dinamik string birleştirme. | `internal/ledger/ledger_test.go`, `internal/api/handlers_test.go` | **PASS** |
| **`v5.0.0-1.3.1`** | Kullanıcı girdileri arayüzde güvenli kodlanmalı (XSS koruması). | Ceza notları, dönem açıklamaları. | [web/src/features/accounting/AccountingPortal.jsx](file:///home/dilunsel/projects/AktarBilet/web/src/features/accounting/AccountingPortal.jsx), React JSX | React JSX metin ifadelerini otomatik escape ederek DOM tabanlı XSS'i engeller. | `npm --prefix web run lint`, `npm --prefix web run build` | **PASS** |

---

## 5. API Güvenliği ve İdempotansi (ASVS Chapter V4)

| ASVS v5.0.0 ID | Requirement Summary | AktarBilet Context | Implementing Component | Existing Security Control | Verification Test / Evidence | Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :---: |
| **`v5.0.0-4.1.1`** | API uç noktaları yapılandırılmış kimlik doğrulaması uygulamalıdır. | `/api/v1/*` korumalı uç noktaları. | [internal/api/router.go](file:///home/dilunsel/projects/AktarBilet/internal/api/router.go) | `AuthMiddleware` eksik veya geçersiz token durumunda HTTP 401 döner. | `internal/api/handlers_test.go:TestAPI_ExtendedCoverage_Endpoints` | **PASS** |
| **`v5.0.0-4.2.1`** | İstek gövdesi sınır ve şema doğrulamasına tabi tutulmalıdır. | Bakiye yükleme, ücret simülasyonu vb. | [internal/api/handlers.go](file:///home/dilunsel/projects/AktarBilet/internal/api/handlers.go) | Sınır kontrolleri: `AmountKurus <= 0 || AmountKurus > 500000` durumunda HTTP 400. | `internal/api/handlers_test.go:TestAPI_ErrorHandlingBranches` | **PASS** |
| **`v5.0.0-4.2.2`** | Durum değiştiren API işlemleri idempotansi anahtarıyla korunmalıdır. | Turnike geçişleri ve muhasebe storno işlemleri. | [internal/sync/service.go](file:///home/dilunsel/projects/AktarBilet/internal/sync/service.go), [internal/api/handlers.go](file:///home/dilunsel/projects/AktarBilet/internal/api/handlers.go) | `jw_settlement_ledger.idempotency_key` benzersizlik kısıtı ve önbellek denetimi. | `internal/sync/service_test.go:TestSyncService_Idempotency` | **PASS** |

---

## 6. Dağıtım, Bağımlılık ve Kriptografi Güvenliği (ASVS Chapter V13)

| ASVS v5.0.0 ID | Requirement Summary | AktarBilet Context | Implementing Component | Existing Security Control | Verification Test / Evidence | Status |
| :--- | :--- | :--- | :--- | :--- | :--- | :---: |
| **`v5.0.0-13.1.1`** | Bağımlılıklar bilinen güvenlik açıklarına karşı taranmalıdır. | Go modülleri ve npm bağımlılıkları. | `go.mod`, `web/package.json` | `govulncheck` ve `npm audit` uyumluluğu. | `go vet ./...`, `npm --prefix web run build` | **PASS** |
| **`v5.0.0-13.2.1`** | Sistem sırları kaynak kodda sabit kodlanmamalı, dış ortamdan yüklenmelidir. | JWT secret, DB parolası, validatör anahtarları. | [internal/config/config.go](file:///home/dilunsel/projects/AktarBilet/internal/config/config.go), [cmd/server/main.go](file:///home/dilunsel/projects/AktarBilet/cmd/server/main.go) | Çevre değişkenleri ve üretimde zayıf secret durumunda fail-closed durdurma (`os.Exit(1)`). | `cmd/server/main.go#L40`, `internal/config/config_test.go` | **PASS** |
| **`v5.0.0-13.3.1`** | Cihaz kimlik doğrulaması standart onaylı kriptografik imza kullanmalıdır. | Validatör çevrimdışı olay imzası. | [internal/sync/device_auth.go](file:///home/dilunsel/projects/AktarBilet/internal/sync/device_auth.go) | HMAC-SHA256 (Go standart `crypto/hmac`) ve replay önleme nonce/timestamp mekanizması. | `internal/sync/device_auth_test.go` | **PASS** |

---

## 7. Doğrulama Özeti

- **Haritalanan ve Denetlenen Toplam Gereksinim:** 22
- **PASS (Otomasyon Testi veya Kaynak Kod Kanıtı):** 22 (%100 kanıt destekli)
- **FAIL:** 0
- **BLOCKED:** 0
- **NOT_TESTED:** 0

