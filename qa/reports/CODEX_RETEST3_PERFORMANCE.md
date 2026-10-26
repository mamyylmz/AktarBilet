# AktarBilet — Codex Retest 3 Performance

**Tarih:** 26 Ekim 2026  
**Karar:** **NOT_TESTED / BLOCKED BY RELEASE POLICY**

## Güvenli yük kapısı

600 saniyelik sustained load ve 1.000.000 benzersiz event koşuları başlatılmadı. Gate A'da repo credential'ıyla HTTP 200 ve gerçek debit üreten P0 açık kanıtlandı; ayrıca Gate C, D ve F başarısız. Retest 3 talimatı, A–F zorunlu kapıları PASS olmadan güvenli load/stress koşusuna geçilmesini açıkça yasaklıyor. Kullanıcının sonraki yönlendirmesi uyarınca bilinen credential'ı yeniden kullanan güvenliksiz throughput akışı da atlandı.

Önceki raporlardaki throughput değerleri bağımsız Retest 3 ölçümü olarak yeniden kullanılmadı.

| Metrik | Retest 3 değeri | Durum |
|---|---:|---|
| Sustained süre | N/A | NOT_TESTED |
| HTTP requests/sec | N/A | NOT_TESTED |
| Persisted events/sec | N/A | NOT_TESTED |
| p50 | N/A | NOT_TESTED |
| p95 | N/A | NOT_TESTED |
| p99 | N/A | NOT_TESTED |
| Success/reject/error rate | N/A | NOT_TESTED |
| CPU/RAM | N/A | NOT_TESTED |
| DB pool/lock basıncı | N/A | NOT_TESTED |
| 1M persisted event | N/A | NOT_TESTED |

## Simulator recovery ölçümü

Bu çalışma performans kanıtı değil, Section 8 fonksiyonel dayanıklılık testidir.

- Seed: **200.005 kart**, **40 hat**, seed `42`.
- Sağlıklı başlangıç snapshot'ı: `180 generated`, `18 batches sent`.
- Backend offline/restart öncesi: `1.000 generated`, `27 sent`, `18 failed`, queue `52`.
- PostgreSQL: `270 persisted`, `270 distinct validator+sequence`, `270 distinct idempotency`, `270 ledger effect`.
- WAL: oluşturulamadı; `/var/lib/aktarbilet-simulator` root sahipli, `appuser` write sırasında `permission denied` aldı.
- Process restart: sayaçlar `0`, queue `0`, replay `0`, volume boş.
- Container recreation + reconnect snapshot'ı: `120 generated`, `12 sent`, fakat PostgreSQL hâlâ `270 persisted`.
- **Açıklanamayan kayıp: 730 event.**
- Duplicate validator+sequence: `0`.
- Global ledger reconciliation mismatch: `0`.

`pending WAL=0` başarı değildir; WAL hiç oluşturulamadığı için pending kayıt kalmamıştır.

Kanıt: `qa/artifacts/codex/retest3/simulator-runtime.json`.

## ESHOT gecikme ve retry ölçümü

Bu değerler kapasite/load sonucu değildir:

| Kontrol | Sonuç |
|---|---:|
| Hat 800 canlı istek | 200, 14 araç, 293,352 ms |
| Hat 470 canlı istek | 200, 5 araç, 1.083,205 ms |
| Hat 35 canlı istek | 200, 8 araç, 1.125,234 ms |
| Hat 800 cache hit | 200, 1,289 ms |
| `Retry-After: 1` | PASS, 1,00 sn |
| `Retry-After: HTTP-date` | PASS, 1,93 sn |
| Uzun Retry-After + context bound | PASS, 0,15 sn |

Backoff 30 saniye ile sınırlandırılmıştır; kaynakta jitter yoktur. Kanıt: `qa/artifacts/codex/retest3/eshot-runtime.json`, `qa/go/transit_rate_limit.audit_test.go`.

## Yük testini açmak için kesin ön koşullar

1. Repo credential P0 ve eksik alanlı signature fallback kapatılmalı; bağımsız güvenlik paketi tam PASS olmalı.
2. Sırasız event finansal determinismi 102/102 içinde PASS olmalı.
3. Simulator volume ownership, WAL/sequence persistence ve recovery loss=0 olmalı.
4. Gerçek MapLibre/PMTiles/GTFS, lint, OpenAPI ve WCAG kapıları PASS olmalı.
5. Bundan sonra ayrı temiz DB'lerde önce 600 saniye, sonra 1M koşusu; HTTP/DB/ledger/duplicate/loss ve CPU/RAM birlikte ölçülmeli.

## Sonuç

Gerçek throughput ve p50/p95/p99 **ölçülmedi**. Bunun nedeni kapasite yetersizliği değil, zorunlu güvenlik/fonksiyon kapılarının FAIL olmasıdır; sonuç PASS değildir.
