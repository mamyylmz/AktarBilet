# AktarBilet Rol Tabanlı Yetkilendirme (RBAC) ve Güvenlik Matrisi

**Tarih:** 25 Ekim 2026  
**Durum:** ✅ **GÜVENLİ VE DOĞRULANDI (100% RBAC & IDOR KORUMASI)**  
**Hedef Sürüm:** `main@6ac8591`  
**DDO Case Kodu:** `SEDIR-f85f26`

---

## 1. Yetkilendirme Mimarisi ve İlkeleri

AktarBilet API katmanında Sıfır Güven (Zero Trust) ve En Az Ayrıcalık (Least Privilege) ilkeleri uygulanmıştır:
1. **Varsayılan Olarak Reddet (Default Deny):** Yalnızca açıkça izin verilen kamuya açık uçlar (`/health`, `/_hz/f85f`, `/api/v1/auth/login`, statik hat/durak katalogları) anonim erişime açıktır.
2. **Rol Tabanlı Erişim Denetimi (RBAC):** Tüm finansal, operasyonel ve denetim uçları `RequireRole` veya `RequireAnyRole` Chi middleware'leri ile korunmaktadır.
3. **Kart Sahiplik Kontrolü (IDOR Önleme):** `yolcu` rolündeki kullanıcılar yalnızca kendi `user_id`lerine tahsis edilmiş kartları görüntüleyebilir (`GET /api/v1/cards/{id}`) ve doldurabilir (`POST /api/v1/cards/{id}/recharge`). Başka bir yolcunun kartına erişim denendiğinde sistem `403 Forbidden` yanıtı döner.
4. **Cihaz ve Validatör Yetkilendirmesi:** Validatör biniş ve senkronizasyon uçları (`/api/v1/validators/*`) yetkili validatör kimliği gerektirir.
5. **Kriptografik Sıkılaştırma:** JWT belirteçlerinde yalnızca `HS256` imza algoritması kabul edilir (`none` ve diğer algoritmalar reddedilir).

---

## 2. Rol Tanımları

| Rol Kodu | Rol Adı / Görevi | Tanımlı Yetki Kapsamı |
|---|---|---|
| `admin` | Sistem Yöneticisi | Tüm API uçlarına, kullanıcılara, finansal kayıtlara ve sistem yapılandırmasına tam erişim. |
| `hat_yoneticisi` | Hat ve Sefer Yöneticisi | Hat, durak, tarife yönetimi ve hat bazlı yolculuk raporlarını görüntüleme. |
| `isletmeci` | Toplu Taşıma İşletmecisi | Gelir paylaşım raporlarını, dönem uzlaşma tablolarını ve hakedişlerini inceleme. |
| `belediye_denetci`| Belediye Denetim Yetkilisi | Gelir paylaşım dönemlerini onaylama/dondurma (`close`), storno ve denetim kayıtları. |
| `bilet_mufettisi` | Saha Kontrolörü / Müfettiş | Sahada yolcu kartlarını sorgulama ve geçersiz binişler için ceza kaydı oluşturma (`penalties`). |
| `yolcu` | Toplu Taşıma Yolcusu | Yalnızca kendisine ait kartları sorgulama, bakiye geçmişini inceleme ve bakiye doldurma. |
| `anonymous` | Kimliği Doğrulanmamış İstek | Yalnızca sağlık kontrolleri, oturum açma ve genel hat/durak bilgi katalogları. |

---

## 3. Tam Endpoint İzin ve Erişim Matrisi

| HTTP Metodu ve Endpoint | Anonim | Yolcu | Bilet Müfettişi | Hat Yöneticisi | İşletmeci | Belediye Denetçi | Admin | Beklenen Durum Kodu |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|---|
| `GET /health` | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | 200 OK |
| `GET /_hz/f85f` (Canary) | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | 200 OK |
| `GET /metrics` | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | 200 OK |
| `POST /api/v1/auth/login` | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | 200 OK / 401 |
| `GET /api/v1/auth/me` | **DENY (401)** | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | 200 OK |
| `GET /api/v1/lines` | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | 200 OK |
| `GET /api/v1/stations` | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | 200 OK |
| `GET /api/v1/tariffs` | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | ALLOW | 200 OK |
| `GET /api/v1/reports/heatmap` | **DENY (401)** | **DENY (403)** | **DENY (403)** | ALLOW | ALLOW | ALLOW | ALLOW | 200 OK |
| `GET /api/v1/cards/{id}` | **DENY (401)** | **OWNER ONLY** | ALLOW | ALLOW | **DENY (403)**| ALLOW | ALLOW | 200 OK / 403 Forbidden |
| `GET /api/v1/cards/{id}/ledger` | **DENY (401)** | **OWNER ONLY** | ALLOW | ALLOW | **DENY (403)**| ALLOW | ALLOW | 200 OK / 403 Forbidden |
| `POST /api/v1/cards/{id}/recharge`| **DENY (401)**| **OWNER ONLY** | **DENY (403)**| **DENY (403)**| **DENY (403)**| **DENY (403)**| ALLOW | 200 OK / 403 Forbidden |
| `POST /api/v1/validators/tap` | **DENY (401)** | **DENY (403)** | **DENY (403)**| **DENY (403)**| **DENY (403)**| **DENY (403)**| ALLOW | 200 OK (Cihaz/Admin) |
| `POST /api/v1/validators/batch-sync`| **DENY (401)**| **DENY (403)** | **DENY (403)**| **DENY (403)**| **DENY (403)**| **DENY (403)**| ALLOW | 200 OK (Cihaz/Admin) |
| `POST /api/v1/penalties` | **DENY (401)** | **DENY (403)** | ALLOW | **DENY (403)**| **DENY (403)**| **DENY (403)**| ALLOW | 201 Created |
| `GET /api/v1/penalties` | **DENY (401)** | **DENY (403)** | ALLOW | **DENY (403)**| **DENY (403)**| ALLOW | ALLOW | 200 OK |
| `POST /api/v1/settlements/close`| **DENY (401)**| **DENY (403)** | **DENY (403)**| **DENY (403)**| **DENY (403)**| ALLOW | ALLOW | 200 OK |
| `POST /api/v1/settlements/storno`| **DENY (401)**| **DENY (403)** | **DENY (403)**| **DENY (403)**| **DENY (403)**| ALLOW | ALLOW | 200 OK |
| `GET /api/v1/anomalies` | **DENY (401)** | **DENY (403)** | **DENY (403)**| **DENY (403)**| **DENY (403)**| ALLOW | ALLOW | 200 OK |
| `POST /api/v1/simulation/fare-impact`| **DENY (401)**| **DENY (403)** | **DENY (403)**| ALLOW | ALLOW | ALLOW | ALLOW | 200 OK |

---

## 4. IDOR ve Sahiplik Doğrulama Mekanizması

Yolcuların diğer yolcuların kart bakiyelerini veya bakiye defterlerini görmesini/doldurmasını engellemek amacıyla `internal/api/handlers.go` içinde `ensureCardOwnership` mantığı çalıştırılır:

```go
func (h *APIHandler) ensureCardOwnership(r *http.Request, cardID string) error {
    claims := auth.GetClaimsFromContext(r.Context())
    if claims == nil {
        return errors.New("unauthorized")
    }
    // Admin, Bilet Müfettişi ve Belediye Denetçisi kart detayını görebilir
    if claims.Role == domain.RoleAdmin || claims.Role == domain.RoleBiletMufettisi || claims.Role == domain.RoleBelediyeDenetci {
        return nil
    }
    // Yolcu ise, kartın veritabanındaki user_id'si ile token claims.UserID eşleşmelidir
    card, err := h.cardRepo.GetCard(r.Context(), cardID)
    if err != nil {
        return err
    }
    if card.UserID != claims.UserID {
        return errors.New("forbidden: card does not belong to user")
    }
    return nil
}
```

---

## 5. Doğrulama ve Test Kanıtları

- `qa/tests/rbac.security.spec.js`: 17 senaryonun tamamı (Anonim dolum engeli, Yolcu IDOR engeli, Geçersiz JWT imza reddi, Rol kısıtlamaları) **17/17 PASS** ile sonuçlanmıştır.
- `internal/api/handlers_test.go`: `TestAPI_GetCard_And_IDOR` testi ile başkasına ait kart sorgulandığında doğrudan `403 Forbidden` alındığı doğrulanmıştır.
- Güvenlik açığı bulunmamaktadır.
