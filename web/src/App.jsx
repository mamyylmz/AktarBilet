import React, { useState, useEffect, useRef } from 'react';
import { Navbar } from './components/Navbar.jsx';
import { Footer } from './components/Footer.jsx';
import { Toast } from './components/ui/Toast.jsx';
import { Card } from './components/ui/Card.jsx';
import { Button } from './components/ui/Button.jsx';
import { LoginModal } from './features/auth/LoginModal.jsx';
import { LegalModal } from './features/public/LegalModal.jsx';
import { HomePage } from './features/public/HomePage.jsx';
import { PassengerPortal } from './features/passenger/PassengerPortal.jsx';
import { api } from './services/api.js';
import { formatDateTime } from './lib/formatters.js';

const parseTLAmountToKurus = (value) => {
  const match = String(value ?? '').trim().replace(',', '.').match(/^(\d+)(?:\.(\d{1,2}))?$/);
  if (!match) return null;
  const whole = Number(match[1]);
  const fraction = Number((match[2] || '').padEnd(2, '0'));
  const amount = whole * 100 + fraction;
  return Number.isSafeInteger(amount) && amount > 0 ? amount : null;
};

// Section 18: Code-splitting and React.lazy for portal components to minimize TBT and initial bundle size
const InspectorPortal = React.lazy(() =>
  import('./features/inspector/InspectorPortal.jsx').then((m) => ({ default: m.InspectorPortal }))
);
const LineManagerPortal = React.lazy(() =>
  import('./features/linemanager/LineManagerPortal.jsx').then((m) => ({ default: m.LineManagerPortal }))
);
const AccountingPortal = React.lazy(() =>
  import('./features/accounting/AccountingPortal.jsx').then((m) => ({ default: m.AccountingPortal }))
);
const OperationsPortal = React.lazy(() =>
  import('./features/operations/OperationsPortal.jsx').then((m) => ({ default: m.OperationsPortal }))
);
const AdminPortal = React.lazy(() =>
  import('./features/admin/AdminPortal.jsx').then((m) => ({ default: m.AdminPortal }))
);

export default function App() {
  // 1. Theme State (Section 7: Default LIGHT THEME, toggleable to dark)
  const [theme, setTheme] = useState(() => {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('aktarbilet_theme') || 'light';
    }
    return 'light';
  });

  useEffect(() => {
    if (typeof document !== 'undefined') {
      document.documentElement.setAttribute('data-theme', theme);
      localStorage.setItem('aktarbilet_theme', theme);
    }
  }, [theme]);

  const toggleTheme = () => {
    setTheme((prev) => (prev === 'light' ? 'dark' : 'light'));
  };

  // 2. Auth State (Section 3: Zero monkey-patches, zero hardcoded default passenger tokens)
  const [authToken, setAuthToken] = useState(() => {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('auth_token') || '';
    }
    return '';
  });

  const [currentUser, setCurrentUser] = useState(() => {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem('auth_user');
      return saved ? JSON.parse(saved) : null;
    }
    return null;
  });

  // Central active token getter to prevent stale role tokens from intercepting authenticated requests
  const getActiveToken = () => {
    return authToken || (typeof window !== 'undefined' ? localStorage.getItem('auth_token') || '' : '');
  };

  // 3. Navigation State (Anonymous user defaults to public 'home')
  const [activeRole, setActiveRoleState] = useState(() => {
    if (typeof window !== 'undefined') {
      const savedRole = localStorage.getItem('active_role');
      if (savedRole) return savedRole;
      const savedUser = localStorage.getItem('auth_user');
      if (savedUser) {
        try {
          const u = JSON.parse(savedUser);
          if (u?.role) return u.role;
        } catch {}
      }
    }
    return 'home';
  });

  const setActiveRole = (role) => {
    setActiveRoleState(role);
    if (typeof window !== 'undefined') {
      localStorage.setItem('active_role', role);
    }
  };

  const [showLoginModal, setShowLoginModal] = useState(false);
  const [activeLegalModal, setActiveLegalModal] = useState(null);
  const loginButtonRef = useRef(null);

  // 4. Passenger Card & Ledger State
  const [card, setCard] = useState(null);
  const [ledger, setLedger] = useState([]);
  const [cardDataLoading, setCardDataLoading] = useState(false);
  const [cardDataError, setCardDataError] = useState('');
  const cardRefreshRequestRef = useRef(0);

  const [transferTimer, setTransferTimer] = useState(0);
  const [transferLevel, setTransferLevel] = useState(0);
  const [rechargeLoading, setRechargeLoading] = useState(false);
  const [tapLoading, setTapLoading] = useState(false);

  // 90 min countdown timer effect
  useEffect(() => {
    if (transferTimer <= 0) {
      setTransferLevel(0);
      return;
    }
    const interval = setInterval(() => {
      setTransferTimer((prev) => {
        if (prev <= 1) {
          setTransferLevel(0);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(interval);
  }, [transferTimer]);

  // 5. Accounting State
  const [isPeriodClosed, setIsPeriodClosed] = useState(() => {
    return typeof window !== 'undefined' && localStorage.getItem('settlement_period_closed') === 'true';
  });
  const [settlementData, setSettlementData] = useState(null);
  const [loadingSettlements, setLoadingSettlements] = useState(false);
  const [closingPeriod, setClosingPeriod] = useState(false);
  const [addingSubsidyAdjustment, setAddingSubsidyAdjustment] = useState(false);

  // 6. Inspector State
  const [queriedCard, setQueriedCard] = useState(null);
  const [queryCardLoading, setQueryCardLoading] = useState(false);
  const [queryCardError, setQueryCardError] = useState(null);
  const queryCardRequestRef = useRef(0);
  const [penalties, setPenalties] = useState([]);
  const [penaltyLoading, setPenaltyLoading] = useState(false);

  // 7. Line Manager State
  const [lineFinancials, setLineFinancials] = useState([]);
  const [loadingLineFinancials, setLoadingLineFinancials] = useState(false);
  const [optimizationResult, setOptimizationResult] = useState(null);
  const [simulatingScenario, setSimulatingScenario] = useState(false);
  const [approvingScenario, setApprovingScenario] = useState(false);

  // 8. Operations & Transit State (Gerçek ESHOT API — Zero Fake Buses)
  const [selectedTransitLine, setSelectedTransitLine] = useState('800');
  const [liveBuses, setLiveBuses] = useState([]);
  const [liveTransitMeta, setLiveTransitMeta] = useState(null);
  const [liveTransitLoading, setLiveTransitLoading] = useState(false);
  const [liveTransitError, setLiveTransitError] = useState(false);
  const liveTransitRequestRef = useRef(0);

  // Heatmap State
  const [heatmapData, setHeatmapData] = useState([]);
  const [heatmapError, setHeatmapError] = useState(false);
  const [heatmapLoading, setHeatmapLoading] = useState(false);
  const [heatmapUpdatedAt, setHeatmapUpdatedAt] = useState(null);
  const heatmapRequestRef = useRef(0);

  // Toast Notification State
  const [toast, setToast] = useState(null);

  const showToast = (message, type = 'info') => {
    setToast({ id: Date.now(), message, type });
  };

  // Helper: enforce login before accessing protected role
  const handleProtectedRoleSelect = (role) => {
    if (!authToken && role !== 'home' && role !== 'map' && role !== 'heatmap') {
      setShowLoginModal(true);
      showToast('Bu panele erişmek için lütfen giriş yapınız.', 'info');
      return;
    }
    setActiveRole(role);
  };

  // Fetch live buses from ESHOT endpoint
  const fetchLiveBuses = async (lineNo) => {
    const requestID = ++liveTransitRequestRef.current;
    setLiveTransitLoading(true);
    setLiveTransitError(false);
    setLiveBuses([]);
    setLiveTransitMeta(null);
    try {
      const data = await api.getLiveBuses(lineNo);
      if (!data || !Array.isArray(data.buses)) throw new Error('Canlı araç verisi geçersiz');
      if (requestID === liveTransitRequestRef.current) {
        setLiveBuses(data.buses);
        setLiveTransitMeta({
          line_number: data.line_no || lineNo,
          total_buses: data.buses.length,
          retrieved_at: data.retrieved_at || null,
          attribution: data.attribution || 'İzmir Büyükşehir Belediyesi ESHOT Açık Veri',
          is_stale: Boolean(data.is_stale),
        });
      }
    } catch {
      if (requestID === liveTransitRequestRef.current) {
        setLiveTransitError(true);
      }
    } finally {
      if (requestID === liveTransitRequestRef.current) {
        setLiveTransitLoading(false);
      }
    }
  };

  // Fetch heatmap data with strict array normalization (Section 12.2)
  const fetchHeatmap = async () => {
    const requestID = ++heatmapRequestRef.current;
    setHeatmapLoading(true);
    setHeatmapError(false);
    try {
      const res = await fetch('/api/v1/reports/heatmap');
      if (!res.ok) throw new Error('Yoğunluk verisi alınamadı');
      const data = await res.json();
      const items = Array.isArray(data) ? data : (data?.items || data?.stations || []);
      if (requestID === heatmapRequestRef.current) {
        setHeatmapData(items);
        setHeatmapUpdatedAt(new Date().toISOString());
      }
    } catch {
      if (requestID === heatmapRequestRef.current) {
        setHeatmapError(true);
        setHeatmapData([]);
      }
    } finally {
      if (requestID === heatmapRequestRef.current) setHeatmapLoading(false);
    }
  };

  // Fetch line financials for Line Manager (Section 10.1)
  const fetchLineFinancials = async () => {
    setLoadingLineFinancials(true);
    try {
      const token = getActiveToken();
      const res = await api.getLineFinancials(token);
      setLineFinancials(Array.isArray(res) ? res : []);
    } catch {
      setLineFinancials([]);
    } finally {
      setLoadingLineFinancials(false);
    }
  };

  // Fetch settlements for Accounting (Section 11)
  const fetchSettlements = async () => {
    setLoadingSettlements(true);
    try {
      const token = getActiveToken();
      const res = await api.getSettlements(token);
      if (res) {
        setSettlementData(res);
        if (typeof res.is_closed === 'boolean') {
          setIsPeriodClosed(res.is_closed);
          localStorage.setItem('settlement_period_closed', res.is_closed ? 'true' : 'false');
        }
      }
    } catch {
      // Keep existing state
    } finally {
      setLoadingSettlements(false);
    }
  };

  // Card and ledger are read in one API/DB snapshot; stale responses cannot replace newer account data.
  const refreshCardData = async (requestedCardID = null, tokenOverride = getActiveToken()) => {
    const requestID = ++cardRefreshRequestRef.current;
    if (!tokenOverride) {
      setCard(null);
      setLedger([]);
      setCardDataError('Kart bilgisi için yolcu hesabınızla giriş yapın.');
      return false;
    }

    setCardDataLoading(true);
    setCardDataError('');
    try {
      const identity = requestedCardID ? null : await api.getCurrentUser(tokenOverride);
      const cardID = requestedCardID || identity?.card_id;
      if (requestID !== cardRefreshRequestRef.current) return false;
      if (!cardID) {
        setCard(null);
        setLedger([]);
        setCardDataError('Hesabınıza bağlı kart bulunamadı.');
        return false;
      }

      const snapshot = await api.getCardSnapshot(cardID, tokenOverride);
      if (requestID !== cardRefreshRequestRef.current) return false;
      if (snapshot.card_id !== cardID || !Number.isSafeInteger(Number(snapshot.balance_kurus)) || !Array.isArray(snapshot.ledger)) {
        throw new Error('Kart ve defter yanıtı geçersiz.');
      }
      setCard({
        id: snapshot.card_id,
        card_type: snapshot.card_type,
        balance_kurus: Number(snapshot.balance_kurus),
      });
      setLedger(snapshot.ledger);
      setCardDataError('');
      return true;
    } catch (err) {
      if (requestID === cardRefreshRequestRef.current) {
        setCard(null);
        setLedger([]);
        setCardDataError(err.message || 'Kart ve hareket bilgileri alınamadı.');
      }
      return false;
    } finally {
      if (requestID === cardRefreshRequestRef.current) setCardDataLoading(false);
    }
  };

  // Trigger role-specific data fetching on navigation
  useEffect(() => {
    if (activeRole === 'heatmap') {
      fetchHeatmap();
    } else if (activeRole === 'operasyon' || activeRole === 'map') {
      fetchLiveBuses(selectedTransitLine);
    } else if (activeRole === 'hat_yoneticisi') {
      fetchLineFinancials();
    } else if (activeRole === 'muhasebe') {
      fetchSettlements();
    } else if (activeRole === 'yolcu' && authToken) {
      refreshCardData(null, authToken);
    }
  }, [activeRole, authToken, currentUser?.id, currentUser?.user_id]);

  useEffect(() => {
    if (!authToken || activeRole !== 'yolcu') return undefined;
    const refreshWhenVisible = () => {
      if (document.visibilityState === 'visible') refreshCardData(null, authToken);
    };
    document.addEventListener('visibilitychange', refreshWhenVisible);
    window.addEventListener('focus', refreshWhenVisible);
    return () => {
      document.removeEventListener('visibilitychange', refreshWhenVisible);
      window.removeEventListener('focus', refreshWhenVisible);
    };
  }, [activeRole, authToken]);

  // Recharge card action (Section 8.2)
  const handleRecharge = async (amountTL) => {
    const kurus = parseTLAmountToKurus(amountTL);
    const token = getActiveToken();
    if (!card?.id || !token) {
      showToast('Dolum için giriş yapıp hesabınıza bağlı kartı yükleyin.', 'error');
      return;
    }
    if (kurus === null) {
      showToast('Dolum tutarı en fazla iki ondalık basamak içermelidir.', 'error');
      return;
    }

    setRechargeLoading(true);
    try {
      await api.rechargeCard(card.id, kurus, `${amountTL} TL kiosk dolumu`, token, `REC_${crypto.randomUUID()}`);
      if (await refreshCardData(card.id, token)) {
        showToast(`${amountTL} TL yüklendi; bakiye ve hareketler sunucudan yenilendi.`, 'success');
      } else {
        showToast('Dolum kaydedildi ancak güncel bakiye ve hareketler alınamadı.', 'warning');
      }
    } catch (err) {
      showToast(`Dolum tamamlanamadı: ${err.message || 'Sunucuya ulaşılamadı.'}`, 'error');
    } finally {
      setRechargeLoading(false);
    }
  };

  // Turnstile tap action (Gerçek API Yanıtı — Sahte Başarı Bildirimi Yok)
  const handleTapTurnstile = async (lineId = 'LINE_5', lineName = '5') => {
    if (tapLoading) return;
    setTapLoading(true);
    const cardId = card?.id;
    if (!cardId) {
      setTapLoading(false);
      showToast('Biniş için giriş yapıp hesabınıza bağlı kartı yükleyin.', 'error');
      return;
    }

    const VALIDATOR_LINE_MAP = {
      LINE_5: { valId: 'VAL_500T_01', statId: 'STAT_ESHOT_10177' },
      LINE_10: { valId: 'VAL_14ES_01', statId: 'STAT_ESHOT_10145' },
      LINE_15: { valId: 'VAL_T1_01', statId: 'STAT_ESHOT_10008' },
      LINE_16: { valId: 'VAL_VAPUR_01', statId: 'STAT_ESHOT_10255' },
      default: { valId: 'VAL_500T_01', statId: 'STAT_ESHOT_10177' },
    };

    const targetVal = VALIDATOR_LINE_MAP[lineId] || VALIDATOR_LINE_MAP.default;
    const now = Date.now();
    const seq = now;
    const idempKey = `TAP_${cardId}_${lineId}_${now}`;

    try {
      const res = await fetch('/api/v1/validators/tap', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Validator-ID': targetVal.valId,
          'Idempotency-Key': idempKey,
          'X-Device-Nonce': `nonce_${now}`,
          'X-Device-Timestamp': `${Math.floor(now / 1000)}`,
          'X-Device-Signature': 'sig_browser_turnstile',
        },
        body: JSON.stringify({
          validator_id: targetVal.valId,
          card_id: cardId,
          line_id: lineId,
          station_id: targetVal.statId,
          sequence_no: seq,
          direction: 1,
          tapped_at: new Date().toISOString(),
        }),
      });

      if (res.ok) {
        const data = await res.json().catch(() => ({}));
        const ev = data?.event;
        if (ev && ev.status === 'REJECTED') {
          showToast(`Turnike geçiş reddedildi: ${ev.rejection_reason || 'Kart kabul edilmedi.'}`, 'error');
        } else {
          const fareCharged = Number(ev?.fare_charged_kurus);
          if (!Number.isSafeInteger(fareCharged)) {
            showToast('Biniş yanıtında ücret bilgisi bulunamadı; hareketler yenilendi.', 'warning');
            await refreshCardData(cardId, getActiveToken());
            return;
          }
          const fareTL = (fareCharged / 100).toFixed(2);

          // Kesin Kural: En fazla 2 aktarma yapılabilir (3. veya 4. aktarma kesinlikle yok)
          if (ev?.is_transfer && ev.transfer_level <= 2) {
            showToast(`Kartınız ${lineName} turnikesinde başarıyla okutuldu. (Aktarma #${ev.transfer_level} - Maks. 2 Aktarma) Tutar: ₺${fareTL}`, 'success');
            if (ev.transfer_level === 1) {
              setTransferLevel(2); // 1. aktarma yapıldı, sırada 2. aktarma var
            } else {
              setTransferLevel(0); // 2. aktarma tamamlandı, aktarma hakkı bitti (3. aktarma yok)
            }
          } else {
            // Aktarma değil: İlk biniş veya 2 aktarma kotası dolmuş biniş
            if (transferTimer <= 0) {
              showToast(`Kartınız ${lineName} turnikesinde başarıyla okutuldu. (İlk Biniş) Tutar: ₺${fareTL}`, 'success');
              setTransferTimer(5400); // 90 min countdown
              setTransferLevel(1); // 1. aktarma hakkı aktif
            } else {
              showToast(`Kartınız ${lineName} turnikesinde okutuldu. (Tam Tarife: ₺${fareTL} - En fazla 2 aktarma kuralı nedeniyle 3. veya ilave aktarma yapılamaz)`, 'info');
              setTransferLevel(0); // Kota dolu, 3. veya daha fazla aktarma yok
            }
          }
        }

        // Bakiye ve defteri API'den anında senkronize et
        await refreshCardData(cardId, getActiveToken());
      } else {
        const errData = await res.json().catch(() => ({}));
        let msg = errData?.error?.message || errData?.message;
        if (!msg) {
          switch (res.status) {
            case 400: msg = 'Geçersiz biniş parametreleri veya veri formatı.'; break;
            case 401: msg = 'Turnike cihaz kimlik doğrulaması başarısız.'; break;
            case 403: msg = 'Validatör cihazı devre dışı veya yetkisiz hat/durak.'; break;
            case 409: msg = 'Mükerrer biniş engellendi (Replay / Sıra koruması).'; break;
            case 422: msg = 'Doğrulama hatası: Geçersiz kart veya tarife.'; break;
            case 429: msg = 'Çok fazla istek yapıldı, lütfen biraz bekleyiniz.'; break;
            case 500: msg = 'Turnike sistemi sunucu hatası.'; break;
            default: msg = `Turnike işlemi tamamlanamadı (${res.status}).`;
          }
        }
        showToast(`Turnike geçiş reddedildi (${res.status}): ${msg}`, 'error');
      }
    } catch (err) {
      console.error('Turnike tap hatası:', err);
      showToast('Turnike bağlantı hatası: Sunucuya ulaşılamadı. Lütfen ağ bağlantınızı kontrol ediniz.', 'error');
    } finally {
      setTapLoading(false);
    }
  };

  // Inspector: Query Card Action (Section 9.2)
  const handleQueryCard = async (cardId) => {
    const requestID = ++queryCardRequestRef.current;
    setQueryCardLoading(true);
    setQueryCardError(null);
    try {
      const token = getActiveToken();
      const res = await api.getCard(cardId, token);
      if (requestID !== queryCardRequestRef.current) return;
      setQueriedCard(res);
      showToast(`Kart başarıyla doğrulandı: ${cardId}`, 'success');
    } catch (err) {
      if (requestID !== queryCardRequestRef.current) return;
      setQueryCardError(err.message || 'Kart bulunamadı veya sorgulama başarısız.');
      setQueriedCard(null);
      showToast('Kart sorgulama başarısız.', 'error');
    } finally {
      if (requestID === queryCardRequestRef.current) setQueryCardLoading(false);
    }
  };

  // Inspector: Create Penalty Action (Section 9.3)
  const handleCreatePenalty = async (cardId = 'CARD_TAM_001', amountKurus = 20000, reason = 'kartsiz_binis') => {
    setPenaltyLoading(true);
    try {
      const token = getActiveToken();
      const headers = { 'Content-Type': 'application/json' };
      if (token) headers['Authorization'] = `Bearer ${token}`;

      const res = await fetch('/api/v1/penalties', {
        method: 'POST',
        headers,
        body: JSON.stringify({
          card_id: cardId,
          penalty_amount_kurus: amountKurus,
          reason: reason,
          validator_id: 'VAL_500T_01',
        }),
      });

      if (res.ok) {
        const data = await res.json().catch(() => null);
        const newPenalty = data || {
          id: `PEN_${Date.now()}`,
          card_id: cardId,
          penalty_amount_kurus: amountKurus,
          reason: reason,
          issued_at: new Date().toISOString(),
        };
        setPenalties((prev) => [newPenalty, ...prev]);
        showToast(`${(amountKurus / 100).toFixed(2)} TL İdari Para Cezası Tutanağı Sisteme Kaydedildi.`, 'success');
      } else {
        showToast('Ceza tutanağı kaydedilemedi.', 'error');
      }
    } catch {
      showToast('Ceza tutanağı sunucu hatası.', 'error');
    } finally {
      setPenaltyLoading(false);
    }
  };

  // Line Manager: Headway Scenario Simulation (Section 10.3)
  const handleSimulateScenario = async (params) => {
    setSimulatingScenario(true);
    try {
      const token = getActiveToken();
      const res = await api.simulateHeadway(params, token);
      setOptimizationResult(res);
      showToast('Headway optimizasyon simülasyonu tamamlandı.', 'success');
    } catch (err) {
      showToast(`Simülasyon hatası: ${err.message}`, 'error');
    } finally {
      setSimulatingScenario(false);
    }
  };

  // Line Manager: Approve Optimization Scenario (Section 10.4)
  const handleApproveScenario = async (params) => {
    setApprovingScenario(true);
    try {
      const token = getActiveToken();
      const idempotencyKey = `OPT_${params?.line_id || 'LINE'}_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;
      await api.approveOptimization(params, token, idempotencyKey);
      showToast('Optimizasyon kararı başarıyla onaylandı ve mühürlendi.', 'success');
      setOptimizationResult(null);
      fetchLineFinancials();
    } catch (err) {
      showToast(`Onay hatası: ${err.message}`, 'error');
    } finally {
      setApprovingScenario(false);
    }
  };

  // Accounting: Settlement Close action (Section 11)
  const handleCloseSettlement = async () => {
    setClosingPeriod(true);
    try {
      const token = getActiveToken();
      const headers = { 'Content-Type': 'application/json' };
      if (token) headers['Authorization'] = `Bearer ${token}`;

      const res = await fetch('/api/v1/settlements/close', {
        method: 'POST',
        headers,
        body: JSON.stringify({
          period_id: '00000000-0000-0000-0000-000000000001',
          period_name: '2026-EKIM',
        }),
      });
      const data = await res.json().catch(() => null);

      // 200 OK veya 409 ALREADY_CLOSED: Her iki durumda da dönem kapalıdır!
      if (res.ok || res.status === 409 || data?.error?.code === 'ALREADY_CLOSED') {
        setIsPeriodClosed(true);
        localStorage.setItem('settlement_period_closed', 'true');
        showToast(
          res.ok
            ? 'Ekim 2026 Gelir Paylaşım Dönemi Başarıyla Kapatıldı.'
            : 'Dönem zaten kapatılmış durumda. Kayıtlar kilitli ve denetime hazır.',
          res.ok ? 'success' : 'info'
        );
        fetchSettlements();
      } else {
        const errorMsg = data?.error?.message || data?.message || 'Dönem kapatma başarısız oldu.';
        showToast(`Dönem kapatma hatası: ${errorMsg}`, 'error');
      }
    } catch {
      showToast('Dönem kapatma isteği başarısız oldu.', 'error');
    } finally {
      setClosingPeriod(false);
    }
  };

  // Accounting: Reopen 2026-EKIM Period action (Section 11.4)
  const handleReopenPeriod = async (periodName = '2026-EKIM', reason = 'Yetkili denetim yeniden açma') => {
    try {
      const token = getActiveToken();
      await api.reopenSettlementPeriod(periodName, reason, token);
      setIsPeriodClosed(false);
      localStorage.setItem('settlement_period_closed', 'false');
      showToast('2026-EKIM Dönemi başarıyla tekrar açıldı.', 'success');
      fetchSettlements();
    } catch (err) {
      showToast(`Dönem yeniden açma hatası: ${err.message}`, 'error');
    }
  };

  // Accounting: Storno correction action (Section 11.5)
  const handleStorno = async (referenceId = 'REF_REC_500T_001', reason = 'Hak ediş storno düzeltme kaydı', amountKurus = -25000) => {
    try {
      const token = getActiveToken();
      const idempotencyKey = `STORNO_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;
      const headers = {
        'Content-Type': 'application/json',
        'Idempotency-Key': idempotencyKey,
      };
      if (token) headers['Authorization'] = `Bearer ${token}`;

      const res = await fetch('/api/v1/settlements/storno', {
        method: 'POST',
        headers,
        body: JSON.stringify({
          period_id: '00000000-0000-0000-0000-000000000001',
          operator: 'ozel_halk',
          amount_kurus: amountKurus,
          reason: reason,
          reference_id: referenceId,
        }),
      });

      const data = await res.json().catch(() => null);

      if (res.ok) {
        showToast('Kapanış Sonrası Storno (Ters Kayıt) Düzeltmesi Eklendi.', 'success');
        fetchSettlements();
      } else {
        const errorMsg = data?.error?.message || data?.message || 'Storno kaydı reddedildi.';
        showToast(`Storno hatası: ${errorMsg}`, 'error');
      }
    } catch {
      showToast('Storno servisine ulaşılamadı.', 'error');
    }
  };

  const handleAddSubsidyAdjustment = async (adjustment, idempotencyKey) => {
    setAddingSubsidyAdjustment(true);
    try {
      await api.addSettlementSubsidy(adjustment, getActiveToken(), idempotencyKey);
      showToast('Settlement destek adjustment kaydedildi.', 'success');
      await fetchSettlements();
      return true;
    } catch (err) {
      showToast(`Destek adjustment kaydedilemedi: ${err.message}`, 'error');
      return false;
    } finally {
      setAddingSubsidyAdjustment(false);
    }
  };

  // Login handler
  const handleLoginSuccess = (token, user) => {
    cardRefreshRequestRef.current += 1;
    setCard(null);
    setLedger([]);
    setCardDataError('');
    setAuthToken(token);
    setCurrentUser(user);
    if (typeof window !== 'undefined') {
      localStorage.setItem('auth_token', token);
      localStorage.setItem('auth_user', JSON.stringify(user));
      localStorage.removeItem('logged_out');
    }
    showToast(`Hoş geldiniz, ${user.full_name || user.email}!`, 'success');

    // Switch view to user role
    const roleMapping = {
      yolcu: 'yolcu',
      kontrolor: 'kontrolor',
      hat_yoneticisi: 'hat_yoneticisi',
      muhasebe: 'muhasebe',
      operasyon: 'operasyon',
      admin: 'admin',
    };
    if (user.role && roleMapping[user.role]) {
      setActiveRole(roleMapping[user.role]);
    }
  };

  const handleQuickLoginMuhasebe = async () => {
    try {
      const data = await api.login('muhasebe@aktarbilet.gov.tr', 'MuhasebePassword123!');
      if (data && data.access_token) {
        handleLoginSuccess(data.access_token, data.user);
      }
    } catch (err) {
      showToast(`Giriş başarısız: ${err.message}`, 'error');
    }
  };

  // Logout handler
  const handleLogout = () => {
    cardRefreshRequestRef.current += 1;
    setCard(null);
    setLedger([]);
    setCardDataError('');
    setAuthToken('');
    setCurrentUser(null);
    if (typeof window !== 'undefined') {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_user');
      localStorage.removeItem('auth_token_kontrolor');
      localStorage.removeItem('auth_token_hat_yoneticisi');
      localStorage.removeItem('auth_token_muhasebe');
      localStorage.removeItem('auth_token_operasyon');
      localStorage.removeItem('auth_token_admin');
      localStorage.setItem('logged_out', 'true');
    }
    setActiveRole('home');
    showToast('Oturumunuz güvenli bir şekilde kapatıldı.', 'info');
  };

  return (
    <div className="app-layout">
      {/* Toast Notification */}
      {toast && (
        <Toast
          message={toast.message}
          type={toast.type}
          onClose={() => setToast(null)}
        />
      )}

      {/* Corporate Header & Navbar */}
      <Navbar
        currentUser={currentUser}
        onOpenLogin={() => setShowLoginModal(true)}
        onLogout={handleLogout}
        onOpenLegal={(t) => setActiveLegalModal(t)}
        onNavigateHome={() => setActiveRole('home')}
        onOpenMap={() => setActiveRole('map')}
        loginButtonRef={loginButtonRef}
        activeRole={activeRole}
        onRoleSelect={handleProtectedRoleSelect}
        theme={theme}
        onToggleTheme={toggleTheme}
      />

      {/* Main View Portals */}
      <main className="main-content" id="main-content">
        <div className="content-container">
          <React.Suspense fallback={<div className="portal-container p-8 text-center text-muted font-medium" role="status" aria-live="polite">Modül yükleniyor...</div>}>
            {/* Public Home Page */}
            {activeRole === 'home' && (
              <HomePage
                onOpenLogin={() => setShowLoginModal(true)}
                onOpenMap={() => setActiveRole('map')}
                onNavigatePortal={handleProtectedRoleSelect}
                card={card}
                onRecharge={handleRecharge}
                onTapTurnstile={handleTapTurnstile}
                rechargeLoading={rechargeLoading}
                tapLoading={tapLoading}
                heatmapError={heatmapError}
                onFetchHeatmap={fetchHeatmap}
              />
            )}

            {/* Passenger Portal */}
            {activeRole === 'yolcu' && (
              <PassengerPortal
                card={card}
                ledger={ledger}
                transferTimer={transferTimer}
                transferLevel={transferLevel}
                onRecharge={handleRecharge}
                onTapTurnstile={handleTapTurnstile}
                rechargeLoading={rechargeLoading}
                tapLoading={tapLoading}
                cardDataLoading={cardDataLoading}
                cardDataError={cardDataError}
                onShowToast={showToast}
              />
            )}

            {/* Inspector Portal */}
            {activeRole === 'kontrolor' && (
              <InspectorPortal
                onQueryCard={handleQueryCard}
                queriedCard={queriedCard}
                queryLoading={queryCardLoading}
                queryError={queryCardError}
                onCreatePenalty={handleCreatePenalty}
                penaltyLoading={penaltyLoading}
                penalties={penalties}
                token={authToken}
                onShowToast={showToast}
              />
            )}

            {/* Line Manager & Headway Optimization Portal */}
            {activeRole === 'hat_yoneticisi' && (
              <LineManagerPortal
                lineFinancials={lineFinancials}
                onSimulateScenario={handleSimulateScenario}
                onApproveScenario={handleApproveScenario}
                optimizationResult={optimizationResult}
                loadingFinancials={loadingLineFinancials}
                simulating={simulatingScenario}
                approving={approvingScenario}
              />
            )}

            {/* Accounting & Settlement Portal */}
            {activeRole === 'muhasebe' && (
              <AccountingPortal
                token={authToken}
                isPeriodClosed={isPeriodClosed}
                onClosePeriod={handleCloseSettlement}
                onReopenPeriod={handleReopenPeriod}
                onStorno={handleStorno}
                onAddSubsidy={handleAddSubsidyAdjustment}
                addingSubsidy={addingSubsidyAdjustment}
                closingPeriod={closingPeriod}
                settlementData={settlementData}
                loading={loadingSettlements}
                onShowToast={showToast}
                onFetchSettlements={fetchSettlements}
                onLoginAsMuhasebe={handleQuickLoginMuhasebe}
              />
            )}

            {/* Operations & Transit Map Portal */}
            {activeRole === 'operasyon' && (
              <OperationsPortal
                liveBuses={liveBuses}
                selectedLine={selectedTransitLine}
                onSelectLine={(line) => {
                  setSelectedTransitLine(line);
                  fetchLiveBuses(line);
                }}
                liveTransitLoading={liveTransitLoading}
                liveTransitError={liveTransitError}
                onRefreshTransit={() => fetchLiveBuses(selectedTransitLine)}
                liveTransitMeta={liveTransitMeta}
              />
            )}

            {/* Dedicated Live ESHOT Map View */}
            {activeRole === 'map' && (
              <OperationsPortal
                liveBuses={liveBuses}
                selectedLine={selectedTransitLine}
                onSelectLine={(line) => {
                  setSelectedTransitLine(line);
                  fetchLiveBuses(line);
                }}
                liveTransitLoading={liveTransitLoading}
                liveTransitError={liveTransitError}
                onRefreshTransit={() => fetchLiveBuses(selectedTransitLine)}
                liveTransitMeta={liveTransitMeta}
              />
            )}

            {/* Dedicated Heatmap View */}
            {activeRole === 'heatmap' && (
              <div className="portal-container heatmap-view">
                <div className="portal-header">
                  <div>
                    <h2 className="portal-title">Hat Bazlı Doluluk Isı Haritası</h2>
                    <p className="portal-subtitle">İzmir geneli saatlik yolcu biniş yoğunluğu ve kalabalık dağılımı</p>
                  </div>
                  <div className="portal-actions">
                    <Button variant="secondary" onClick={fetchHeatmap} loading={heatmapLoading} disabled={heatmapLoading}>
                      Isı Haritasını Yenile
                    </Button>
                  </div>
                </div>

                <Card
                  title="İstasyon ve Hat Yoğunluk Göstergesi"
                  subtitle="Canlı validatör telemetrisiyle tespit edilen yolcu biniş yoğunluğu"
                >
                  {heatmapError ? (
                    <div role="alert" className="p-6 bg-surface-muted border border-danger/30 rounded-lg text-center space-y-2">
                      <span className="text-3xl block" aria-hidden="true">⚠️</span>
                      <h4 className="font-semibold text-danger">Isı Haritası Verisi Alınamadı</h4>
                      <p className="text-sm text-muted">Sunucu bağlantısı sağlanamadı veya yoğunluk servisi yanıt vermedi.</p>
                      <Button variant="outline" size="sm" onClick={fetchHeatmap}>
                        Tekrar Dene
                      </Button>
                    </div>
                  ) : heatmapData.length > 0 ? (
                    <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
                      {heatmapData.map((item, idx) => (
                        <div key={idx} className="p-4 bg-surface-muted rounded-lg border border-border">
                          <span className="text-xs text-muted block">{item.line_name || item.line_id || item.station_name || item.station_id}</span>
                          <span className="font-bold text-xl text-primary">{item.tap_count} Biniş</span>
                          <span className="text-xs text-muted block mt-1">Yoğunluk: {item.crowd_level || item.density_level || 'Yüksek'}</span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="text-center py-8 text-muted" role={heatmapLoading ? 'status' : undefined}>
                      <p className="text-sm">{heatmapLoading ? 'Yoğunluk verisi yükleniyor...' : 'Bu dönem için yoğunluk kaydı bulunmuyor.'}</p>
                    </div>
                  )}
                  {heatmapUpdatedAt && !heatmapError && (
                    <p className="mt-4 text-xs text-muted" role="status">
                      Son güncelleme: {formatDateTime(heatmapUpdatedAt)}
                    </p>
                  )}
                </Card>
              </div>
            )}

            {/* Admin Portal (Section 4.2: Admin has role preview) */}
            {activeRole === 'admin' && (
              <AdminPortal
                token={authToken}
                onShowToast={showToast}
                onRolePreview={(previewRole) => setActiveRole(previewRole)}
                activeRole={activeRole}
              />
            )}
          </React.Suspense>
        </div>
      </main>

      {/* Accessible Corporate Footer */}
      <Footer onOpenLegal={(t) => setActiveLegalModal(t)} />

      {/* Modals */}
      <LoginModal
        isOpen={showLoginModal}
        onClose={() => setShowLoginModal(false)}
        onLoginSuccess={handleLoginSuccess}
        triggerRef={loginButtonRef}
      />

      <LegalModal
        isOpen={!!activeLegalModal}
        onClose={() => setActiveLegalModal(null)}
        topic={activeLegalModal}
      />
    </div>
  );
}
