import React from 'react';

export function Footer({ onOpenLegal }) {
  return (
    <footer className="app-footer" role="contentinfo">
      <div className="footer-container">
        <div className="footer-nav" aria-label="Alt Bilgi Gezintisi">
          <button
            type="button"
            onClick={() => onOpenLegal('about')}
            className="footer-link"
          >
            Hakkımızda
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal('how-it-works')}
            className="footer-link"
          >
            Nasıl Çalışır?
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal('help')}
            className="footer-link"
          >
            Yardım & SSS
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal('privacy')}
            className="footer-link"
          >
            Gizlilik Politikası
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal('kvkk')}
            className="footer-link"
          >
            KVKK Aydınlatma
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal('terms')}
            className="footer-link"
          >
            Kullanım Koşulları
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal('contact')}
            className="footer-link"
          >
            İletişim & Destek
          </button>
        </div>
        <div className="footer-copyright">
          <p>© 2026 AktarBilet Şehir İçi Entegre Ulaşım ve Akıllı Biletleme Platformu. Tüm hakları saklıdır.</p>
          <p className="text-xs text-muted mt-1">
            Canlı otobüs telemetrisi İzmir Büyükşehir Belediyesi ESHOT Genel Müdürlüğü Açık Veri Portalı entegrasyonu ile sağlanmaktadır.
          </p>
        </div>
      </div>
    </footer>
  );
}
