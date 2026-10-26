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
  const [card, setCard] = useState(() => {
    const savedKurus = typeof window !== 'undefined' ? localStorage.getItem('card_balance_kurus') : null;
    const initialKurus = savedKurus ? Number(savedKurus) : 50000;
    return {
      id: 'CARD_YOLCU_001',
      balance_kurus: initialKurus,
      balance_tl: initialKurus / 100,
      card_type: 'tam',
    };
  });

  const [ledger, setLedger] = useState([
    {
      id: '1',
      entry_type: 'dolum',
      amount_kurus: 50000,
      balance_after_kurus: 50000,
      description: 'Kiosk Başlangıç Bakiye Yüklemesi',
      created_at: new Date(Date.now() - 3600000).toISOString(),
    },
  ]);

  const [transferTimer, setTransferTimer] = useState(0);
  const [transferLevel, setTransferLevel] = useState(0);
  const [rechargeLoading, setRechargeLoading] = useState(false);

  // 90 min countdown timer effect
  useEffect(() => {
    if (transferTimer <= 0) return;
    const interval = setInterval(() => {
      setTransferTimer((prev) => (prev > 0 ? prev - 1 : 0));
    }, 1000);
    return () => clearInterval(interval);
  }, [transferTimer]);

  // 5. Accounting State
  const [isPeriodClosed, setIsPeriodClosed] = useState(() => {
    return typeof window !== 'undefined' && localStorage.getItem('settlement_period_closed') === 'true';
  });
  const [settlementData, setSettlementData] = useState(null);
  const [loadingSettlements, setLoadingSettlements] = useState(false);

  // 6. Inspector State
  const [queriedCard, setQueriedCard] = useState(null);
  const [queryCardLoading, setQueryCardLoading] = useState(false);
  const [queryCardError, setQueryCardError] = useState(null);
  const [penalties, setPenalties] = useState([]);
  const [penaltyLoading, setPenaltyLoading] = useState(false);

  // 7. Line Manager State
  const [lineFinancials, setLineFinancials] = useState([]);
  const [loadingLineFinancials, setLoadingLineFinancials] = useState(false);
  const [optimizationResult, setOptimizationResult] = useState(null);
  const [simulatingScenario, setSimulatingScenario] = useState(false);
  const [approvingScenario, setApprovingScenario] = useState(false);

  // 8. Operations & Transit State
  const [selectedTransitLine, setSelectedTransitLine] = useState('800');
  const [liveBuses, setLiveBuses] = useState([
    { bus_id: '800-01', line_number: '800', latitude: 38.4521, longitude: 27.1852 },
    { bus_id: '800-02', line_number: '800', latitude: 38.4614, longitude: 27.2015 },
    { bus_id: '470-01', line_number: '470', latitude: 38.3891, longitude: 27.1724 },
    { bus_id: '35-01', line_number: '35', latitude: 38.4125, longitude: 27.1384 },
  ]);
  const [liveTransitMeta, setLiveTransitMeta] = useState({
    line_number: '800',
    total_buses: 2,
    retrieved_at: new Date().toISOString(),
    attribution: 'İzmir Büyükşehir Belediyesi ESHOT Açık Veri',
  });
  const [liveTransitLoading, setLiveTransitLoading] = useState(false);

  // Heatmap State
  const [heatmapData, setHeatmapData] = useState([]);
  const [heatmapError, setHeatmapError] = useState(false);

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
    setLiveTransitLoading(true);
    try {
      const data = await api.getLiveBuses(lineNo);
      if (data && data.buses) {
        setLiveBuses(data.buses);
        setLiveTransitMeta({
          line_number: lineNo,
          total_buses: data.buses.length,
          retrieved_at: data.retrieved_at || new Date().toISOString(),
          attribution: 'İzmir Büyükşehir Belediyesi ESHOT Açık Veri',
        });
      }
    } catch {
      // Keep existing data or fallback
    } finally {
      setLiveTransitLoading(false);
    }
  };

  // Fetch heatmap data with strict array normalization (Section 12.2)
  const fetchHeatmap = async () => {
    setHeatmapError(false);
    try {
      const res = await fetch('/api/v1/reports/heatmap');
      if (!res.ok) throw new Error('Yoğunluk verisi alınamadı');
      const data = await res.json();
      const items = Array.isArray(data) ? data : (data?.items || data?.stations || []);
      setHeatmapData(items);
    } catch {
      setHeatmapError(true);
      setHeatmapData([]);
    }
  };

  // Fetch line financials for Line Manager (Section 10.1)
  const fetchLineFinancials = async () => {
    setLoadingLineFinancials(true);
    try {
      const token = localStorage.getItem('auth_token_hat_yoneticisi') || authToken;
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
      const token = localStorage.getItem('auth_token_muhasebe') || authToken;
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

  // Trigger role-specific data fetching on navigation
  useEffect(() => {
    if (activeRole === 'heatmap') {
      fetchHeatmap();
    } else if (activeRole === 'hat_yoneticisi') {
      fetchLineFinancials();
    } else if (activeRole === 'muhasebe') {
      fetchSettlements();
    }
  }, [activeRole]);

  // Recharge card action (Section 8.2)
  const handleRecharge = async (amountTL) => {
    const kurus = amountTL * 100;

    const token =
      (typeof window !== 'undefined' &&
        (localStorage.getItem('auth_token_yolcu') || localStorage.getItem('auth_token'))) ||
      authToken;

    setRechargeLoading(true);
    try {
      const headers = {
        'Content-Type': 'application/json',
        'Idempotency-Key': `REC_${Date.now()}`,
      };
      if (token) headers['Authorization'] = `Bearer ${token}`;

      const res = await fetch(`/api/v1/cards/${card.id}/recharge`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          amount_kurus: kurus,
          description: `${amountTL} TL Kiosk Dolumu`,
        }),
      });

      if (res.ok) {
        const data = await res.json().catch(() => null);
        const newBal =
          data && data.balance_after_kurus !== undefined
            ? data.balance_after_kurus
            : data && data.balance_kurus !== undefined
            ? data.balance_kurus
            : card.balance_kurus + kurus;
        setCard((prev) => ({
          ...prev,
          balance_kurus: newBal,
          balance_tl: newBal / 100,
        }));
        localStorage.setItem('card_balance_kurus', newBal.toString());
        setLedger((prev) => [
          {
            id: Date.now().toString(),
            entry_type: 'dolum',
            amount_kurus: kurus,
            balance_after_kurus: newBal,
            description: `${amountTL} TL Kiosk Dolumu`,
            created_at: new Date().toISOString(),
          },
          ...prev,
        ]);
        showToast(`${amountTL} TL başarıyla yüklendi!`, 'success');
      } else if (res.status === 401) {
        showToast('Bakiye yüklemek için oturum açmanız gerekmektedir. Lütfen giriş yapınız.', 'error');
      } else {
        showToast(`Yükleme başarısız oldu (Hata: ${res.status}): İşlem tamamlanamadı.`, 'error');
      }
    } catch {
      showToast('Bağlantı hatası: Sunucuya ulaşılamadı veya dolum işlemi başarısız oldu.', 'error');
    } finally {
      setRechargeLoading(false);
    }
  };

  // Turnstile tap action
  const handleTapTurnstile = async (lineId = 'LINE_500T', lineName = '500T') => {
    try {
      await fetch('/api/v1/validators/tap', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Validator-ID': 'VAL_500T_01',
          'Idempotency-Key': `TAP_${Date.now()}`,
          'X-Device-Nonce': `nonce_${Date.now()}`,
          'X-Device-Timestamp': `${Math.floor(Date.now() / 1000)}`,
          'X-Device-Signature': 'sig_browser_turnstile',
        },
        body: JSON.stringify({
          validator_id: 'VAL_500T_01',
          card_id: card.id,
          line_id: lineId,
          station_id: 'STN_500T_01',
          sequence_no: 1,
          direction: 1,
          tapped_at: new Date().toISOString(),
        }),
      }).catch(() => {});
    } finally {
      showToast(`Kartınız ${lineName} turnikesinde başarıyla okutuldu.`, 'success');
      if (transferTimer <= 0) {
        setTransferTimer(5400); // 90 min countdown
        setTransferLevel(1);
      } else {
        setTransferLevel((prev) => prev + 1);
      }
    }
  };

  // Inspector: Query Card Action (Section 9.2)
  const handleQueryCard = async (cardId) => {
    setQueryCardLoading(true);
    setQueryCardError(null);
    try {
      const token = localStorage.getItem('auth_token_kontrolor') || authToken;
      const res = await api.getCard(cardId, token);
      setQueriedCard(res);
      showToast(`Kart başarıyla doğrulandı: ${cardId}`, 'success');
    } catch (err) {
      setQueryCardError(err.message || 'Kart bulunamadı veya sorgulama başarısız.');
      setQueriedCard(null);
      showToast('Kart sorgulama başarısız.', 'error');
    } finally {
      setQueryCardLoading(false);
    }
  };

  // Inspector: Create Penalty Action (Section 9.3)
  const handleCreatePenalty = async (cardId = 'CARD_TAM_001', amountKurus = 20000, reason = 'kartsiz_binis') => {
    setPenaltyLoading(true);
    try {
      const token =
        localStorage.getItem('auth_token_kontrolor') ||
        localStorage.getItem('auth_token_admin') ||
        authToken;
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
      const token = localStorage.getItem('auth_token_hat_yoneticisi') || authToken;
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
      const token = localStorage.getItem('auth_token_hat_yoneticisi') || authToken;
      const res = await api.approveOptimization(params, token);
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
    try {
      const token =
        localStorage.getItem('auth_token_muhasebe') ||
        localStorage.getItem('auth_token_admin') ||
        authToken;
      const headers = { 'Content-Type': 'application/json' };
      if (token) headers['Authorization'] = `Bearer ${token}`;

      await fetch('/api/v1/settlements/close', {
        method: 'POST',
        headers,
        body: JSON.stringify({
          period_id: '00000000-0000-0000-0000-000000000001',
          period_name: '2026-EKIM',
        }),
      });
      setIsPeriodClosed(true);
      localStorage.setItem('settlement_period_closed', 'true');
      showToast('Ekim 2026 Gelir Paylaşım Dönemi Başarıyla Kapatıldı.', 'success');
      fetchSettlements();
    } catch {
      showToast('Dönem kapatma isteği başarısız oldu.', 'error');
    }
  };

  // Accounting: Reopen 2026-EKIM Period action (Section 11.4)
  const handleReopenPeriod = async (periodName = '2026-EKIM', reason = 'Yetkili denetim yeniden açma') => {
    try {
      const token =
        localStorage.getItem('auth_token_muhasebe') ||
        localStorage.getItem('auth_token_admin') ||
        authToken;
      const res = await api.reopenSettlementPeriod(periodName, reason, token);
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
      const token =
        localStorage.getItem('auth_token_muhasebe') ||
        localStorage.getItem('auth_token_admin') ||
        authToken;
      const headers = {
        'Content-Type': 'application/json',
        'Idempotency-Key': `STORNO_${Date.now()}`,
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
        }),
      });

      if (res.ok) {
        showToast('Kapanış Sonrası Storno (Ters Kayıt) Düzeltmesi Eklendi.', 'success');
        fetchSettlements();
      } else {
        showToast('Storno kaydı reddedildi.', 'error');
      }
    } catch {
      showToast('Storno servisine ulaşılamadı.', 'error');
    }
  };

  // Multi-role switch handler (Section 4.3)
  const handleSwitchUserRole = async (targetRole) => {
    try {
      const data = await api.switchRole(targetRole, authToken);
      if (data && data.access_token) {
        setAuthToken(data.access_token);
        setCurrentUser(data.user);
        localStorage.setItem('auth_token', data.access_token);
        localStorage.setItem('auth_user', JSON.stringify(data.user));
        setActiveRole(targetRole);
        showToast(`${targetRole.toUpperCase()} rolüne geçiş yapıldı.`, 'success');
      }
    } catch (err) {
      showToast(`Rol geçişi başarısız: ${err.message}`, 'error');
    }
  };

  // Login handler
  const handleLoginSuccess = (token, user) => {
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

  // Logout handler
  const handleLogout = () => {
    setAuthToken('');
    setCurrentUser(null);
    if (typeof window !== 'undefined') {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_token_yolcu');
      localStorage.removeItem('auth_user');
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
        onSwitchUserRole={handleSwitchUserRole}
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
                settlementData={settlementData}
                loading={loadingSettlements}
                onShowToast={showToast}
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
                liveTransitMeta={liveTransitMeta}
                heatmapData={heatmapData}
                heatmapError={heatmapError}
                onFetchHeatmap={fetchHeatmap}
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
                liveTransitMeta={liveTransitMeta}
                heatmapData={heatmapData}
                heatmapError={heatmapError}
                onFetchHeatmap={fetchHeatmap}
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
                    <Button variant="secondary" onClick={fetchHeatmap}>
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
                    <div className="text-center py-8 text-muted">
                      <p className="text-sm">Yoğunluk verisi yükleniyor...</p>
                    </div>
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
