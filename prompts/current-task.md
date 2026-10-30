# AKTARBİLET — IMMEDIATE GITHUB PUSH, DOCKER RECOVERY, FINAL ACCEPTANCE & DELIVERY

## ROL

AktarBilet projesinin son geliştirme, altyapı onarımı, şartname doğrulaması ve GitHub tesliminden sorumlu kıdemli DevOps ve yazılım mühendisisin.

Proje dizini:

`/home/dilunsel/projects/AktarBilet`

Çalışma ortamı:
- Windows 11.
- WSL2 Ubuntu.
- Docker Desktop.
- Go backend.
- PostgreSQL 16.
- React/Vite frontend.
- GitHub Actions.

Öncelikli belgeler:

- `proje.md`
- `AGENTS.md`
- `IMPLEMENTATION_HANDOFF.md`
- `qa/reports/CODEX_FINAL_DELIVERY_ACCEPTANCE.md`
- `README.md`

## KESİN ÇALIŞMA SIRASI

1. **DERHAL İLK GITHUB PUSH.**
2. Docker Desktop / WSL arızasını gider.
3. AktarBilet backend, frontend ve PostgreSQL servislerini çalıştır.
4. Kalan `proje.md` kabul engellerini gider.
5. Playwright ve bütün zorunlu regresyonları doğrula.
6. README ve canary bütünlüğünü kontrol et.
7. Son commit/push işlemini gerçekleştir.
8. GitHub Actions başarısızlıklarını düzelt ve son CI sonucunu doğrula.

İlk push için Docker'ın çalışmasını bekleme.

---

# FAZ 1 — İLK PUSH ZORUNLU

Bu faz ilk gerçekleştirilecek işlemdir.

Önce yalnız hızlı güvenlik kontrolleri yap:

```bash
git status --short
git diff --stat
git branch --show-current
git remote -v
git diff --check
```

Kontrol et:
- GitHub remote ve branch doğru mu?
- `.env`, token, credential, private key veya kişisel gizli veri commit'e giriyor mu?
- Docker volume, QA veritabanı, build çıktısı veya gereksiz büyük dosyalar staging alanında mı?
- README'deki beş soru-cevap bölümü korunuyor mu?
- Yedi canary korunuyor mu?

Mevcut başarı kanıtları:

- Playwright: 157 PASS / 0 FAIL / 0 SKIP / 0 FLAKY.
- Go test/vet: PASS.
- Domain coverage: %77,5.
- Frontend lint/build: PASS.
- Canary: 7/7 PASS.
- Race ve güncel canlı F/T kabulü: henüz tamamlanmadı.

Bu nedenle mevcut sürümü `READY` değil, `NOT_READY` durumuyla kaydet.

İlgili dosyaları güvenli biçimde stage et, anlamlı commit oluştur ve normal push yap.

Kullanıcı bu görev için commit/push yetkisi vermiştir.

Kesin yasaklar:
- Force push.
- Hard reset.
- Kullanıcı değişikliklerini silme.
- Git geçmişini yeniden yazma.
- Gizli anahtarları commit etme.

İlk push başarılı olunca commit SHA, branch ve remote bilgilerini kaydet.

İlk push sonrasında CI başarısız olsa bile sonraki faza geç.

**İlk push yapılmadan Docker onarımına veya uzun testlere başlama.**

Push engellenirse güvenli şekilde root cause araştır. Remote geçmişi ezme. Çözülemiyorsa durumu raporla.

---

# FAZ 2 — DOCKER DESKTOP / WSL RECOVERY

## MEVCUT HATA

Docker Desktop şu mesajda takılıyor:

`Starting the Docker Engine...`

Ayrıca şu duyuruyu gösteriyor:

`Security update: eMudhra certificate compromise`

Docker Desktop güncellenemiyor ve AktarBilet servisleri çalışmıyor.

### Temel yaklaşım

eMudhra duyurusunun Docker Engine başlatma hatasının doğrudan nedeni olduğunu varsayma.

Önce gerçek arızayı izole et:

- Docker Desktop uygulaması.
- Docker Engine.
- Windows Docker servisi.
- WSL2.
- Docker context.
- Docker Desktop WSL entegrasyonu.
- Disk/bellek.
- Windows firewall/network.
- Güncelleme veya sertifika hatası.

### 2.1. Salt okunur teşhis

Önce:

```bash
docker version
docker info
docker context ls
docker compose version
```

kontrollerini yap.

Ardından Windows ortamında mümkünse:

```powershell
wsl --status
wsl --list --verbose
Get-Service com.docker.service -ErrorAction SilentlyContinue
```

çıktılarını incele.

PowerShell erişimini mevcut WSL ortamından sağlayabiliyorsan kullan; çalışmıyorsa kullanıcıya ilgili komutları ver.

Şunları ayır:

1. Docker CLI çalışıyor ama daemon erişilemiyor.
2. Docker Desktop servisi durdurulmuş.
3. Docker Engine WSL içinde başlatılamıyor.
4. WSL entegrasyonu kopmuş.
5. Docker disk veya dosya sistemi hatası var.
6. Güncelleme bağımsız biçimde başarısız.

Gereksiz servis kurma veya Docker yapılandırmasını değiştirme.

### 2.2. Güvenli kurtarma

Önce en az müdahaleli işlemleri dene.

Sıra:

1. Docker Desktop durumunu kontrol et.
2. Gerekliyse Docker Windows servisini normal şekilde yeniden başlat.
3. Docker Desktop'ı yeniden aç.
4. Docker Engine bağlantısını kontrol et.
5. WSL entegrasyonunu doğrula.
6. Güncelleme engeli varsa resmî Docker Desktop güncelleme yöntemini değerlendir.

Windows servisleri yönetici yetkisi gerektiriyorsa yetki yükseltmesini kendiliğinden aşmaya çalışma.

### KRİTİK WSL KURALI

Codex şu anda WSL Ubuntu içinde çalışıyor olabilir.

Bu durumda:

`wsl --shutdown`

komutunu kendi çalışan oturumunu öldürecek şekilde otomatik yürütme.

WSL shutdown gerekiyorsa:

- `IMPLEMENTATION_HANDOFF.md` dosyasına mevcut aşamayı kaydet.
- Kullanıcıya Windows PowerShell'de çalıştıracağı komutları bildir.
- Oturum yeniden başladıktan sonra handoff üzerinden devam et.

Onarım için Windows'un yeniden başlatılması gerekiyorsa da aynı kural geçerli.

### 2.3. Kesin veri koruma

Aşağıdakileri uygulama:

- Docker Desktop factory reset.
- Clean/Purge data.
- Docker volume silme.
- Global `docker system prune`.
- `docker volume prune`.
- `wsl --unregister`.
- PostgreSQL data directory silme.
- Kullanıcının diğer projelerindeki container/image/volume'leri kaldırma.
- Mevcut Docker Desktop'ı önce kaldırarak yeniden kurma.

Özellikle AktarBilet'in PostgreSQL verilerini koru.

Güncelleme gerekiyorsa resmî Docker Desktop kaynağını kullan.

TLS sertifika doğrulamasını devre dışı bırakma.

eMudhra güvenlik düzeltmelerini baypas etme.

### 2.4. Docker Engine doğrulaması

Onarım sonrasında:

```bash
docker info
docker compose version
```

başarılı çalışmalı.

Docker Engine'in erişilebilir olması ayrı bir PASS kriteridir.

---

# FAZ 3 — AKTARBİLET SERVİSLERİNİ AYAĞA KALDIR

Docker Engine çalıştıktan sonra:

1. AktarBilet dizinine geç.
2. Compose yapılandırmasını kontrol et.
3. Mevcut volume'leri ve container durumunu incele.
4. Güvenli migration/seed davranışını doğrula.
5. Yalnız AktarBilet servislerini başlat.

Başlangıç kontrolleri:

```bash
cd /home/dilunsel/projects/AktarBilet

docker compose config
docker compose ps
```

Ardından mevcut compose yapısına uygun şekilde servisleri başlat.

Gerekiyorsa:

```bash
docker compose up -d --build
```

Mevcut volume'leri silme.

Migration veya seed işlemi finansal verileri değiştirebilecekse önce mevcut davranışı incele.

Kontrol et:

- PostgreSQL 16 sağlıklı.
- Go API çalışıyor.
- Frontend çalışıyor.
- Health endpoint erişilebilir.
- `/_hz/f85f` 200 dönüyor.
- `X-DDO-Stamp: f85f26` korunuyor.
- Servisler restart loop'a girmiyor.

Başlatma başarısızsa yalnız ilgili container loglarını incele.

Herhangi bir uygulama hatasını Docker Engine arızasıyla karıştırma.

---

# FAZ 4 — PROJE.MD KABUL ENGELLERİ

Önceki raporda açık kalan sorunlar:

- Güncel race testi.
- `make verify`.
- Canlı F/T kabulü.
- Hat–durak sıralaması.
- Tarifelerin yürürlük tarihi ve doğruluğu.
- OpenAPI.
- k6.
- Chaos recovery.

Bunları `proje.md` zorunluluklarına göre doğrula.

## 4.1. Race

WSL'de gcc eksikliği, Docker'da I/O ve SIGBUS sorunları raporlandı.

Önce Docker arızasının race sorununa etkisini değerlendir.

İzole Go 1.23 ortamında:

`go test -race ./...`

çalıştır.

Gerçek bir data race varsa:

Reproduce → Root cause → Minimal fix → Retest.

## 4.2. make verify

WSL'de `make` yoksa uygun ortamı kullan.

Docker verifier alternatifi `proje.md` tarafından kabul ediliyorsa onu da değerlendirebilirsin.

QA backend ve PostgreSQL'i test boyunca erişilebilir tut.

Zorunlu doğrulamalar:

- F-1–F-7.
- T-1–T-5.
- Health.
- Authentication.
- Authorization.
- RBAC.

Sabit PASS çıktısı üretme.

15 dakika sınırını koru.

Başarısız testte non-zero exit zorunlu.

## 4.3. İzmir hatları

Korunması gereken mevcut veriler:

- 450 aktif İzmir hattı.
- 11.010 aktif durak.
- 40 pasif İstanbul hattı.
- Dört ayrı örnek hesap.
- Dört ayrı örnek kart.

Bu verileri yeniden üretme veya silme.

Asgari 40 doğrulanmış İzmir hattı şartını tekrar kontrol et.

Hat–durak sıralaması ve referans bütünlüğünü doğrula.

Resmî kaynaktan bilinmeyen sıra bilgisini uydurma.

## 4.4. Tarife

Gerçek İzmir tarifesiyle demo/simülasyon tarifesini ayır.

`proje.md` ücret motoru kuralları zorunlu önceliktir.

Geçmiş finansal kayıtları yeniden fiyatlandırma.

Gerçek tarifeye geçiş gerekiyorsa tarih bazlı sürümleme ve kategori eşleşmesini doğrula.

## 4.5. OpenAPI, k6, chaos

- OpenAPI'nin güncel endpoint'lerle eşleşmesi.
- Gerçek test sonucu.
- Hata oranı.
- p50/p95/p99.
- Persisted-event throughput.
- HMAC.
- Ledger reconciliation.
- Bağımlılık arızası sonrası recovery.

kontrollerini yap.

Gerçek kanıt yoksa PASS verme.

---

# FAZ 5 — REGRESYON

Önceki Playwright sonucu:

`157 PASS / 0 FAIL / 0 SKIP / 0 FLAKY`

Bu sonucu koru.

Yapılan düzeltmelerden sonra gerekli testleri çalıştır:

- Go test.
- Go vet.
- Go race.
- Domain coverage >= %70.
- Frontend lint/build.
- Playwright full suite.
- `make verify`.
- Gerçek PostgreSQL entegrasyonları.
- F/T acceptance.
- Seven canaries.
- Docker Compose health.
- `git diff --check`.

Her hata için:

1. Reproduce.
2. Root cause.
3. Minimal fix.
4. Targeted retest.
5. Related regression.

Testleri gevşetme, skip yapma veya production hatasını gizleme.

---

# FAZ 6 — README KORUMASI

README'deki kullanıcının cevapladığı beş doğrulama sorusu ve cevapları **birebir korunacak**.

Korunan bölüm:

`## 12. README’de Cevaplanacak Doğrulama Soruları`

Bu alanı:

- Yeniden yazma.
- Özetleme.
- Silme.
- Yeniden sıralama.
- Biçimlendirme.
- İçerisindeki prompt örneklerini değiştirme.

İlk push'taki içerikle birebir hash/diff kontrolü yap.

Diğer README bölümlerini yalnız doğrulanmış sonuçlarla güncelle.

Son satır:

`DDO Case Kodu: SEDIR-f85f26`

aynen korunacak.

---

# FAZ 7 — SON PUSH VE CI

Docker onarımı ve kalan geliştirmelerden sonra ikinci push gerçekleştir.

Son push öncesi:

- Git diff kontrolü.
- Secret kontrolü.
- README koruma kontrolü.
- Canary.
- Test sonuçları.
- Gerçek kabul durumu.

Yerel testler başarılıysa nihai değişiklikleri anlamlı commit'lere ayır.

Testlerin bir kısmı engellenmişse doğrulanmış düzeltmeleri ve dürüst raporları yine push et; READY iddiasında bulunma.

Normal push yap.

Force push yasak.

### GitHub Actions

Son push'un CI sonuçlarını kontrol et.

CI FAIL olursa:

1. Başarısız job/step'i belirle.
2. Logları incele.
3. Root cause bul.
4. Minimal fix uygula.
5. Yerel testi çalıştır.
6. Yeni commit oluştur.
7. Normal push gerçekleştir.
8. Yeni CI sonucunu kontrol et.

Zorunlu CI job'ları başarılı olmadan `GITHUB_CI: PASS` yazma.

---

# FAZ 8 — RAPOR VE HANDOFF

Güncelle:

- `qa/reports/CODEX_FINAL_DELIVERY_ACCEPTANCE.md`
- `IMPLEMENTATION_HANDOFF.md`

Kısa rapor:

### FIRST PUSH
- Commit SHA.
- Branch.
- Başarı durumu.

### DOCKER RECOVERY
- Root cause.
- Uygulanan çözüm.
- Docker Engine durumu.
- PostgreSQL durumu.
- Backend durumu.
- Frontend durumu.

### ACCEPTANCE
- F-1–F-7.
- T-1–T-5.
- Race.
- `make verify`.
- Playwright.
- Coverage.
- OpenAPI/k6/chaos.
- README koruması.

### FINAL PUSH
- Commit SHA.
- Branch.
- CI run URL.
- GitHub Actions sonucu.

### FINAL STATE
- `DOCKER_ENGINE: PASS/FAIL/BLOCKED`
- `APPLICATION_SERVICES: PASS/FAIL/BLOCKED`
- `PROJE_MD_ACCEPTANCE: PASS/FAIL/BLOCKED`
- `GITHUB_CI: PASS/FAIL/BLOCKED`
- `DELIVERY_STATUS: READY/NOT_READY`

Yalnız bütün zorunlu kriterler doğrulandığında READY ver.

## SON TALİMAT

**İLK ÖNCE GITHUB PUSH YAP.**

Docker'ın çalışmamasını ilk push'u ertelemek için gerekçe yapma.

İlk push tamamlanınca Docker Engine sorununu kullanıcı verilerini koruyarak çöz.

WSL shutdown gerekiyorsa kendi oturumunu sonlandırmadan kullanıcıya Windows PowerShell adımlarını ver.

Docker düzeldikten sonra AktarBilet servislerini ayağa kaldır.

Sonra `proje.md` kabul engellerini root cause → fix → retest yaklaşımıyla gider.

En son bütün güvenli değişiklikleri yeniden push et ve GitHub Actions sonuçlarını doğrula.

**Şimdi FAZ 1 ile başla.**