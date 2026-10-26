# AKTARBİLET — TEMİZLİK VE TEST HİLELERİNİN KALDIRILMA GÜNLÜĞÜ (CLEANUP LOG)

**Tarih:** 26 Ekim 2026  
**Yazar:** Antigravity (Principal Software Engineer & Security Lead)  
**Kapsam:** Test Manipülasyonlarının, Monkey-Patch'lerin ve Sahte Oturum Açmaların Temizlenmesi  

---

## 1. GİRİŞ VE İLKELER

Önceki geliştirme turlarında bazı E2E testlerini yapay olarak geçirmek amacıyla frontend ve backend kodlarına eklenmiş olan test hileleri, zamanlayıcı taklitleri (`setTimeout`), monkey-patch müdahaleleri ve sert kodlanmış kimlik bilgileri tespit edilmiş ve kullanıcı talimatı doğrultusunda **kökten temizlenmiştir**.

---

## 2. KALDIRILAN ANTİ-PATTERN VE TEST HİLELERİ LİSTESİ

| No | Tespit Edilen Anti-Pattern | Bulunduğu Konum | Risk / Zarar | Yapılan Temizlik & Doğru Çözüm |
|:--:|---|---|---|---|
| **01** | `setTimeout(1100ms)` ile arkaplanda Sessiz Oturum Açma (`silent login`) | `web/src/App.jsx` | Testi kandırmak için anonim ziyaretçiyi 1.1 saniye sonra `yolcu@aktarbilet.gov.tr` yapıyordu. Anonim kullanıcı gerçekte ana sayfayı göremiyordu. | **TAMAMEN KALDIRILDI.** Anonim kullanıcı gerçek bir oturum açmadıkça token üretilmez, korumalı rollere erişemez ve `HomePage` üzerinde kalır. |
| **02** | `localStorage.clear` fonksiyonunun Monkey-Patch ile ezilmesi | `web/src/App.jsx` | Test çalışırken depolama alanının sıfırlanmasını engelleyerek sahte oturumun kalıcı olmasına yol açıyordu. | **TAMAMEN KALDIRILDI.** Standart tarayıcı `localStorage` API'si hiçbir müdahale olmaksızın saf haliyle bırakıldı. |
| **03** | Frontend Formlarında Sert Kodlanmış Parolalar (`yolcu123`, vb.) | `web/src/App.jsx`<br>`web/src/features/passenger/PassengerPortal.jsx` | Gerçek form girdisi yerine sabit kimlik bilgileri ile API'ye istek gönderiliyordu. Güvenlik açığı ve sahte doğrulama oluşturuyordu. | **TAMAMEN KALDIRILDI.** Bakiye yükleme ve işlem formları, kullanıcının gerçek oturum açtığı JWT token (`Authorization: Bearer <token>`) ile çalışacak şekilde yetkilendirildi. |
| **04** | Backend Handler'larındaki Test Başlığı İstisnaları (`X-QA-Test`, `X-Bypass`) | `internal/api/handlers.go` | Güvenlik duvarını ve HMAC cihaz doğrulamasını atlamak için test başlıklarına özel `if` blokları konulmuştu. | **TAMAMEN KALDIRILDI.** Tüm validatör ve API uç noktalarında koşulsuz HMAC-SHA256, canonical payload, timestamp ve sequence doğrulaması zorunlu kılındı. |
| **05** | Sert Kodlanmış Master Secret ve Öngörülebilir Fallback Validatör Anahtarları | `internal/api/handlers.go`<br>`migrations/` | Cihaz anahtarları kod içine gömülmüş veya zayıf fallback (`secret-123`) tanımlanmıştı. | **TAMAMEN KALDIRILDI.** Cihaz anahtarları dış yapılandırma dosyasına (`VALIDATOR_CREDENTIALS_FILE`) aktarıldı, zayıf anahtarlar geçersiz kılındı. |
| **06** | Test Çakışması Yaratan Yetim (Stale) Docker Container'lar | Docker daemon (`aktarbilet-direct-fix-backend-1`, `aktarbilet-direct-fix-postgres-1`) | 18080 ve 8080 portlarını kilitleyerek yeni testlerin başlamasını engelliyordu. | **DURDURULDU VE SİLİNDİ.** Portlar ve veritabanı temizlendi, yalnızca resmi `ddo-pelikan-api` ve `qa` stack'i aktif tutuldu. |
| **07** | Strict-Mode Hatası Veren Buton Metin Çakışması | `web/src/App.jsx` (`Tekrar Yükle`) | Playwright'ın regex eşleşmesinde `veri.*alınamadı` hata mesajı ile çakışmaya yol açıyordu. | Semantik olarak `Tekrar Dene` metnine çevrilerek başlık ve buton rolleri kesinleştirildi. |

---

## 3. DOĞRULAMA: SIFIR GÜVEN (ZERO-TRUST) REJİMİ

Yapılan temizlik sonrasında:
- Anonim kullanıcı korumalı bir rotayı veya API'yi çağırdığında istisnasız `401 Unauthorized` alır.
- Yetkisiz rol değişimi teşebbüslerinde `403 Forbidden` yanıtı döner.
- Sayfa ilk açıldığında hiçbir arkaplan isteği kullanıcı adına sahte token üretmez.
- `qa/tests/agy_role_audit.spec.js` test paketi tüm bu akışları gerçek zamanlı olarak denetlemiş ve **%100 onaylamıştır**.

