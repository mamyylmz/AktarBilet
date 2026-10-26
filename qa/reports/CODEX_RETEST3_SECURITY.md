# AktarBilet — Codex Retest 3 Security

**Tarih:** 26 Ekim 2026  
**Karar:** **FAIL — P0**  
**Ortam:** `aktarbilet-codex-retest3` adlı izole Docker Compose/PostgreSQL ortamı

Hiçbir production dosyası değiştirilmedi. Bu denetim için yalnız `qa/security/retest3_device_security.spec.js`, QA artifact'leri ve raporlar eklendi. Secret değerleri bu raporda özellikle redakte edilmiştir.

## Bulgular

### P0 — Depodaki master secret ile herhangi bir provisioned validator taklit edilebiliyor

- **Konum:** `.env.example:12`, `docker-compose.yml:32,75,94`, `qa/docker-compose.qa.yml:28,68`, `qa/helpers/http.js:4`, `cmd/simulator/main.go:254-257`, `cmd/verify/main.go:34-36`, `cmd/server/main.go:222`.
- **Kök neden:** Aynı deployable master secret birden fazla izlenen dosyada literal/default olarak bulunuyor. Sunucu cihaz anahtarlarını bu master secret ve bilinen validator ID'sinden deterministik HMAC ile türetip `jw_validator_credentials` içine yazıyor. Repo okuyucusu cihaz anahtarını çevrimdışı hesaplayabiliyor.
- **Reproduction:** `VAL_500T_01` için repo kaynaklı master secret'tan cihaz anahtarı türetildi; rastgele nonce/idempotency key ve doğru raw-body imzasıyla gerçek `/api/v1/validators/tap` çağrısı yapıldı.
- **Beklenen:** HTTP 401/403; bakiye, ledger, event, nonce ve sequence değişmemeli.
- **Gerçek:** HTTP 200; bakiye **10.000 → 7.000 kuruş**, ledger **1 → 2**, event **0 → 1**, nonce **0 → 1**; validator sequence ilerledi.
- **PostgreSQL etkisi:** Yetkisiz 3.000 kuruş debit ve kalıcı event oluştu.
- **Kanıt:** `qa/artifacts/codex/retest3/device-security-results.json`, `qa/artifacts/codex/retest3/security-summary.json`.
- **En küçük düzeltme:** Tüm repo/default master değerlerini kaldır; secret eksikken her ortamda non-zero exit et; tüm etkilenmiş anahtarları iptal et/döndür; cihaz başına rastgele, repo dışı provision edilen bağımsız secret kullan. Repo master'ından yeniden türetme veya otomatik credential oluşturma yapma.

### P0 — Eksik alanlı canonical-hash fallback imzalı gövde değişikliğini kabul ediyor

- **Konum:** `internal/api/handlers.go:449-455`.
- **Kök neden:** Raw body imzası başarısız olduğunda yalnız `card_id:line_id:station_id` alanlarının hash'i deneniyor. `direction`, `tapped_at`, `validator_id`, `sequence_no` ve diğer JSON alanları ikinci doğrulama yoluna bağlı değil.
- **Reproduction:** Geçerli cihaz anahtarıyla üç alanlı canonical değer imzalandı; gönderilen gövdede `direction` ve `tapped_at` değiştirildi.
- **Beklenen:** HTTP 401 ve sıfır state değişimi.
- **Gerçek:** HTTP 200; bakiye **10.000 → 7.000 kuruş**, yeni ledger/event/nonce oluştu ve sequence ilerledi.
- **PostgreSQL etkisi:** İmzalanmamış alanlarla kalıcı finansal mutation.
- **Kanıt:** `qa/artifacts/codex/retest3/device-security-results.json`, `qa/artifacts/codex/retest3/security-summary.json`.
- **En küçük düzeltme:** Eksik alanlı fallback'i kaldır. Tek bir açıkça sürümlenmiş canonical serialization veya exact raw body kullan; bütün iş alanlarını imzaya bağla ve tek doğrulama yap.

### P1 — Aynı sequence, farklı imzalı payload replay yerine başarılı sayılıyor

- **Konum:** `internal/sync/device_auth.go:142-151`, `internal/sync/service.go:111-129`.
- **Kök neden:** `sequence_no <= current` olduğunda aynı `(validator_id, sequence_no)` ile herhangi bir event bulunması doğrulamayı geçiriyor. Service de sequence eşleşen mevcut event'i tüm payload alanlarını karşılaştırmadan döndürüyor.
- **Reproduction:** İlk geçerli tap HTTP 200 işlendi. Aynı validator/sequence fakat farklı kart, direction, nonce ve idempotency key ile doğru imzalı ikinci istek gönderildi.
- **Beklenen:** HTTP 409 payload conflict/replay.
- **Gerçek:** HTTP 200. İkinci kartta finansal mutation olmadı, ancak farklı payload başarılı kabul edildi ve protokol doğru ACK veremedi.
- **PostgreSQL etkisi:** İkinci debit yok; yanlış başarılı yanıt nedeniyle audit/teslim semantiği bozuk.
- **Kanıt:** `qa/artifacts/codex/retest3/device-security-results.json`.
- **En küçük düzeltme:** Saklanan event için tüm güvenlik ve iş alanlarının canonical hash'ini tut; exact eşleşmede idempotent 200, herhangi bir farkta 409 dön.

### P1 — Validator'ın atanmadığı line/station üzerinde ücret yazılabiliyor

- **Konum:** `internal/api/handlers.go:486-497,521-530`.
- **Kök neden:** Handler yalnız validator'ın aktifliğini sorguluyor; `jw_validators.line_id/station_id` ile istek değerlerini karşılaştırmıyor.
- **Reproduction:** `VAL_500T_01`, başka hat/durak çiftiyle doğru cihaz anahtarı kullanılarak imzalandı.
- **Beklenen:** HTTP 403/422 ve sıfır state değişimi.
- **Gerçek:** HTTP 200; bakiye **10.000 → 7.000 kuruş**, ledger/event/nonce oluştu, sequence ilerledi.
- **PostgreSQL etkisi:** Fiziksel ataması yanlış validator adına kalıcı debit.
- **Kanıt:** `qa/artifacts/codex/retest3/device-security-results.json`.
- **En küçük düzeltme:** İmza doğrulamasından sonra, mutation öncesinde validator satırını kilitleyerek aktif `line_id` ve `station_id` eşleşmesini doğrula; batch'in her item'ında aynı kuralı uygula.

### P1 — Secret yokken fail-fast yapılmıyor

- **Konum:** `internal/config/config.go:23-31`, `cmd/server/main.go:40`.
- **Kök neden:** Config yalnız `production` değerinde panic ediyor; diğer ortamlarda izlenen literal QA secret'ına düşüyor. Sonraki uzunluk kontrolü bu default nedeniyle geçiyor.
- **Reproduction:** `VALIDATOR_HMAC_SECRET` boş override ile ayrı backend container'ı başlatıldı.
- **Beklenen:** Non-zero exit, HTTP listener açılmamalı.
- **Gerçek:** Container `running`, exit code 0; DB bağlantısı, seed ve port 8081 listener loglandı.
- **PostgreSQL etkisi:** Seed yeniden çalıştı; test sayımları değişmedi ancak öngörülebilir credential'lar aktif kaldı.
- **Kanıt:** `qa/artifacts/codex/retest3/bootstrap-runtime.json`.
- **En küçük düzeltme:** Environment ayrımı olmadan boş/zayıf secret'ı reddet; config/default/test helper/simulator/verify içindeki literal fallback'leri kaldır.

### P1 — Browser bundle demo kullanıcı parolasıyla sessiz login yapıyor

- **Konum:** `web/src/App.jsx:119-132,280-300,449-460`.
- **Kök neden:** Kart fetch/recharge ve muhasebe akışları açık metin demo credential ile `/auth/login` çağrısı yapıyor.
- **Reproduction:** Temiz browser state'te kullanıcı etkileşimi izlendi.
- **Beklenen:** Oturum yoksa login UI/hatası; istemci bundle'ında parola olmamalı.
- **Gerçek:** UAT-01 üç login çağrısı gördü; test FAIL.
- **PostgreSQL etkisi:** Bu vaka özelinde doğrulanmış finansal mutation yok; gömülü credential ile yetkisiz oturum edinme yüzeyi var.
- **Kanıt:** `qa/artifacts/codex/retest3/regression-102.json`.
- **En küçük düzeltme:** Tüm otomatik login ve parolaları frontend'den kaldır; kullanıcı tarafından kurulan session/JWT olmadan işlem başlatma.

## Başarılı güvenlik kontrolleri

- Önceki bağımsız auth/RBAC/IDOR/JWT paketi: **8/8 PASS**.
- Eski sızmış cihaz credential'ı: 401/403 ve sıfır finansal değişiklik.
- Anonymous tap ve batch: reddedildi.
- Passenger JWT ile validator ingest: reddedildi.
- Sabit QA idempotency key ve rastgele key ile başkasının kartına recharge: reddedildi, DB değişmedi.
- Batch signing principal ile event validator uyuşmazlığı: reddedildi.
- JWT Base64URL false-negative düzeltmesi: gerçek signature byte XOR edildi ve 401 alındı.
- Başka provisioned cihaz anahtarı: 401, bakiye/ledger/event/nonce/sequence değişmedi.
- Exact signed retry: iki HTTP 200, bir event ve bir debit.
- Revoked cihaz: 403; nonce ve sequence tüketilmedi.

## Eksik kalan güvenlik matrisi

P0 doğrulandıktan sonra kullanıcı yönlendirmesiyle yeni aktif exploit tekrarları durduruldu. Tekli ve batch için bütün kombinasyonların yeniden genişletilmiş koşusu yapılmadı: mixed-validator batch, sahte header varyasyonları, restart sonrası replay ve 429/500 item-level ACK matrisi **NOT_TESTED**. Bu eksikler FAIL kararını hafifletmez.

## Sonuç

Gate A **FAIL**. Yetkisiz finansal mutation iki ayrı yolla gerçek PostgreSQL'de kanıtlandı. Credential rotation, exact payload binding ve validator assignment enforcement tamamlanıp aynı negatif paket temiz DB'de tam PASS olmadan release veya güvenli yük testi yapılamaz.
