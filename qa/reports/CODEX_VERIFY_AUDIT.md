# Codex `make verify` Denetimi

## Sonuç

**FAIL / P0 — False verification success.** Verifier backend'e ulaşılamadığında başarısız olmak yerine in-memory moda geçiyor. Canlı backend'e ulaştığında da F-1..F-7 ve T-1..T-5 gerçek API/PostgreSQL yerine aynı in-memory doğrulayıcıdan alınıyor.

## Yeniden üretim

Host ortamında `make` yoktur; aynı Makefile resmi `golang:1.23` image'ında, kaynak read-only bağlanarak çalıştırıldı.

| Koşu | Komut özeti | Beklenen | Gerçek |
|---|---|---|---|
| Offline-negative | `docker run ... -e BACKEND_URL=http://127.0.0.1:1 golang:1.23 bash -lc 'make verify'` | non-zero, FAIL | **exit 0, 14/14 PASS** |
| Live-positive | aynı image, `--network host`, `BACKEND_URL=http://127.0.0.1:18080` | tüm maddeler gerçek API+DB | **exit 0, 14/14 PASS**, fakat yalnız SYS-1/SYS-2 canlı |

Tam offline komutu:

```sh
docker run --rm -v "$PWD":/src:ro -w /src \
  -e BACKEND_URL=http://127.0.0.1:1 golang:1.23 \
  bash -lc 'make verify'
```

## Kontrol akışı

- `Makefile:17-18`: hedef yalnız `go run ./cmd/verify`.
- `cmd/verify/main.go:48-62`: `/health` 200 değilse `runDirectVerifications()` çağrılır; bağlantı hatası failure değildir.
- `cmd/verify/main.go:69-90`: yalnız dönen item'lardan biri `Passed=false` ise exit 1; fallback item'ları geçtiği için offline durum exit 0 olur.
- `cmd/verify/main.go:94-104`: memory ledger/sync/settlement/detector ve nil DB handler kurulur.
- `cmd/verify/main.go:364-418`: live mod yalnız health/canary ve seed login yapar; kalan maddeleri `runDirectVerifications()` ile kopyalar.

## Madde bazlı gerçek veri kaynağı

| ID | Live koşuda gerçek endpoint / yanıt | PostgreSQL assertion | Gerçek F/T assertion | Audit sonucu |
|---|---|---|---|---|
| SYS-1 | `GET /health` 200; `GET /_hz/f85f` 200; stamp kontrolü yalnız `/health` | Yok | Gerçek HTTP | PASS, sınırlı |
| SYS-2 | `POST /api/v1/auth/login` 200 ve token non-empty | Yok | Gerçek HTTP | PASS, sınırlı |
| F-1 | Yok | Yok | Memory card 1000 + recharge 5000 = 6000 (`142-154`) | **Geçersiz PASS** |
| F-2 | Yok | Yok | Memory sync event status `PROCESSED` (`156-177`) | **Geçersiz PASS** |
| F-3 | In-memory router `GET /api/v1/lines` | Yok | Yalnız status 200; yönetim/CRUD yok (`179-191`) | **Geçersiz PASS** |
| F-4 | Yok | Yok | Pure pricing: tam/öğrenci/yaşlı üç eşitlik (`193-211`) | Unit-seviyesi, live kanıt değil |
| F-5 | Yok | Yok | Pure function, iki eşit 1000 kuruş pay (`213-227`) | Unit-seviyesi, live kanıt değil |
| F-6 | In-memory router POST penalty | Yok | Yalnız 201; nil pool nedeniyle persistence yok (`229-244`) | **Geçersiz PASS** |
| F-7 | In-memory router GET heatmap | Yok | Yalnız 200; hardcoded fallback (`246-258`) | **Geçersiz PASS** |
| T-1 | Yok | Yok | Pure pricing: bir transfer + ters yön; sınırlar/gece/limit/cap yok (`260-277`) | Eksik assertion |
| T-2 | Yok | Yok | Memory aynı request iki kez, ID eşit (`279-301`) | PostgreSQL/restart/batch yok |
| T-3 | Yok | Yok | Memory ledger; başlangıç bakiyesi ledger'a dahil değil (`303-323`) | PostgreSQL/append-only yok |
| T-4 | Yok | Yok | Memory settlement close + storno boolean (`325-342`) | DB/kuruş toplamı/idempotency yok |
| T-5 | Yok | Yok | Memory detector iki nokta (`344-359`) | DB anomaly/toplu veri yok |

## Anlamlı assertion değerlendirmesi

- Offline erişim hatası assertion'ı yok; ana negatif beklenti doğrudan ihlal ediliyor.
- F-3/F-7 yalnız HTTP 200'e bakıyor; veri kaynağı ve içerik doğrulanmıyor.
- F-6 yalnız 201'e bakıyor; DB satırı yok.
- T-1 90 dakika exact boundary, ikinci/üçüncü transfer, gece, öğrenci limit ve capping'i doğrulamıyor.
- T-2 farklı payload ile aynı key, `(validator_id, sequence_no)`, restart ve PostgreSQL etkisini doğrulamıyor.
- T-3 DB trigger, rollback, concurrent update ve `balance=SUM(ledger)` doğrulamıyor.
- T-4 pay toplamı, DB dönemi, duplicate close ve duplicate storno doğrulamıyor.
- T-5 `jw_anomalies` persistence doğrulamıyor; bağımsız canlı test bu alanın başarısız olduğunu kanıtladı.

## Yanlış PASS ile çelişen canlı bulgular

- F-1: yolcu keyfi sahipsiz karta dolum yapabiliyor.
- F-2/T-2: sahte validator kısmi debit bırakıyor; conflicting key 200; sıra deterministik değil.
- F-3: mutation route yok, line manager POST 405.
- F-6: UI başarısız istekte başarı gösteriyor; ceza listesi rolü aşırı geniş.
- T-3: live DB'de beş balance/ledger mismatch mevcut (test fixture kaynaklı olsa da verifier görmüyor).
- T-4: duplicate close/storno başarısız.
- T-5: API anomaliyi gösterirken DB satırı 0.

## AGY için minimal düzeltme kabul kriteri

1. Backend erişilemez/health DB-ready değilse verifier derhal non-zero.
2. Her F/T item gerçek endpoint'i çağırmalı ve benzersiz fixture üzerinde DB öncesi/sonrası assertion yapmalı.
3. Fixture/DB adı açıkça QA ile sınırlandırılmalı; cleanup transaction veya benzersiz run ID kullanmalı.
4. Her item endpoint, status, DB assertion ve hata nedenini yazmalı; 200 tek başına PASS olmamalı.
5. CI PostgreSQL service ile bu komutu çalıştırmalı; ayrıca offline-negative komutunun başarısız olduğunu test etmeli.

