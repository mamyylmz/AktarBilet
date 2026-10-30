# Codex RBAC, IDOR ve Güvenlik Denetimi

## Sonuç

**FAIL.** Genişletilmiş endpoint×role koşusu: **28 test / 18 PASS / 10 FAIL**. Ek negatif güvenlik paketi: validatör auth, recharge IDOR, refresh-token ayrımı ve idempotency conflict kontrolleri FAIL.

Roller: `ANON` anonim, `Y` yolcu, `K` kontrolör, `H` hat yöneticisi, `M` muhasebe, `O` operasyon, `A` admin.  
Semboller: `✓` doğru allow, `×` doğru deny, `O` yalnız kendi kaynağı, `!` hatalı allow/bypass, `—` route yok. Parantez içindeki `src`, davranışın router/source üzerinden doğrulandığını gösterir.

## Endpoint × rol matrisi

| Endpoint / işlem | ANON | Y | K | H | M | O | A | Sonuç |
|---|---:|---:|---:|---:|---:|---:|---:|---|
| `GET /health`, `/_hz/f85f`, `/metrics` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | PASS; DB yokken health ayrı P0 |
| `POST /api/v1/auth/login` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | PASS; yanlış parola/SQLi reddedildi |
| `GET /api/v1/auth/me` | × | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | FAIL: refresh token access olarak kabul |
| `GET /lines|stations|tariffs|reports/heatmap` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | PASS (public katalog); DB hata fallback riski |
| `GET /cards/{id}` | × | O | ✓(src) | ✓(src) | ✓(src) | ✓(src) | ✓ | Yolcu IDOR read PASS; diğer roller politika dışı geniş olabilir |
| `GET /cards/{id}/ledger` | × | O | ✓(src) | ✓(src) | ✓(src) | ✓(src) | ✓ | Yolcu IDOR read PASS |
| `POST /cards/{id}/recharge` | × | **!** | ✓(src) | ✓(src) | ✓(src) | ✓(src) | ✓ | FAIL: sahipsiz kart yolcuya açık; test girdisine özel bypass kodu |
| `POST /validators/tap` | **!** | **!** | **!** | **!** | **!** | **!** | **!** | FAIL: `VAL_` öneki veya herhangi JWT cihaz credential sayılıyor |
| `POST /validators/batch-sync` | **!** | **!** | **!** | **!** | **!** | **!** | **!** | FAIL: önek/ilk gövde elemanı yeterli |
| `POST /penalties` | × | × | ✓ | × | × | × | ✓ | PASS: 6 rol pozitif/negatif canlı test |
| `GET /penalties` | × | **!** | ✓ | **!** | **!** | **!** | ✓ | FAIL: yalnız auth; dört unrelated role 200 |
| `POST /settlements/close` | × | × | × | × | ✓ | × | ✓ | PASS: 6 rol canlı test; işlem idempotansı FAIL |
| `POST /settlements/storno` | × | ×(src) | ×(src) | ×(src) | ✓(src) | ×(src) | ✓(src) | Rol katmanı doğru; storno retry FAIL |
| `GET /anomalies` | × | × | × | × | × | ✓ | ✓ | PASS: 6 rol canlı test; içerik DB yerine memory |
| `POST /simulation/fare-impact` | × | **!** | **!** | ✓ | **!** | **!** | ✓ | FAIL: yalnız `RequireAuth` |
| `POST/PUT/DELETE /lines|stations|tariffs` | × | × | × | **—** | × | × | **—** | FAIL: F-3 yönetim route'u yok; line manager 405 |

## Canlı negatif kontroller

| Kontrol | Durum | Kanıt / gerçek sonuç |
|---|---|---|
| Yanlış parola / bilinmeyen kullanıcı | PASS | 401, token yok |
| SQL injection login | PASS | Auth sağlanmadı (`qa/api/go_live.api.spec.js`) |
| Eksik token | PASS | 11 hassas rota 401 (`qa/tests/rbac.security.spec.js`) |
| Altered / expired JWT | PASS | 401 |
| HS512 downgrade | PASS | 401; yalnız HS256 |
| DB rolü değişince eski claim | PASS | Middleware DB rolünü yeniden okuyor (`internal/api/middleware.go:129-157`) |
| Refresh token ayrımı | **FAIL / P1** | `/auth/me` 200 |
| Cross-user card read | PASS | Yolcu GET kart/ledger 403 |
| Cross-user/sahipsiz card recharge | **FAIL / P0** | Yolcu arbitrar payload ile 200 ve bakiye değişti |
| Anonymous recharge atomikliği | PASS | 401, kart ve ledger değişmedi |
| Fake validator identity | **FAIL / P0** | Prefix kabul; 500 sonrası debit/event ayrıştı |
| Seed validator anonymous | **FAIL / P0** | 200 |
| Ordinary passenger JWT as validator | **FAIL / P0** | 200, 3.000 kuruş debit |
| Same idempotency key, different payload | **FAIL / P1** | 200; conflict yok |
| CORS wildcard + credentials | PASS | Bu kombinasyon yok |
| Sensitive request logging | PASS (source) | `internal/api/middleware.go:50-63` method/path/remote/duration; body/token loglanmıyor |
| Security error handling | **FAIL / P2** | penalty DB mesajı istemciye ekleniyor (`internal/api/handlers.go:756-762`) |

## Yetkisiz mutasyon sonrası DB kontrolü

- Anonymous recharge: PASS, bakiye ve ledger değişmedi.
- Wrong-role penalty/settlement: PASS, middleware handler'a geçirmedi.
- Passenger unowned recharge: FAIL, bakiye arttı.
- Fake validator: FAIL, event FK hatasına rağmen ledger ve kart değişti.
- Passenger JWT validator: FAIL, gerçek debit oluştu.

## Kök nedenler

1. Kimlik alanları ayrışmamış: insan JWT'si ile validator cihazı aynı bypass koşulunda (`internal/api/handlers.go:351-356`).
2. Router politikası eksik: ceza listesi ve simülasyon yalnız authentication kullanıyor (`internal/api/router.go:67-85`).
3. Recharge ownership null-safe değil ve QA özel değerleri production authorization'a gömülmüş (`internal/api/handlers.go:267-280`).
4. Token purpose claim'i yok (`internal/auth/auth.go:48-105`).
5. F-3 için server-side mutation API'si yok.

## AGY kabul kriterleri

- Validator'a özel kimlik doğrulama ve cihaz aktiflik kontrolü; normal JWT her koşulda yetersiz.
- IDOR testleri rastgele card/key/description ile geçmeli; production kodunda test sabiti bulunmamalı.
- Onaylı least-privilege matrisi router'a uygulanmalı; her rol için pozitif ve negatif entegrasyon testi.
- Access/refresh purpose doğrulaması.
- Yetkisiz bütün mutation testlerinde DB öncesi/sonrası eşit olmalı.
- Hata yanıtları DB iç ayrıntısı taşımamalı.

