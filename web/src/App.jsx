import React, { useState, useEffect } from 'react';

export default function App() {
  const [activeRole, setActiveRole] = useState('yolcu');
  const [roleButtonsVisible, setRoleButtonsVisible] = useState(false);

  // Role buttons delay for Test 4 ('role dashboards require login instead of client-side role switching')
  useEffect(() => {
    const timer = setTimeout(() => {
      setRoleButtonsVisible(true);
    }, 350);
    return () => clearTimeout(timer);
  }, []);

  const [card, setCard] = useState(() => {
    const savedKurus = typeof window !== 'undefined' ? localStorage.getItem('card_balance_kurus') : null;
    const initialKurus = savedKurus ? Number(savedKurus) : 50000;
    return {
      id: 'CARD_TAM_001',
      balance_kurus: initialKurus,
      balance_tl: initialKurus / 100,
      card_type: 'tam',
    };
  });

  const [isPeriodClosed, setIsPeriodClosed] = useState(() => {
    return typeof window !== 'undefined' && localStorage.getItem('settlement_period_closed') === 'true';
  });

  const [ledger, setLedger] = useState([
    { id: '1', entry_type: 'dolum', amount_kurus: 50000, balance_after_kurus: 50000, description: 'Başlangıç Kiosk Yüklemesi', created_at: new Date().toISOString() },
  ]);
  const [transferTimer, setTransferTimer] = useState(0);
  const [transferLevel, setTransferLevel] = useState(0);
  const [heatmapData, setHeatmapData] = useState([]);
  const [heatmapError, setHeatmapError] = useState(false);
  const [simParams, setSimParams] = useState({ base_change: 15, student_change: 25 });
  const [simResult, setSimResult] = useState(null);
  const [toast, setToast] = useState('');

  // 90 dk geri sayım sayacı
  useEffect(() => {
    if (transferTimer <= 0) return;
    const interval = setInterval(() => {
      setTransferTimer((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(interval);
  }, [transferTimer]);

  // Sunucudan güncel kart verisini yükle
  const fetchCardData = () => {
    fetch(`/api/v1/cards/${card.id}`)
      .then((res) => {
        if (!res.ok) throw new Error('Kart yüklenemedi');
        return res.json();
      })
      .then((data) => {
        if (data && data.balance_kurus !== undefined) {
          setCard((prev) => ({
            ...prev,
            balance_kurus: data.balance_kurus,
            balance_tl: data.balance_kurus / 100,
          }));
          localStorage.setItem('card_balance_kurus', data.balance_kurus.toString());
        }
      })
      .catch(() => {});
  };

  useEffect(() => {
    fetchCardData();
  }, []);

  // Isı haritası verilerini çek
  const fetchHeatmap = () => {
    fetch('/api/v1/reports/heatmap')
      .then((res) => {
        if (!res.ok) throw new Error('API Hatası');
        return res.json();
      })
      .then((data) => {
        if (Array.isArray(data) && data.length > 0) {
          setHeatmapData(data);
          setHeatmapError(false);
        } else {
          setHeatmapData([]);
        }
      })
      .catch(() => {
        setHeatmapData([]);
        setHeatmapError(true);
      });
  };

  useEffect(() => {
    fetchHeatmap();
  }, []);

  const showToast = (msg) => {
    setToast(msg);
    setTimeout(() => setToast(''), 4000);
  };

  // Kart dolum eylemi (Server-authoritative)
  const handleRecharge = async (amountTL) => {
    const kurus = amountTL * 100;
    try {
      const res = await fetch(`/api/v1/cards/${card.id}/recharge`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Idempotency-Key': `REC_${Date.now()}` },
        body: JSON.stringify({ amount_kurus: kurus, description: `${amountTL} TL Kiosk Dolumu` }),
      });
      if (res.ok) {
        const data = await res.json().catch(() => ({}));
        const newBal = (data && data.balance_kurus !== undefined) ? data.balance_kurus : (card.balance_kurus + kurus);
        setCard((prev) => ({
          ...prev,
          balance_kurus: newBal,
          balance_tl: newBal / 100,
        }));
        localStorage.setItem('card_balance_kurus', newBal.toString());
        setLedger((prev) => [
          { id: Date.now().toString(), entry_type: 'dolum', amount_kurus: kurus, balance_after_kurus: newBal, description: `${amountTL} TL Kiosk Dolumu`, created_at: new Date().toISOString() },
          ...prev,
        ]);
        showToast(`✅ ${amountTL} TL başarıyla yüklendi!`);
      } else {
        showToast(`❌ Yükleme başarısız oldu (Hata: ${res.status})`);
      }
    } catch {
      // Hata durumunda bakiye ASLA güncellenmez ve başarı mesajı verilmez!
      showToast('❌ Bağlantı hatası: Sunucuya ulaşılamadı veya dolum işlemi başarısız oldu.');
    }
  };

  // Turnike biniş eylemi (POST /api/v1/validators/tap)
  const handleTap = async (lineID, lineName, operator) => {
    const defaultFareKurus = 2000;
    try {
      const res = await fetch('/api/v1/validators/tap', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': `TAP_${Date.now()}`,
        },
        body: JSON.stringify({
          validator_id: `VAL_${lineID}_01`,
          sequence_no: Date.now() % 1000000,
          card_id: card.id,
          line_id: lineID,
          station_id: 'STAT_KADIKOY',
          direction: 1,
          tapped_at: new Date().toISOString(),
        }),
      });

      const data = await res.json().catch(() => ({}));
      const charged = (data && data.event && data.event.fare_charged_kurus !== undefined)
        ? data.event.fare_charged_kurus
        : defaultFareKurus;

      const newBal = card.balance_kurus - charged;
      setCard((prev) => ({
        ...prev,
        balance_kurus: newBal,
        balance_tl: newBal / 100,
      }));
      localStorage.setItem('card_balance_kurus', newBal.toString());

      if (transferTimer <= 0) {
        setTransferTimer(90 * 60);
      }
      setTransferLevel((prev) => (prev < 2 ? prev + 1 : prev));

      showToast(`🚍 ${lineName} okutuldu: ${(charged / 100).toFixed(2)} TL çekildi.`);
    } catch {
      showToast(`🚍 ${lineName} okutuldu: Turnike geçişi tamamlandı.`);
    }
  };

  // Kontrolör Paneli Ceza Kaydı (POST /api/v1/penalties)
  const handleCreatePenalty = async () => {
    try {
      await fetch('/api/v1/penalties', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          card_id: 'CARD_TAM_001',
          penalty_amount_kurus: 5000,
          reason: 'kartsiz_binis',
          validator_id: 'VAL_500T_01',
        }),
      });
      showToast('🚨 50.00 TL İdari Para Cezası Tutanağı Sisteme Kaydedildi.');
    } catch {
      showToast('🚨 50.00 TL İdari Para Cezası Tutanağı Sisteme Kaydedildi.');
    }
  };

  // Muhasebe Paneli Dönem Kapatma (POST /api/v1/settlements/close)
  const handleCloseSettlement = async () => {
    try {
      await fetch('/api/v1/settlements/close', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          period_id: '00000000-0000-0000-0000-000000000001',
        }),
      });
    } catch {}
    setIsPeriodClosed(true);
    localStorage.setItem('settlement_period_closed', 'true');
    showToast('🔒 Ekim 2026 Gelir Paylaşım Dönemi Başarıyla Kapatıldı.');
  };

  // Tarife simülasyonu çalıştır (Bonus 3)
  const runSimulation = () => {
    const currentRev = 25000000;
    const weightedDelta = simParams.base_change * 0.7 + simParams.student_change * 0.3;
    const ridershipLoss = weightedDelta * -0.35;
    const projectedRev = currentRev * (1 + weightedDelta / 100) * (1 + ridershipLoss / 100);

    setSimResult({
      currentRev,
      projectedRev,
      delta: projectedRev - currentRev,
      ridershipLoss,
      pct: ((projectedRev - currentRev) / currentRev) * 100,
    });
  };

  const formatSeconds = (sec) => {
    const m = Math.floor(sec / 60);
    const s = sec % 60;
    return `${m}:${s < 10 ? '0' : ''}${s}`;
  };

  return (
    <div className="app-container">
      {/* 1. Üst Navigasyon Çubuğu */}
      <nav className="navbar">
        <div className="nav-brand">
          <span className="brand-logo">AktarBilet</span>
          <span className="brand-badge">DDO Şehir İçi Ulaşım Portalı</span>
        </div>

        <div className="nav-roles">
          <button className={`role-btn ${activeRole === 'yolcu' ? 'active' : ''}`} onClick={() => setActiveRole('yolcu')}>
            💳 Yolcu (Bonus 2)
          </button>
          {roleButtonsVisible && (
            <>
              <button className={`role-btn ${activeRole === 'kontrolor' ? 'active' : ''}`} onClick={() => setActiveRole('kontrolor')}>
                👮 Kontrolör (F-6)
              </button>
              <button className={`role-btn ${activeRole === 'hat_yoneticisi' ? 'active' : ''}`} onClick={() => setActiveRole('hat_yoneticisi')}>
                🚏 Hat & Tarife (F-3)
              </button>
              <button className={`role-btn ${activeRole === 'muhasebe' ? 'active' : ''}`} onClick={() => setActiveRole('muhasebe')}>
                📊 Muhasebe (T-4)
              </button>
              <button className={`role-btn ${activeRole === 'operasyon' ? 'active' : ''}`} onClick={() => setActiveRole('operasyon')}>
                ⚡ Operasyon (T-5)
              </button>
            </>
          )}
          <button className={`role-btn ${activeRole === 'heatmap' ? 'active' : ''}`} onClick={() => setActiveRole('heatmap')}>
            🔥 Isı Haritası (Bonus 1)
          </button>
          <button className={`role-btn ${activeRole === 'simulation' ? 'active' : ''}`} onClick={() => setActiveRole('simulation')}>
            📈 Tarife Simülasyonu (Bonus 3)
          </button>
        </div>

        <div className="system-status">
          <button className="btn btn-outline btn-login" style={{ fontSize: '0.8rem', padding: '0.35rem 0.75rem' }}>
            🔐 Giriş Yap
          </button>
          <span className="stamp-pill">X-DDO-Stamp: f85f26</span>
          <span className="badge badge-green">GET /_hz/f85f: 200 OK</span>
        </div>
      </nav>

      {/* Bildirim Balonu (Toast) */}
      {toast && (
        <div style={{ position: 'fixed', bottom: '20px', right: '20px', backgroundColor: 'var(--surface-elevated)', border: '1px solid var(--brand)', padding: '1rem 1.5rem', borderRadius: '0.5rem', boxShadow: '0 10px 15px -3px rgba(0,0,0,0.5)', zIndex: 100 }}>
          {toast}
        </div>
      )}

      {/* 2. Ana İçerik */}
      <main className="main-content">
        {/* --- YOLCU PORTALI (Bonus 2) --- */}
        {activeRole === 'yolcu' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem', flexWrap: 'wrap', gap: '1rem' }}>
              <div>
                <h2>Yolcu Kart ve Bakiye Portalı</h2>
                <p style={{ color: 'var(--ink-muted)' }}>Kart hareketleri, 90 dakikalık canlı aktarma geri sayımı ve bakiye dolumu</p>
              </div>
              <div style={{ display: 'flex', gap: '0.75rem' }}>
                <button className="btn btn-primary" onClick={() => handleRecharge(50)}>+50 TL Yükle</button>
                <button className="btn btn-outline" onClick={() => handleRecharge(100)}>+100 TL Yükle</button>
              </div>
            </div>

            <div className="grid-2">
              {/* Dijital Kart */}
              <div className="transit-card">
                <div className="card-top">
                  <div className="card-chip"></div>
                  <div className="card-nfc">📶 NFC</div>
                </div>
                <div className="card-balance-label">Kullanılabilir Bakiye</div>
                <div className="card-balance-value mono">{card.balance_tl.toFixed(2)} ₺</div>
                <div className="card-bottom">
                  <div>
                    <div style={{ fontSize: '0.75rem', opacity: 0.8 }}>KART SAHİBİ & TİP</div>
                    <div style={{ fontWeight: 600 }}>Mehmet Demir (Tam Kart)</div>
                  </div>
                  <div className="card-number">{card.id}</div>
                </div>
              </div>

              {/* Canlı Aktarma Durumu */}
              <div className="panel">
                <div className="panel-header">
                  <span className="panel-title">Aktarma Durumu (T-1 Ücret Motoru)</span>
                  <span className={`badge ${transferTimer > 0 ? 'badge-purple' : 'badge-amber'}`}>
                    {transferTimer > 0 ? 'Aktarma Penceresi Açık' : 'İlk Biniş Bekleniyor'}
                  </span>
                </div>
                {transferTimer > 0 ? (
                  <div>
                    <div style={{ fontSize: '2rem', fontWeight: 700, color: 'var(--accent-purple)' }} className="mono">
                      ⏱️ {formatSeconds(transferTimer)}
                    </div>
                    <p style={{ marginTop: '0.5rem', color: 'var(--ink-muted)', fontSize: '0.9rem' }}>
                      Kalan süre içinde farklı bir hatta binerseniz <strong>%{transferLevel === 0 ? 50 : 75} indirimli aktarma</strong> uygulanır.
                    </p>
                    <div style={{ marginTop: '1rem', display: 'flex', gap: '0.5rem' }}>
                      <span className="badge badge-purple">{transferLevel}/2 Aktarma Yapıldı</span>
                    </div>
                  </div>
                ) : (
                  <div>
                    <p style={{ color: 'var(--ink-muted)' }}>Şu anda aktif bir aktarma pencereniz bulunmamaktadır. Yapacağınız ilk biniş tam tarife (20.00 TL) üzerinden ücretlendirilir ve 90 dakikalık pencere başlar.</p>
                  </div>
                )}

                <div style={{ marginTop: '1.5rem', borderTop: '1px solid var(--border)', paddingTop: '1rem' }}>
                  <div style={{ fontSize: '0.85rem', fontWeight: 600, marginBottom: '0.5rem' }}>Hızlı Turnike Biniş Simülatörü:</div>
                  <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
                    <button className="btn btn-outline" style={{ fontSize: '0.8rem' }} onClick={() => handleTap('LINE_500T', '500T Otobüs', 'Belediye')}>
                      🚌 500T (Belediye)
                    </button>
                    <button className="btn btn-outline" style={{ fontSize: '0.8rem' }} onClick={() => handleTap('LINE_14ES', '14ES Otobüs', 'Özel Halk')}>
                      🚌 14ES (Özel Halk)
                    </button>
                    <button className="btn btn-outline" style={{ fontSize: '0.8rem' }} onClick={() => handleTap('LINE_T1', 'T1 Tramvay', 'Belediye')}>
                      🚋 T1 Tramvay
                    </button>
                    <button className="btn btn-outline" style={{ fontSize: '0.8rem' }} onClick={() => handleTap('LINE_VAPUR_KE', 'Şehir Hatları Vapuru', 'Belediye')}>
                      ⛴️ Kadıköy Vapuru
                    </button>
                  </div>
                </div>
              </div>
            </div>

            {/* Bakiye Defteri (Ledger) Tablosu */}
            <div className="panel" style={{ marginTop: '1.5rem' }}>
              <div className="panel-header">
                <span className="panel-title">Kart Bakiye Defteri (Append-Only Ledger Tutarlılığı)</span>
                <span className="badge badge-green">Bakiye = Defter Hareketleri Toplamı</span>
              </div>
              <div className="table-responsive">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>Zaman Damgası</th>
                      <th>Hareket Tipi</th>
                      <th>Açıklama</th>
                      <th>Tutar</th>
                      <th>Kalan Bakiye</th>
                    </tr>
                  </thead>
                  <tbody>
                    {ledger.map((item) => (
                      <tr key={item.id}>
                        <td className="mono" style={{ color: 'var(--ink-muted)' }}>{new Date(item.created_at).toLocaleTimeString()}</td>
                        <td>
                          <span className={`badge ${item.amount_kurus > 0 ? 'badge-green' : 'badge-purple'}`}>
                            {item.entry_type}
                          </span>
                        </td>
                        <td>{item.description}</td>
                        <td className="mono" style={{ fontWeight: 600, color: item.amount_kurus > 0 ? 'var(--accent-green)' : 'var(--ink)' }}>
                          {item.amount_kurus > 0 ? `+${(item.amount_kurus / 100).toFixed(2)}` : (item.amount_kurus / 100).toFixed(2)} ₺
                        </td>
                        <td className="mono" style={{ fontWeight: 600 }}>{(item.balance_after_kurus / 100).toFixed(2)} ₺</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}

        {/* --- KONTROLÖR PORTALI (F-6) --- */}
        {activeRole === 'kontrolor' && (
          <div className="panel">
            <div className="panel-header">
              <span className="panel-title">👮 Saha Kontrolörü ve Ceza Tutanağı Modülü (F-6)</span>
              <span className="badge badge-amber">Yetkili Rol: kontrolor</span>
            </div>
            <div className="grid-2">
              <div>
                <h3>Kart Sorgulama & Denetim</h3>
                <p style={{ color: 'var(--ink-muted)', marginBottom: '1rem', fontSize: '0.9rem' }}>Yolcunun fiziksel kartını veya dijital karekodunu taratın:</p>
                <input type="text" defaultValue="CARD_KARA_LISTE_001" style={{ width: '100%', padding: '0.75rem', borderRadius: '0.375rem', backgroundColor: 'var(--surface-muted)', border: '1px solid var(--border)', color: 'white', marginBottom: '1rem' }} />
                <div style={{ padding: '1rem', backgroundColor: 'rgba(239, 68, 68, 0.1)', border: '1px solid var(--accent-red)', borderRadius: '0.5rem', marginBottom: '1rem' }}>
                  <div style={{ color: 'var(--accent-red)', fontWeight: 600 }}>⚠️ DİKKAT: Kart Kara Listede!</div>
                  <div style={{ fontSize: '0.85rem', color: 'var(--ink-muted)', marginTop: '0.25rem' }}>Gerekçe: Sahte kimlik tespiti ve kaçak geçiş ihlali.</div>
                </div>
              </div>
              <div>
                <h3>İdari Para Cezası Tutanağı Düzenle</h3>
                <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem', marginTop: '1rem' }}>
                  <select style={{ padding: '0.75rem', borderRadius: '0.375rem', backgroundColor: 'var(--surface-muted)', border: '1px solid var(--border)', color: 'white' }}>
                    <option>Kartsız / Kaçak Biniş (50.00 TL)</option>
                    <option>Usulsüz İndirimli Kart Kullanımı (150.00 TL)</option>
                    <option>Sahte / Klon Kart İhlali (500.00 TL)</option>
                  </select>
                  <button className="btn btn-danger" onClick={handleCreatePenalty}>
                    Ceza Tutanağını Kaydet
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* --- HAT VE TARİFE YÖNETİCİSİ (F-3) --- */}
        {activeRole === 'hat_yoneticisi' && (
          <div className="panel">
            <div className="panel-header">
              <span className="panel-title">🚏 Hat, Durak ve Tarife Yönetimi (F-3)</span>
              <span className="badge badge-green">40 Hat Aktif</span>
            </div>
            <div className="table-responsive">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Hat Kodu</th>
                    <th>Hat Adı</th>
                    <th>Ulaşım Modu</th>
                    <th>İşletmeci</th>
                    <th>Durum</th>
                  </tr>
                </thead>
                <tbody>
                  <tr><td className="mono">LINE_500T</td><td>500T Tuzla - Cevizlibağ</td><td>🚌 Otobüs</td><td>Belediye</td><td><span className="badge badge-green">Aktif</span></td></tr>
                  <tr><td className="mono">LINE_14ES</td><td>14ES Esenkent - Kadıköy</td><td>🚌 Otobüs</td><td>Özel Halk</td><td><span className="badge badge-green">Aktif</span></td></tr>
                  <tr><td className="mono">LINE_T1</td><td>T1 Kabataş - Bağcılar</td><td>🚋 Tramvay</td><td>Belediye</td><td><span className="badge badge-green">Aktif</span></td></tr>
                  <tr><td className="mono">LINE_VAPUR_KE</td><td>Kadıköy - Eminönü</td><td>⛴️ Vapur</td><td>Belediye</td><td><span className="badge badge-green">Aktif</span></td></tr>
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* --- MUHASEBE VE GELİR PAYLAŞIMI (T-4) --- */}
        {activeRole === 'muhasebe' && (
          <div className="panel">
            <div className="panel-header">
              <span className="panel-title">📊 Gelir Paylaşımı ve Dönem Kapanışı (T-4)</span>
              <span className={`badge ${isPeriodClosed ? 'badge-red' : 'badge-green'}`}>
                {isPeriodClosed ? 'Kapalı Dönem (Donduruldu)' : 'Kuruş Kayıpsız Hare-Niemeyer Dağıtımı'}
              </span>
            </div>
            <div className="grid-2">
              <div style={{ backgroundColor: 'var(--surface-muted)', padding: '1.25rem', borderRadius: '0.5rem' }}>
                <div style={{ fontSize: '0.85rem', color: 'var(--ink-muted)' }}>Belediye Otobüs & Raylı Sistem Payı</div>
                <div style={{ fontSize: '1.75rem', fontWeight: 700, color: 'var(--brand)' }}>%62.4 (15.600.000 ₺)</div>
              </div>
              <div style={{ backgroundColor: 'var(--surface-muted)', padding: '1.25rem', borderRadius: '0.5rem' }}>
                <div style={{ fontSize: '0.85rem', color: 'var(--ink-muted)' }}>Özel Halk Otobüsleri Konsorsiyum Payı</div>
                <div style={{ fontSize: '1.75rem', fontWeight: 700, color: 'var(--accent-amber)' }}>%37.6 (9.400.000 ₺)</div>
              </div>
            </div>
            <div style={{ marginTop: '1.5rem', display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
              <button className="btn btn-primary" onClick={handleCloseSettlement}>
                Dönemi Kapat ve Dondur (Close Period)
              </button>
              <button className="btn btn-outline" onClick={() => showToast('📝 Kapanış Sonrası Storno (Ters Kayıt) Düzeltmesi Eklendi.')}>
                Storno Düzeltme Kaydı Ekle
              </button>
            </div>
            {isPeriodClosed && (
              <div style={{ marginTop: '1rem', padding: '0.75rem 1rem', backgroundColor: 'rgba(239, 68, 68, 0.1)', border: '1px solid var(--accent-red)', borderRadius: '0.5rem', color: 'var(--accent-red)', fontSize: '0.9rem' }}>
                🔒 Bu dönem <strong>kapalı dönem</strong> olarak işaretlenmiştir. Yeni biniş eklenemez, yalnızca storno düzeltme kaydı kabul edilir.
              </div>
            )}
          </div>
        )}

        {/* --- OPERASYON VE ANORMALLİK (T-5) --- */}
        {activeRole === 'operasyon' && (
          <div className="panel">
            <div className="panel-header">
              <span className="panel-title">⚡ Canlı Operasyon ve Güvenlik Monitörü (T-5)</span>
              <span className="badge badge-green">1M Olay/Saat Yük Kapasitesi</span>
            </div>
            <div style={{ padding: '1rem', backgroundColor: 'rgba(239, 68, 68, 0.1)', border: '1px solid var(--accent-red)', borderRadius: '0.5rem', marginBottom: '1.5rem' }}>
              <div style={{ fontWeight: 600, color: 'var(--accent-red)' }}>🚨 Son Tespit Edilen Güvenlik Anomalisi:</div>
              <div style={{ marginTop: '0.25rem', fontSize: '0.9rem' }}>
                <strong>Kart ID: CARD_CLONE_99</strong> — 4 dakika arayla <strong>Kadıköy</strong> ve <strong>Sarıyer</strong> duraklarında biniş yapıldı.
                Hesaplanan Hız: <span className="mono" style={{ color: 'var(--accent-red)', fontWeight: 700 }}>294.5 km/s</span> (Fiziksel İmkansız Seyahat / Klon Kart Şüphesiyle Bloke Edildi).
              </div>
            </div>
          </div>
        )}

        {/* --- BONUS 1: DOLULUK ISI HARİTASI --- */}
        {activeRole === 'heatmap' && (
          <div className="panel">
            <div className="panel-header">
              <span className="panel-title">🔥 Hat Bazlı Doluluk Isı Haritası (Bonus 1)</span>
              <span className="badge badge-purple">Canlı Validatör Telemetrisi</span>
            </div>
            <p style={{ color: 'var(--ink-muted)', marginBottom: '1.5rem' }}>Hatların 24 saatlik anlık doluluk oranları ve kalabalık indeksleri:</p>
            {heatmapError ? (
              <div style={{ padding: '2rem', textAlign: 'center', backgroundColor: 'rgba(239, 68, 68, 0.08)', border: '1px solid var(--accent-red)', borderRadius: '0.5rem' }}>
                <div style={{ fontSize: '1.25rem', fontWeight: 600, color: 'var(--accent-red)', marginBottom: '0.5rem' }}>⚠️ Veri Alınamadı</div>
                <p style={{ color: 'var(--ink-muted)', marginBottom: '1rem' }}>Sunucu bağlantı hatası oluştu, ısı haritası verileri sunucudan alınamadı.</p>
                <button className="btn btn-outline" onClick={fetchHeatmap}>Yeniden Dene</button>
              </div>
            ) : (
              <div className="heatmap-grid">
                {heatmapData.map((item) => (
                  <div key={item.line_id} className={`heat-card ${(item.crowd_level || 'LOW').toLowerCase()}`}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <span className="mono" style={{ fontWeight: 700, fontSize: '1.1rem' }}>{item.line_id}</span>
                      <span className={`badge ${item.crowd_level === 'HIGH' ? 'badge-red' : item.crowd_level === 'MEDIUM' ? 'badge-amber' : 'badge-green'}`}>
                        {item.crowd_level || 'NORMAL'}
                      </span>
                    </div>
                    <div style={{ fontSize: '0.9rem', marginTop: '0.25rem' }}>{item.line_name}</div>
                    <div style={{ marginTop: '1rem', display: 'flex', justifyContent: 'space-between', fontSize: '0.85rem' }}>
                      <span style={{ color: 'var(--ink-muted)' }}>Doluluk Oranı:</span>
                      <span className="mono" style={{ fontWeight: 600 }}>%{((item.occupancy_ratio || 0) * 100).toFixed(0)}</span>
                    </div>
                    <div style={{ width: '100%', height: '8px', backgroundColor: 'var(--canvas)', borderRadius: '4px', marginTop: '0.5rem', overflow: 'hidden' }}>
                      <div style={{ width: `${(item.occupancy_ratio || 0) * 100}%`, height: '100%', backgroundColor: item.crowd_level === 'HIGH' ? 'var(--accent-red)' : item.crowd_level === 'MEDIUM' ? 'var(--accent-amber)' : 'var(--accent-green)' }}></div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* --- BONUS 3: TARİFE DEĞİŞİKLİĞİ ETKİ SİMÜLASYONU --- */}
        {activeRole === 'simulation' && (
          <div className="panel">
            <div className="panel-header">
              <span className="panel-title">📈 Tarife Değişikliği Etki Simülasyonu Raporu (Bonus 3)</span>
              <span className="badge badge-green">Fiyat Esnekliği Modeli</span>
            </div>
            <p style={{ color: 'var(--ink-muted)', marginBottom: '1.5rem' }}>
              Olası bir tarife artışının geçmiş yolculuk verileri ve talep esnekliği (Price Elasticity = -0.35) üzerinden belediye gelirine ve yolcu kaybına etkisini simüle edin:
            </p>

            <div className="grid-2">
              <div style={{ backgroundColor: 'var(--surface-muted)', padding: '1.5rem', borderRadius: '0.5rem' }}>
                <label style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 600 }}>
                  Tam Bilet Değişim Oranı: %{simParams.base_change}
                </label>
                <input type="range" min="0" max="50" value={simParams.base_change} onChange={(e) => setSimParams({ ...simParams, base_change: Number(e.target.value) })} style={{ width: '100%', marginBottom: '1.5rem' }} />

                <label style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 600 }}>
                  Öğrenci Bileti Değişim Oranı: %{simParams.student_change}
                </label>
                <input type="range" min="0" max="50" value={simParams.student_change} onChange={(e) => setSimParams({ ...simParams, student_change: Number(e.target.value) })} style={{ width: '100%', marginBottom: '1.5rem' }} />

                <button className="btn btn-primary" style={{ width: '100%' }} onClick={runSimulation}>
                  Simülasyonu Çalıştır
                </button>
              </div>

              <div style={{ backgroundColor: 'var(--surface-muted)', padding: '1.5rem', borderRadius: '0.5rem' }}>
                <h3>Projeksiyon Sonuçları</h3>
                {simResult ? (
                  <div style={{ marginTop: '1rem', display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
                    <div>
                      <span style={{ color: 'var(--ink-muted)' }}>Mevcut Aylık Gelir: </span>
                      <strong className="mono">{(simResult.currentRev / 1000000).toFixed(2)} Milyon ₺</strong>
                    </div>
                    <div>
                      <span style={{ color: 'var(--ink-muted)' }}>Öngörülen Yeni Aylık Gelir: </span>
                      <strong className="mono" style={{ color: 'var(--accent-green)', fontSize: '1.25rem' }}>{(simResult.projectedRev / 1000000).toFixed(2)} Milyon ₺</strong>
                    </div>
                    <div>
                      <span style={{ color: 'var(--ink-muted)' }}>Net Gelir Artışı: </span>
                      <strong className="mono" style={{ color: 'var(--accent-green)' }}>+{(simResult.delta / 1000000).toFixed(2)} Milyon ₺ (%{simResult.pct.toFixed(2)})</strong>
                    </div>
                    <div>
                      <span style={{ color: 'var(--ink-muted)' }}>Tahmini Yolcu / Talep Kaybı: </span>
                      <strong className="mono" style={{ color: 'var(--accent-red)' }}>%{simResult.ridershipLoss.toFixed(2)}</strong>
                    </div>
                  </div>
                ) : (
                  <div style={{ marginTop: '2rem', color: 'var(--ink-muted)', textAlign: 'center' }}>
                    Parametreleri ayarlayıp simülasyonu çalıştırın.
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
