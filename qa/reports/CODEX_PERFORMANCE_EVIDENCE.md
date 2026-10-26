# Codex Bağımsız Performans Kanıtı

**Tarih:** 26 Ekim 2026

**Ortam:** WSL2 Linux, 20 vCPU, 7,6 GiB RAM, 2 GiB swap; Docker Engine 28.3.3; PostgreSQL 16 Alpine; Go 1.23.6; Node 24.19.0.

**İzole proje:** `aktarbilet-codex-rc2`; API `127.0.0.1:18080`, PostgreSQL `127.0.0.1:15432`, frontend `127.0.0.1:13000`.
**Genel sonuç:** **FAIL (release capacity gate)**. Ham kalıcılık hedefi geçti; güvenli/geçerli finansal işleme kapasitesi ve 1.000.000 event kanıtı yok.

## 1. On dakikalık gerçek HTTP + PostgreSQL ingestion

Komut:

```bash
cd qa
QA_API_URL=http://127.0.0.1:18080 \
QA_SUSTAINED_SECONDS=600 QA_BATCHES_PER_SECOND=15 QA_BATCH_SIZE=20 \
QA_RUN_ID=20261026-final node load/sustained-ingest.mjs
```

| Ölçüm | Sonuç |
|---|---:|
| Süre (monotonic) | 599,999 sn |
| Batch / saniye | 15 |
| Batch boyutu | 20 event |
| Gönderilen | 180.000 |
| HTTP kabul edilen | 180.000 |
| PostgreSQL’de kalıcı benzersiz event | 180.000 |
| Eksik event | 0 |
| Tekil idempotency key | 180.000 |
| Tekil `(validator_id, sequence_no)` | 180.000 |
| HTTP batch hatası / beklenmeyen yanıt | 0 / 0 |
| İstek hızı | 15 batch request/sn |
| Kabul ve kalıcılık hızı | **300 event/sn** |
| Batch p50 / p95 / p99 | **209,46 / 1.071,17 / 1.279,17 ms** |
| Ledger etkisi | 180.000 |
| Duplicate ledger reference | 0 |
| Event statüsü | 180.000 `NEGATIVE_TOLERANCE` |

Kanıtlar: `qa/artifacts/codex/sustained-ingest-client.json`, `qa/artifacts/codex/sustained-ingest-db.json`.

Yük ortası kaynak anlık görüntüsü:

| Container | CPU | Bellek | Net I/O | Block I/O | PID |
|---|---:|---:|---:|---:|---:|
| backend | %60,15 | 16,91 MiB | 464/435 MB | 24,6 kB/0 B | 30 |
| postgres | %84,44 | 208,9 MiB | 373/427 MB | 1,34 MB/1,55 GB | 22 |

PostgreSQL anlık durumu: 13 active, 3 idle, 1 idle-in-transaction bağlantı; 0 bekleyen kilit. Koşu sonrasında 1 active ve 16 idle bağlantı görüldü.

### PERF-P1-01 — Ham throughput sonucu güvenli kapasite PASS’i değildir

- **Dosya/satır:** `internal/api/handlers.go:457-500`, `internal/sync/service.go:214-283`, `qa/load/sustained-ingest.mjs:1-108`.
- **Tekrar üretim:** Yukarıdaki 600 saniyelik komut ve ardından run-id bazlı DB sorguları.
- **Beklenen:** ≥278 geçerli, yetkili, finansal olarak nihai event/sn; sıfır kayıp/çift etki ve bütün invariant’lar.
- **Gerçek:** 300 kalıcı event/sn elde edildi, fakat endpoint anonim `VAL_` prefix’iyle kabul etti ve event’lerin tamamı `NEGATIVE_TOLERANCE`; audit 0. Bu nedenle gerçek validatör-auth ve nihai gelir pipeline kapasitesi ölçülmedi.
- **Kanıt:** İki sustained artifact ve `CODEX_SECURITY_FINDINGS.md#sec-p0-01`.
- **Minimal düzeltme:** Güvenli batch auth + atomik pipeline düzeltildikten sonra aktif seed cihaz anahtarıyla, işlenmiş/rejected ayrımını ve settlement etkisini ölçen testi tekrarla.

### PERF-P1-02 — 1.000.000 event testi NOT_TESTED

- **Dosya/satır:** `proje.md:118-126`, `qa/load/event-load.mjs:1-75`.
- **Tekrar üretim:** Mevcut repo koşusunda 1.000.000 event artifact’i bulunmadığını kontrol et: `find qa/artifacts -type f | rg 'million|1000000'`.
- **Beklenen:** Ortam destekliyorsa bir milyon benzersiz event; DB sayımı, kayıp, duplicate finansal etki ve latency kanıtı.
- **Gerçek:** 180.000 event/10 dakika koşuldu. Bir milyon event koşulmadı; AGY metrikleri bağımsız kanıt sayılmadı.
- **Kanıt:** `qa/artifacts/codex/sustained-ingest-client.json`.
- **Minimal düzeltme:** Önce P0 güvenlik/atomiklik hatalarını gider; sonra 1.000.000 benzersiz event’i kontrollü izole DB’de çalıştır ve event/ledger/audit/settlement invariant’larını raporla.

## 2. Eşzamanlı finans ve tutarlılık

`qa/db/final_financial.audit.spec.js` içindeki altı gerçek API+PostgreSQL testi geçti:

- Aynı kartta 20 paralel tap: bakiye ve tekil etki korundu.
- 15 tap + 15 recharge paralel akışı: ledger eşitliği korundu.
- 20 concurrent duplicate: bir event/bir debit.
- Yetersiz bakiye yarışında -5.000 kuruş tolerans altına inilmedi.
- Batch retry: her öğede bir event/bir finansal etki.
- Backend restart sonrası recharge retry: kalıcı ve idempotent.

Ancak genel Playwright koşusunda sırasız olay determinismi, settlement close ve storno retry başarısızdır. Bu nedenle eşzamanlılık alt kümesi PASS olsa da finansal performans kapısı FAIL’dir.

## 3. PostgreSQL kesintisi ve toparlanma

- PostgreSQL durdurulunca çalışan backend `/health` için 503, `status=DOWN` verdi.
- PostgreSQL tekrar başlatılınca backend 200/UP oldu.
- Koşunun 180.000 event’i kayıpsız kaldı.
- Kanıt: `qa/artifacts/codex/postgres-recovery.json`.

Not: Bu PASS, fresh-start sırasında DB yokken memory-mode/health fail-open hatasını düzeltmez.

## 4. Simülatör kuyruk ve backpressure

### PERF-P0-03 — Simülatör çalışma zamanında event teslim edemedi

- **Dosya/satır:** `cmd/simulator/main.go:85-90`, `cmd/simulator/main.go:94-113`, `cmd/simulator/main.go:142-194`, `cmd/simulator/main.go:199-249`.
- **Tekrar üretim:** `docker compose -p aktarbilet-codex-rc2 -f qa/docker-compose.qa.yml -f qa/docker-compose.final-audit.yml --profile simulator up -d --build simulator` ve `/status` + DB sayımı.
- **Beklenen:** Geçerli referanslarla gönderim, offline kalıcı kuyruk, restart sonrası flush, sıfır açıklanamayan kayıp.
- **Gerçek:** 50 event üretildi; 0 batch sent, 5 failed; DB event sayısı 66’dan 66’ya kaldı. Seed’de 4 validatör varken üretici 6 adet olmayan `VAL_SIM_*` dahil 10 ID seçiyor. Queue yalnız RAM channel; dolunca drop; shutdown drain/wait yok.
- **Kanıt:** `qa/artifacts/codex/simulator-runtime.json`.
- **Minimal düzeltme:** Seed-referans manifesti; disk/SQLite/WAL kuyruk; ack/result validation; restart replay; worker waitgroup ve drain; başarısız öğeleri retry/DLQ.

Deterministik iki manifest karşılaştırması yapılamadı; simülatör manifest üretmiyor. Queue restart testi yapısal olarak karşılanamaz. Sonuç **FAIL**.

## 5. Frontend performansı

Headless Chromium, 1440×900, gerçek frontend:

| Ölçüm | Sonuç |
|---|---:|
| HTTP | 200 |
| DOMContentLoaded | 327,1 ms |
| load event | 330,2 ms |
| network-idle wall time | 1.479 ms |
| Resource count | 5 |

`loadEvent <1s` hedefi geçti. Ancak Google Fonts’a 5 dış istek yapıldı; self-hosted font gereksinimi FAIL. Kanıt: `qa/artifacts/codex/frontend-navigation.json`.

## 6. Sonuç

- **Ham sürdürülebilir kalıcılık:** PASS, 300 event/sn × 10 dakika.
- **Güvenli validatör ingestion kapasitesi:** FAIL.
- **Nihai gelir/settlement kapasitesi:** NOT_TESTED.
- **1.000.000 event:** NOT_TESTED.
- **PostgreSQL interruption/recovery:** PASS.
- **Concurrent finance alt kümesi:** PASS; out-of-order/settlement invariant’ları FAIL.
- **Simulator pressure/restart queue:** FAIL.
- **Frontend temel yüklenme:** PASS; self-hosted kaynak kapısı FAIL.

