import React from 'react';
import { Modal } from '../../components/ui/Modal.jsx';

export function LegalModal({
  topic, // 'about' | 'privacy' | 'kvkk' | 'terms' | 'contact' | 'how-it-works' | 'help'
  isOpen,
  onClose,
}) {
  const getTitle = () => {
    switch (topic) {
      case 'about':
        return 'Hakkımızda — AktarBilet Sistemi';
      case 'privacy':
        return 'Gizlilik ve Veri Güvenliği Politikası';
      case 'kvkk':
        return 'KVKK Aydınlatma Metni';
      case 'terms':
        return 'Kullanım Koşulları ve Hizmet Şartları';
      case 'contact':
        return 'İletişim ve Operasyonel Destek';
      case 'how-it-works':
        return 'AktarBilet Nasıl Çalışır?';
      case 'help':
        return 'Yardım ve Sıkça Sorulan Sorular (SSS)';
      default:
        return 'Kurumsal Bilgilendirme';
    }
  };

  const renderContent = () => {
    switch (topic) {
      case 'about':
        return (
          <div className="space-y-4 text-sm text-foreground">
            <p>
              <strong>AktarBilet</strong>, modern büyükşehir toplu taşıma ihtiyaçları doğrultusunda geliştirilmiş; kuruş hassasiyetli finansal defter (append-only double-entry ledger), 90 dakikalık entegre aktarma mekanizması, çevrimdışı validatör senkronizasyonu ve yüksek eşzamanlılık desteği sunan yeni nesil akıllı biletleme ve filo yönetim platformudur.
            </p>
            <p>
              Sistemimiz; yolcu konforu, saha denetim operasyonları, hat frekans optimizasyonları ve şeffaf işletmeci gelir mutabakatını tek bir çatı altında birleştirmektedir.
            </p>
            <div className="p-3 bg-surface-muted rounded-lg border border-border text-xs text-muted space-y-1">
              <p><strong>Önemli Bilgilendirme ve Açık Veri Atfı:</strong></p>
              <p>
                AktarBilet bağımsız bir mühendislik ve simülasyon platformudur. Canlı araç ve sefer verileri, İzmir Büyükşehir Belediyesi ESHOT Genel Müdürlüğü Açık Veri Portalı entegrasyonu ile sağlanmaktadır. Sistemde sunulan tarifeler test ve simülasyon amaçlı referans değerlerdir.
              </p>
            </div>
          </div>
        );

      case 'how-it-works':
        return (
          <div className="space-y-4 text-sm text-foreground">
            <div>
              <h4 className="font-semibold text-base mb-1">1. Biniş ve 90 Dakika Aktarma Kuralı (En Fazla 2 Aktarma)</h4>
              <p>
                Otobüs, metro veya vapura ilk biniş yapıldığında sistem standart tarifeyi uygular ve 90 dakikalık geri sayım sayacını başlatır. Sayaç süresince farklı hatlarda en fazla 2 aktarma (1. aktarma %50, 2. aktarma %75 indirimle veya ücretsiz) yapılabilir. Üçüncü, dördüncü veya daha fazla aktarma seçeneği kesinlikle bulunmamakta olup, müteakip binişler tam tarife üzerinden ücretlendirilir.
              </p>
            </div>
            <div>
              <h4 className="font-semibold text-base mb-1">2. Kuruş Hassasiyetli Defter (Zero Float)</h4>
              <p>
                Tüm finansal bakiye ve ücret hareketleri tam sayı (integer kuruş) cinsinden kaydedilir. Hiçbir kayıtta ondalıklı kayan nokta (float) yuvarlama hatası veya tutar kaybı oluşmaz.
              </p>
            </div>
            <div>
              <h4 className="font-semibold text-base mb-1">3. Çevrimdışı Validatör Senkronizasyonu</h4>
              <p>
                Araçlardaki validatörler tünel veya baz istasyonu kopukluklarında binişleri yerel bellekte güvenle saklar. Ağ bağlantısı kurulduğunda monotonik sıra numarası ve HMAC-SHA256 imzası ile sunucuya sırayla aktarılır.
              </p>
            </div>
            <div>
              <h4 className="font-semibold text-base mb-1">4. Çoklu Rol ve Yetkilendirme (RBAC)</h4>
              <p>
                Platform; Yolcu, Kontrolör, Hat Yöneticisi, Muhasebe, Operasyon ve Sistem Yöneticisi rollerini ayrık yetki seviyeleriyle destekler.
              </p>
            </div>
          </div>
        );

      case 'help':
        return (
          <div className="space-y-4 text-sm text-foreground">
            <div>
              <h4 className="font-semibold text-sm mb-1">Kart Bakiyemi Nasıl Yükleyebilirim?</h4>
              <p className="text-muted">
                Yolcu Portalına giriş yaptıktan sonra demo kiosk alanından +50 TL, +100 TL veya +200 TL tutarlarını seçerek bakiyenizi anında artırabilirsiniz.
              </p>
            </div>
            <div>
              <h4 className="font-semibold text-sm mb-1">Aktarma Sürem Dolduğunda Ne Olur?</h4>
              <p className="text-muted">
                90 dakikalık aktarma penceresi dolduğunda yapılacak ilk biniş yeni bir seyahat döngüsü başlatır ve ilk biniş tarifesi üzerinden ücretlendirilir.
              </p>
            </div>
            <div>
              <h4 className="font-semibold text-sm mb-1">Geçmiş Binişlerimi Nereden Görebilirim?</h4>
              <p className="text-muted">
                Yolcu Portalı içerisindeki "Hesap ve Biniş Hareketleri" tablosundan tarih, hat, tutar ve bakiye detaylarıyla tüm geçmişinizi inceleyebilirsiniz.
              </p>
            </div>
            <div>
              <h4 className="font-semibold text-sm mb-1">Saha Kontrol Denetimi Nasıl Yapılır?</h4>
              <p className="text-muted">
                Saha kontrolörleri el terminalleri üzerinden kart numaranızı sorgulayarak son turnike geçişinizin geçerlilik durumunu ve ceza geçmişini görüntüleyebilir.
              </p>
            </div>
          </div>
        );

      case 'privacy':
        return (
          <div className="space-y-3 text-sm text-foreground">
            <p>
              AktarBilet platformu, kullanıcılarının bilgi güvenliğini en üst seviyede tutmayı taahhüt eder.
            </p>
            <h4 className="font-semibold text-sm">İşlenen Veri Kategorileri</h4>
            <ul className="list-disc pl-5 space-y-1 text-muted">
              <li>Kullanıcı kimlik ve oturum verileri (E-posta, kullanıcı adı, JWT oturum bilgisi).</li>
              <li>Seyahat ve biniş telemetrisi (Kart ID, turnike validatör kodu, biniş tarihi/saati, hat ve durak bilgisi).</li>
              <li>Finansal defter hareketleri (Yükleme tutarları, çekimler, aktarma indirimleri ve kuruş bakiye).</li>
            </ul>
            <p className="text-muted mt-2">
              Tüm oturum anahtarları, cihaz kimlikleri ve şifreleme parametreleri endüstri standardı HMAC-SHA256 ve bcrypt algoritmalarıyla korunur. Veriler hiçbir ticari üçüncü tarafa devredilmez.
            </p>
          </div>
        );

      case 'kvkk':
        return (
          <div className="space-y-3 text-sm text-foreground">
            <p>
              6698 sayılı Kişisel Verilerin Korunması Kanunu ("KVKK") uyarınca, veri sorumlusu sıfatıyla AktarBilet tarafından işlenen kişisel verileriniz hakkında bilgilendirme aşağıda özetlenmiştir:
            </p>
            <h4 className="font-semibold text-sm">1. Veri İşleme Amaçları ve Hukuki Sebepler</h4>
            <p className="text-muted">
              Kişisel verileriniz; toplu taşıma biletleme hizmetlerinin ifası, 90 dakika aktarma kurallarının uygulanması, usulsüz kullanımın önlenmesi, kamu bütçesi ve işletmeci mutabakatlarının denetlenmesi hukuki yükümlülükleri kapsamında işlenmektedir.
            </p>
            <h4 className="font-semibold text-sm">2. İlgili Kişi Hakları (KVKK Md. 11)</h4>
            <p className="text-muted">
              Veri sahipleri; verilerinin işlenip işlenmediğini öğrenme, yanlış verilerin düzeltilmesini talep etme ve kanuni sınırlar çerçevesinde silinmesini isteme hakkına sahiptir. Başvurular destek kanallarımız üzerinden iletilebilir.
            </p>
          </div>
        );

      case 'terms':
        return (
          <div className="space-y-3 text-sm text-foreground">
            <p>
              AktarBilet sistemini kullanan tüm yolcular, saha personeli ve işletmeciler aşağıdaki kullanım şartlarını kabul etmiş sayılır:
            </p>
            <ul className="list-disc pl-5 space-y-1 text-muted">
              <li>Kartlar kişiseldir; indirimli veya serbest kartların hak sahibi olmayan kişilerce kullanımı durumunda idari para cezası uygulanır.</li>
              <li>Turnike okutma simülasyonları ve bakiye yüklemeleri demo altyapısında gerçek zamanlı işlenir; gerçek bankacılık ödeme işlemi yerine geçmez.</li>
              <li>Ağ kopukluğu veya cihaz arızalarında validatörün kaydettiği biniş verisi esastır; sunucuya senkronize edildiğinde geriye dönük hak kaybı oluşmaz.</li>
            </ul>
          </div>
        );

      case 'contact':
        return (
          <div className="space-y-4 text-sm text-foreground">
            <p>
              AktarBilet platformu, şehir içi ulaşım operasyonları ve teknik destek konularında aşağıdaki resmi iletişim kanallarımızdan bize ulaşabilirsiniz:
            </p>
            <div className="p-4 bg-surface-muted rounded-lg border border-border space-y-2">
              <div>
                <span className="font-semibold block text-xs uppercase tracking-wider text-muted">Operasyon Merkezi</span>
                <span className="text-foreground">AktarBilet Şehir İçi Ulaşım Bilgi İşlem Merkezi</span>
              </div>
              <div>
                <span className="font-semibold block text-xs uppercase tracking-wider text-muted">E-Posta Desteği</span>
                <span className="font-mono text-primary">destek@aktarbilet.gov.tr</span>
              </div>
              <div>
                <span className="font-semibold block text-xs uppercase tracking-wider text-muted">Hizmet Saatleri</span>
                <span className="text-muted">7 Gün 24 Saat Kesintisiz Saha ve Sistem Desteği</span>
              </div>
            </div>
            <p className="text-xs text-muted">
              Acil durak arızaları ve turnike ihlalleri için saha kontrolörleri doğrudan merkez operasyon konsolu üzerinden anomali bildirimi oluşturabilir.
            </p>
          </div>
        );

      default:
        return <p className="text-sm text-muted">İçerik hazırlanıyor...</p>;
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={getTitle()}
      ariaLabel={getTitle()}
      maxWidth="620px"
    >
      <div className="modal-inner-scroll max-h-[70vh] overflow-y-auto pr-1">
        {renderContent()}
      </div>
    </Modal>
  );
}
