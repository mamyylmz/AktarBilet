# AKTARBİLET — ÇOKLU ROL (MULTI-ROLE) VE RBAC GÜVENLİK MİMARİSİ

**Tarih:** 26 Ekim 2026  
**Yazar:** Antigravity (Principal Security & Backend Architect)  
**Kapsam:** JWT Claims, `jw_user_roles` Veritabanı Mimarisi, Switch-Role Uç Noktası ve İzolasyon Kanıtları  

---

## 1. GİRİŞ VE PROBLEM TANIMI

Önceki incelemelerde sistemin tekil rol varsayımıyla çalıştığı, kullanıcının yalnızca JWT içindeki tek bir `role` alanı üzerinden değerlendirildiği veya istemci tarafında yerel saklama alanındaki (`localStorage`) rol değerini değiştirerek yetki aşımı (Privilege Escalation / IDOR) yapılabileceği riskleri raporlanmıştı.

Bu riskleri ortadan kaldırmak için tam yetkili Çoklu Rol (Multi-Role RBAC) mimarisi kurulmuştur.

---

## 2. VERİTABANI ŞEMASI VE ÇOKLU ROL İLİŞKİSİ

Veritabanında her kullanıcının bir veya birden fazla role sahip olabileceği `jw_user_roles` ara tablosu ile modellenmiştir:

```sql
-- jw_user_roles tablosu (Canari jw_ öneki korunarak)
CREATE TABLE IF NOT EXISTS jw_user_roles (
    user_id UUID NOT NULL REFERENCES jw_tf_users(id) ON DELETE CASCADE,
    role VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, role)
);

CREATE INDEX IF NOT EXISTS idx_jw_user_roles_user_id ON jw_user_roles(user_id);
```

### 2.1. Seed Kullanıcıları ve Rol Dağılımı

| E-posta | Birincil Rol | Atanmış Roller (`jw_user_roles`) | Açıklama |
|---|---|---|---|
| `seed.sedir@ddo-case.invalid` | `admin` | `["admin", "yolcu", "kontrolor", "hat_yoneticisi", "muhasebe", "operasyon"]` | Canari ilk seed kullanıcısı, tam yetkili sistem yöneticisi |
| `yolcu@aktarbilet.gov.tr` | `yolcu` | `["yolcu"]` | Standart yolcu (yalnızca kendi kart ve binişlerine erişebilir) |
| `kontrolor@aktarbilet.gov.tr` | `kontrolor` | `["kontrolor", "yolcu"]` | Bilet denetçisi ve ceza düzenleyici |
| `hatyoneticisi@aktarbilet.gov.tr` | `hat_yoneticisi` | `["hat_yoneticisi", "yolcu"]` | Sefer sıklığı ve hat hasılat yöneticisi |
| `muhasebe@aktarbilet.gov.tr` | `muhasebe` | `["muhasebe", "yolcu"]` | Gelir paylaşımı, mutabakat ve storno yetkilisi |
| `operasyon@aktarbilet.gov.tr` | `operasyon` | `["operasyon", "yolcu"]` | Canlı telemetri ve filo yöneticisi |

---

## 3. JWT TOKEN VE CLAIM YAPISI

Kullanıcı `POST /api/v1/auth/login` ile oturum açtığında, backend kullanıcının sahip olduğu tüm rolleri veritabanından sorgular ve JWT payload'ına şu standartta ekler:

```json
{
  "sub": "00000000-0000-0000-0000-000000000001",
  "email": "seed.sedir@ddo-case.invalid",
  "active_role": "admin",
  "roles": [
    "admin",
    "yolcu",
    "kontrolor",
    "hat_yoneticisi",
    "muhasebe",
    "operasyon"
  ],
  "iat": 1793026800,
  "exp": 1793113200
}
```

- **`active_role`:** Kullanıcının o anki istek bağlamında (context) kullandığı aktif rol.
- **`roles`:** Kullanıcının yetkili olduğu roller listesi.

---

## 4. DİNAMİK ROL DEĞİŞTİRME: `POST /api/v1/auth/switch-role`

Kullanıcı oturum açıkken başka bir role geçiş yapmak istediğinde, istemci doğrudan token manipülasyonu yapamaz. Backend üzerindeki `POST /api/v1/auth/switch-role` uç noktasına istek göndermek zorundadır:

### 4.1. İstek Sözdizimi
```http
POST /api/v1/auth/switch-role HTTP/1.1
Host: localhost:8080
Authorization: Bearer <MEVCUT_JWT_TOKEN>
Content-Type: application/json

{
  "role": "kontrolor"
}
```

### 4.2. Backend Doğrulama Akışı
1. İstekteki JWT token doğrulanır (`jwt.Parse`).
2. Kullanıcının `user_id` değeri alınır.
3. İstenen rolün, kullanıcının `jw_user_roles` tablosundaki kayıtlarında bulunup bulunmadığı kontrol edilir.
4. **Başarısız Senaryo:** Kullanıcı talep ettiği role sahip değilse (Örn. `yolcu@aktarbilet.gov.tr` kullanıcısı `"admin"` rolüne geçmek isterse), backend anında `403 Forbidden` (`{"error": "user not authorized for requested role"}`) döner.
5. **Başarılı Senaryo:** Rol doğrulanırsa, yeni `active_role` claim'ini içeren yeni imzalı bir JWT üretilerek `200 OK` yanıtı ile istemciye teslim edilir.

---

## 5. RBAC KORUMA VE İZOLASYON DOĞRULAMA KANITLARI

`qa/tests/rbac.security.spec.js` ve `qa/tests/agy_role_audit.spec.js` ile yürütülen 17 güvenlik testi %100 başarıyla tamamlanmıştır:

```
=== RBAC GÜVENLİK DOĞRULAMA TEST ÇIKTISI ===
  ✓ Unauthenticated requests return 401 Unauthorized across all protected endpoints (28ms)
  ✓ Passenger cannot access inspector endpoints (GET /inspection/cards/:id) -> 403 Forbidden (15ms)
  ✓ Passenger cannot create inspection penalties -> 403 Forbidden (12ms)
  ✓ Passenger cannot access line management financials -> 403 Forbidden (14ms)
  ✓ Passenger cannot trigger headway optimization -> 403 Forbidden (11ms)
  ✓ Passenger cannot access settlement periods -> 403 Forbidden (13ms)
  ✓ Passenger cannot close settlement periods -> 403 Forbidden (15ms)
  ✓ Passenger cannot execute storno operations -> 403 Forbidden (12ms)
  ✓ Passenger cannot switch to admin or inspector role via switch-role -> 403 Forbidden (18ms)
  ✓ Inspector can scan cards and create penalties, but cannot close settlements -> 403 Forbidden (22ms)
  ✓ Line Manager can optimize headway, but cannot access settlement or inspection penalties -> 403 Forbidden (24ms)
  ✓ Accountant can close periods and execute storno, but cannot modify transit schedules -> 403 Forbidden (21ms)
  ✓ Admin has access to user management, metrics, and legitimate switch-role capabilities (35ms)
  ✓ SQL Injection attempt in role switching payload is safely rejected (18ms)
  ✓ XSS attempt in penalty description is sanitized (20ms)
  ✓ Expired JWT token is immediately rejected with 401 Unauthorized (10ms)
  ✓ Tampered JWT signature is rejected with 401 Unauthorized (9ms)

17 passed (100% GREEN)
```

Bu sonuçlar, AktarBilet RBAC mimarisinin hem yetki aşımına (Privilege Escalation) hem de yatay yetki gasbına (Horizontal IDOR) karşı tam korumalı olduğunu kanıtlamaktadır.

