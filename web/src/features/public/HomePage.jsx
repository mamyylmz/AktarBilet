import React, { useState } from 'react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { formatKurus } from '../../lib/formatters';

export function HomePage({
  onOpenLogin,
  onOpenMap,
  onNavigatePortal,
  card,
  onRecharge,
  onTapTurnstile,
  rechargeLoading = false,
  heatmapError = false,
  onFetchHeatmap,
}) {
  const [openFaq, setOpenFaq] = useState(null);
  const [heatmapLoaded, setHeatmapLoaded] = useState(false);

  const toggleFaq = (index) => {
    setOpenFaq(openFaq === index ? null : index);
  };

  const faqs = [
    {
      q: 'AktarBilet kartımı nasıl doldurabilirim?',
      a: 'Online portal üzerinden veya İzmir genelindeki fiziksel kiosk noktalarından kredi kartı, banka kartı ya da nakit ile anında yükleme yapabilirsiniz. Yüklenen bakiye saniyeler içinde kartınıza yansır ve güvenli finansal defterde kuruş kuruş saklanır.',
    },
    {
      q: '90 Dakika Aktarma Sistemi nasıl işler?',
      a: 'İlk binişinizi gerçekleştirdikten sonra başlayan 90 dakikalık süre boyunca yapacağınız aktarmalar ücretsiz veya tarife tipinize göre indirimli olarak uygulanır. Validatörlerin çevrimdışı olması durumunda dahi aktarma hakkınız geriye dönük olarak garanti altındadır.',
    },
    {
      q: 'İnternet bağlantısı kesildiğinde veya turnike çevrimdışıyken kart geçerli midir?',
      a: 'Evet. AktarBilet validatörleri güvenli yerel çevrimdışı çalışma moduna (offline mode) sahiptir. Cihaz tekrar çevrimiçi olduğunda tüm binişler HMAC imzasıyla sunucuya senkronize edilir ve hak kaybı yaşanmaz.',
    },
    {
      q: 'Bakiye hareketlerimi ve biniş geçmişimi nerede görebilirim?',
      a: 'Sisteme giriş yaptıktan sonra "Yolcu Portalı" altında yer alan "Hesap ve Biniş Hareketleri" bölümünden tüm yükleme, aktarma ve biniş işlemlerinizi tarih, hat ve tutar detaylarıyla inceleyebilirsiniz.',
    },
  ];

  return (
    <div className="home-page" data-testid="public-home-page">
      {/* Hero Section */}
      <section className="hero-section">
        <div className="hero-content">
          <div className="hero-badge">İzmir Akıllı Ulaşım Altyapısı</div>
          <h1 className="hero-title">
            Şehir İçi Ulaşımda Güvenli, Hızlı ve Kesintisiz Biletleme
          </h1>
          <p className="hero-lead">
            AktarBilet; İzmir genelindeki tüm otobüs, metro, tramvay ve vapur hatlarında tek bir kartla seyahat etmenizi sağlar.
            90 dakika aktarma hakkı, kuruş hassasiyetli bakiye güvencesi ve canlı araç takibiyle toplu taşıma cebinizde.
          </p>
          <div className="hero-actions">
            <Button
              variant="primary"
              size="lg"
              onClick={onOpenLogin}
            >
              Hemen Başlayın
            </Button>
            <Button
              variant="outline"
              size="lg"
              onClick={onOpenMap}
            >
              Canlı Sefer Haritası
            </Button>
          </div>
        </div>
      </section>

      {/* Feature Highlights Grid */}
      <section className="features-section">
        <div className="features-grid">
          <div className="feature-card">
            <div className="feature-icon" aria-hidden="true">⏱️</div>
            <h3 className="feature-title">90 Dakika Aktarma Kuralı</h3>
            <p className="feature-desc">
              İlk binişinizden itibaren 90 dakika boyunca İzmir genelinde aktarma yapın. Kesintisiz yolculuğun tadını çıkarın.
            </p>
          </div>

          <div className="feature-card">
            <div className="feature-icon" aria-hidden="true">🔒</div>
            <h3 className="feature-title">Kuruş Hassasiyetli Defter</h3>
            <p className="feature-desc">
              Finansal hareketleriniz asla yuvarlanmaz. Append-only çift taraflı kayıt ile bakiyeniz kuruşu kuruşuna korunur.
            </p>
          </div>

          <div className="feature-card">
            <div className="feature-icon" aria-hidden="true">📡</div>
            <h3 className="feature-title">Çevrimdışı Güvenilirlik</h3>
            <p className="feature-desc">
              Tünel veya istasyonlarda ağ kopsa bile validatörler kesintisiz çalışır; bağlantı geldiğinde otomatik eşitlenir.
            </p>
          </div>

          <div className="feature-card">
            <div className="feature-icon" aria-hidden="true">🗺️</div>
            <h3 className="feature-title">Canlı ESHOT Araç Takibi</h3>
            <p className="feature-desc">
              Otobüslerin anlık konumlarını, sefer saatlerini ve durak doluluk oranlarını gerçek zamanlı haritada izleyin.
            </p>
          </div>
        </div>
      </section>

      {/* 90 Minutes Transfer Rule Deep-Dive */}
      <section className="transfer-rule-section mt-8">
        <Card
          title="90 Dakika Aktarma Sistemi Nasıl Çalışır?"
          subtitle="İzmir Büyükşehir Belediyesi toplu taşıma standartlarına uygun aktarma kuralı işleyişi"
        >
          <div className="transfer-steps-grid">
            <div className="transfer-step-box">
              <div className="step-num">1. Adım</div>
              <h4 className="step-title">İlk Biniş (Ana Giriş)</h4>
              <p className="step-desc">
                Otobüs, metro veya vapura ilk binişinizde tam tarife üzerinden ücretlendirme yapılır ve 90 dakikalık aktarma sayacı başlar.
              </p>
            </div>
            <div className="transfer-step-box">
              <div className="step-num">2. Adım</div>
              <h4 className="step-title">90 Dakikalık Aktarma Penceresi (En Fazla 2 Aktarma)</h4>
              <p className="step-desc">
                Sayaç sıfırlanana kadar en fazla 2 indirimli veya ücretsiz aktarma yapılabilir. 3. veya 4. aktarma seçeneği kesinlikle bulunmaz.
              </p>
            </div>
            <div className="transfer-step-box">
              <div className="step-num">3. Adım</div>
              <h4 className="step-title">Süre Bitimi ve Yeni Döngü</h4>
              <p className="step-desc">
                90 dakika dolduktan sonra yapılacak ilk biniş yeni bir seyahat döngüsü başlatır ve sayaç yeniden 90 dakikaya kurulur.
              </p>
            </div>
          </div>
        </Card>
      </section>

      {/* Tariffs Table */}
      <section className="tariffs-section mt-8">
        <Card
          title="Güncel Şehir İçi Ulaşım Tarifeleri (Simülasyon & Demo)"
          subtitle="İzmir geneli standart bilet, aktarma ve kontrol tarifeleri referans tablosu"
        >
          <div className="p-3 bg-surface-muted rounded-md mb-4 text-xs text-muted">
            <p>
              * Not: Tabloda yer alan tutarlar AktarBilet platformu test, denetim ve simülasyon referans tarifeleridir.
            </p>
          </div>
          <div className="table-responsive" tabIndex={0} role="region" aria-label="Tarife Bilgileri Tablosu">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">Tarife Türü</th>
                  <th scope="col" className="text-right">İlk Biniş Bedeli</th>
                  <th scope="col">90 Dk İçi Aktarma (Maks. 2 Kez)</th>
                  <th scope="col">Kullanım Kapsamı</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td className="font-semibold">Tam Kart</td>
                  <td className="text-right font-mono font-semibold">17,50 ₺</td>
                  <td><span className="badge badge-success">Ücretsiz (0,00 ₺)</span></td>
                  <td>Tüm ESHOT, Metro, İZBAN, Tramvay ve İZDENİZ</td>
                </tr>
                <tr>
                  <td className="font-semibold">Öğrenci Kartı</td>
                  <td className="text-right font-mono font-semibold">7,50 ₺</td>
                  <td><span className="badge badge-success">Ücretsiz (0,00 ₺)</span></td>
                  <td>Milli Eğitim ve YÖK onaylı öğrenci kartları</td>
                </tr>
                <tr>
                  <td className="font-semibold">Öğretmen & 60+ Yaş</td>
                  <td className="text-right font-mono font-semibold">11,50 ₺</td>
                  <td><span className="badge badge-success">Ücretsiz (0,00 ₺)</span></td>
                  <td>İlgili hak sahipliği onaylanmış kişiselleştirilmiş kartlar</td>
                </tr>
                <tr>
                  <td className="font-semibold">Saha Kontrol İhlal Cezası</td>
                  <td className="text-right font-mono font-semibold text-danger">200,00 ₺</td>
                  <td>-</td>
                  <td>Yetkisiz veya bilet okutulmaksızın yapılan binişler</td>
                </tr>
              </tbody>
            </table>
          </div>
        </Card>
      </section>

      {/* Demo Kiosk & Turnstile Simulator Section */}
      <section className="demo-kiosk-section mt-8" data-testid="demo-kiosk-section">
        <Card
          title="Demo Kiosk ve Turnike Test Alanı"
          subtitle="Kuruş hassasiyetli bakiye simülasyonu ve çevrimdışı validatör biniş testi"
        >
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 items-center">
            {/* Virtual Card */}
            <div className="transit-virtual-card">
              <div className="card-top-row flex justify-between items-center">
                <span className="card-chip-icon" aria-hidden="true" />
                <span className="card-type-badge font-mono uppercase">DEMO KART</span>
              </div>
              <div className="card-middle-row my-4">
                <span className="card-balance-label text-xs uppercase tracking-wider text-white/80">Güncel Bakiye</span>
                <div className="card-balance-value font-mono text-3xl font-bold text-white mt-1">
                  {formatKurus(card?.balance_kurus !== undefined ? card.balance_kurus : 50000)}
                </div>
              </div>
              <div className="card-bottom-row flex justify-between items-end text-xs text-white/90">
                <div>
                  <span className="block opacity-75">Kart Numarası</span>
                  <span className="font-mono font-semibold">{card?.id || 'CARD_YOLCU_001'}</span>
                </div>
                <div>
                  <span className="badge badge-success text-xs">Aktif</span>
                </div>
              </div>
            </div>

            {/* Kiosk Controls */}
            <div className="space-y-4">
              <div>
                <h4 className="font-semibold text-foreground mb-2">Kiosk Bakiye Yükleme</h4>
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="primary"
                    onClick={() => onRecharge && onRecharge(50)}
                    disabled={rechargeLoading}
                  >
                    +50 TL Yükle
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => onRecharge && onRecharge(100)}
                    disabled={rechargeLoading}
                  >
                    +100 TL Yükle
                  </Button>
                </div>
              </div>

              <div>
                <h4 className="font-semibold text-foreground mb-2">Turnike Biniş Simülasyonu</h4>
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="secondary"
                    onClick={() => onTapTurnstile && onTapTurnstile('LINE_500T', '500T')}
                  >
                    500T (Halkapınar - Alsancak)
                  </Button>
                  <Button
                    variant="secondary"
                    onClick={() => onTapTurnstile && onTapTurnstile('LINE_14ES', '14ES')}
                  >
                    14ES (Fahrettin Altay - Konak)
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </Card>
      </section>

      {/* Transit Heatmap Diagnostic Section */}
      <section className="heatmap-preview-section mt-8">
        <Card
          title="Hat ve Durak Doluluk Isı Haritası"
          subtitle="İzmir genelindeki anlık biniş yoğunluk telemetrisi"
        >
          <div className="space-y-4">
            <p className="text-sm text-muted">
              Turnikelerden gelen anlık biniş verilerine dayalı hat yoğunluk durumunu görüntüleyin:
            </p>
            <Button
              variant="outline"
              onClick={async () => {
                setHeatmapLoaded(true);
                if (onFetchHeatmap) await onFetchHeatmap();
              }}
            >
              Yoğunluk Raporunu Görüntüle
            </Button>

            {heatmapLoaded && heatmapError && (
              <div className="p-4 bg-danger/10 border border-danger/30 rounded-lg text-danger text-sm" role="alert">
                <p className="font-semibold mb-1">Sunucu bağlantı hatası oluştu</p>
                <p>Yoğunluk verisi alınamadı, lütfen yeniden deneyiniz.</p>
                <div className="mt-3">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onFetchHeatmap && onFetchHeatmap()}
                  >
                    Yeniden Dene
                  </Button>
                </div>
              </div>
            )}
          </div>
        </Card>
      </section>

      {/* FAQ Accordion */}
      <section className="faq-section mt-8">
        <Card
          title="Sıkça Sorulan Sorular"
          subtitle="AktarBilet kullanımı, kart işlemleri ve yolculuk kuralları hakkında merak edilenler"
        >
          <div className="faq-accordion">
            {faqs.map((faq, index) => {
              const isOpen = openFaq === index;
              return (
                <div key={index} className="faq-item">
                  <button
                    type="button"
                    className="faq-question-btn"
                    onClick={() => toggleFaq(index)}
                    aria-expanded={isOpen}
                    aria-controls={`faq-answer-${index}`}
                  >
                    <span className="faq-question-text">{faq.q}</span>
                    <span className="faq-icon" aria-hidden="true">{isOpen ? '−' : '+'}</span>
                  </button>
                  {isOpen && (
                    <div id={`faq-answer-${index}`} className="faq-answer">
                      <p>{faq.a}</p>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </Card>
      </section>

      {/* Informative Overview Box */}
      <section className="cta-box mt-8">
        <div className="cta-content text-center py-8">
          <h2 className="cta-title">Yolculuğunuza Hemen Başlayın</h2>
          <p className="cta-desc max-w-xl mx-auto text-muted">
            AktarBilet hesabınıza yukarıdaki "Giriş Yap" butonu ile erişerek kart bakiyenizi sorgulayabilir, bakiye yükleyebilir ve binişlerinizi takip edebilirsiniz.
          </p>
        </div>
      </section>
    </div>
  );
}
