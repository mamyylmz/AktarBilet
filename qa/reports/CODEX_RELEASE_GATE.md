# Codex Release Gate

## Nihai karar

**RELEASE DECISION: NOT_READY**  
Denetim zamanı: 2026-10-25 04:56 TRT  
Değerlendirilen kaynak: yerel `main@6ac85910e3cc8312b1759827b87342adc4d08b41` + uncommitted production değişiklikleri.

Bu karar CI veya Docker erişimsizliği nedeniyle BLOCKED değildir. Kritik yollar gerçek API ve izole PostgreSQL ile çalıştırıldı ve yeniden üretilebilir ürün hataları bulundu.

## Zorunlu gate'ler

| Gate | Durum | Release etkisi |
|---|---|---|
| Offline-negative `make verify` non-zero | **FAIL P0** | Tek komut kabul kanıtı güvenilmez |
| F-1..F-7/T-1..T-5 gerçek HTTP+DB doğrulama | **FAIL P0** | 12 madde in-memory |
| PostgreSQL zorunlu readiness | **FAIL P0** | DB yokken sahte healthy/in-memory |
| Finansal işlem atomikliği | **FAIL P0** | Event hatasında debit kalıyor |
| Validator device authentication | **FAIL P0** | Prefix/normal JWT ile mutation |
| Card ownership / IDOR | **FAIL P0** | Yolcu keyfi sahipsiz karta dolum |
| Arrival-order determinism | **FAIL P1** | Aynı olay seti farklı bakiye |
| Settlement close/storno idempotency | **FAIL P1** | Duplicate finansal kayıt/değişen close |
| T5 durable anomaly | **FAIL P1** | API true, DB 0 |
| RBAC least privilege | **FAIL P1** | 28 kontrolde 10 FAIL |
| Browser critical workflows | **FAIL P1** | Auth/role/storno/success durumu server-authoritative değil |
| Audit trail ve DB constraints | **FAIL P1** | Audit 0, invalid direction kabul |
| Ana Compose tek komut çalışma | **FAIL P1** | Current worktree frontend build olmuyor |
| Go unit/live tests | PASS | 58/58 yalnız live DB ile |
| Go race live DB | **FAIL** | Seed timeout; “zero races” kanıtlanmadı |
| Coverage ≥70 core | PASS | Toplam 82.6% |
| Seven canary | 6 PASS, 1 runtime BLOCKED | Container adı static doğru; main runtime build yüzünden yok |
| 1M valid event/hour sustained | **NOT_TESTED** | 20k kısa koşu 538.31 event/s; 1 saat yok |
| CI aynı local SHA/worktree | **FAIL** | CI yalnız eski remote SHA'yı test etti |

## AGY'nin tamamlaması gereken sıra

1. **Verifier'ı düzelt:** erişilemeyen backend hard FAIL; 14 maddenin tamamını gerçek endpoint + DB assertion'a geçir; CI'da offline-negative ekle.
2. **Finansal sınırı atomik yap:** validator doğrulama, event, ledger, kart/sayaç ve anomaly aynı transaction; her DB hatası rollback.
3. **Validator auth kur:** kayıtlı/aktif cihaz credential'ı; normal kullanıcı JWT'sini ve yalnız `VAL_` önekini reddet; simulator ID'lerini seed ile eşle.
4. **IDOR/test-tailoring'i kaldır:** `QA-IDOR-001`/`IDOR probe` koşullarını sil; her yolcu mutation'ında `card.user_id == user.id`.
5. **Fail-closed startup/readiness:** production/QA DB yoksa backend başlamasın; `/health` 503.
6. **Offline determinism:** kart olaylarını kronolojik replay/reprice et; farkı append-only iade/ek ücretle düzelt; iki arrival order aynı sonuç vermeli.
7. **Settlement/idempotency:** conditional close; storno ve diğer mutation'larda scoped key + payload hash; conflicting reuse 409/422.
8. **T5 persistence:** gerçek event FK ile anomaly'yi transaction içinde yaz; GET'i DB'den oku; insert hatasını yutma.
9. **RBAC/token:** onaylı endpoint×role matrisi; refresh/access purpose; ceza/simülasyon kısıtları; F-3 management API.
10. **UI'yi server-authoritative yap:** demo parolalarını bundle'dan çıkar; gerçek login/rol; tüm success durumlarını response+DB'ye bağla; storno gerçek API çağrısı.
11. **Şema/audit/para:** event direction CHECK, aynı transaction audit kayıtları, doğru `iade` tipi, parasal float alanlarını kaldır.
12. **Teslim zinciri:** `.dockerignore` + deterministic frontend install; ana Compose runtime smoke; OpenAPI; PostgreSQL'li CI ve coverage/race/security/browser gate.
13. **Test izolasyonu:** live Go fixture'larını transaction/cleanup ile izole et; başlangıç bakiyesini ledger ile kur; race deadline'ını düzelt.
14. **Yeniden kanıt:** P0/P1 regresyon paketi 0 FAIL, ana Compose PASS, live `make verify` gerçek DB, ardından 1 saat/1M valid event testi.

## Yeniden denetim için minimum komut seti

```sh
# Offline verifier mutlaka non-zero olmalı
BACKEND_URL=http://127.0.0.1:1 make verify

# İzole PostgreSQL ile bütün Go yolları; skip kabul edilmez
DATABASE_URL='<isolated-qa-url>' go test -count=1 ./internal/...
DATABASE_URL='<isolated-qa-url>' go test -race -count=1 ./internal/...
go vet ./...

# Gerçek stack ve bağımsız QA
docker compose up --build --detach --wait
cd qa && npm run test:all

# Düzeltilmiş P0/P1 hedefleri
npx playwright test api/go_live.api.spec.js db/go_live.db.spec.js \
  db/final_financial.audit.spec.js db/final_anomaly.audit.spec.js \
  security/go_live.security.spec.js security/final_rbac.audit.spec.js \
  e2e/go_live.uat.spec.js
```

## Onay koşulu

Release ancak tüm P0/P1 maddeleri kapatılıp gerçek PostgreSQL üzerinde yeniden PASS olduğunda, ana Compose current worktree'de ayağa kalktığında, live race suite non-zero olmadan tamamlandığında ve CI aynı commit SHA'yı aynı gate'lerle test ettiğinde **READY** olabilir. Önceki raporların PASS etiketleri bu koşulların yerine geçmez.

