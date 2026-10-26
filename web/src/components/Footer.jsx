import React from 'react';

export function Footer({ onOpenLegal }) {
  return (
    <footer className="app-footer" role="contentinfo">
      <div className="footer-container">
        {/* Üst Bölüm: 7 Kurumsal ve Yasal Bilgilendirme Bağlantısı */}
        <nav className="footer-nav" aria-label="Alt Bilgi Kurumsal ve Yasal Bağlantılar">
          <button
            type="button"
            onClick={() => onOpenLegal && onOpenLegal('about')}
            className="footer-link"
          >
            Hakkımızda
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal && onOpenLegal('how-it-works')}
            className="footer-link"
          >
            Nasıl Çalışır?
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal && onOpenLegal('help')}
            className="footer-link"
          >
            Yardım & SSS
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal && onOpenLegal('privacy')}
            className="footer-link"
          >
            Gizlilik Politikası
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal && onOpenLegal('kvkk')}
            className="footer-link"
          >
            KVKK Aydınlatma
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal && onOpenLegal('terms')}
            className="footer-link"
          >
            Kullanım Koşulları
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal && onOpenLegal('contact')}
            className="footer-link"
          >
            İletişim & Destek
          </button>
        </nav>

        {/* Bölücü Çizgi */}
        <div className="footer-divider" role="separator" />

        {/* Alt Bölüm: Eksiksiz Telif ve ESHOT Bilgilendirmeleri */}
        <div className="footer-copyright">
          <p className="copyright-text">
            © 2026 AktarBilet Şehir İçi Entegre Ulaşım ve Akıllı Biletleme Platformu. Tüm hakları saklıdır.
          </p>
          <p className="attribution-text">
            Canlı otobüs telemetrisi İzmir Büyükşehir Belediyesi ESHOT Genel Müdürlüğü Açık Veri Portalı entegrasyonu ile sağlanmaktadır.
          </p>
        </div>
      </div>
    </footer>
  );
}
