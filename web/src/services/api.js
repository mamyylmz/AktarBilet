// AktarBilet Centralized REST API Service Client

const BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

/**
 * Perform an authenticated HTTP request with JSON parsing and standardized error handling
 */
async function request(endpoint, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  };

  const token = options.token || (typeof window !== 'undefined' ? localStorage.getItem('auth_token') : null);
  if (token && !headers['Authorization']) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const url = `${BASE_URL}${endpoint}`;
  const response = await fetch(url, {
    ...options,
    headers,
  });

  const contentType = response.headers.get('content-type') || '';
  let data = null;
  if (contentType.includes('application/json')) {
    data = await response.json().catch(() => null);
  }

  if (!response.ok) {
    const errorMsg = data?.error?.message || data?.message || `İşlem başarısız oldu (HTTP ${response.status})`;
    const error = new Error(errorMsg);
    error.status = response.status;
    error.code = data?.error?.code || 'API_ERROR';
    error.data = data;
    throw error;
  }

  return data;
}

export const api = {
  // 1. Kimlik Doğrulama
  login: async (email, password) => {
    return request('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
  },

  switchRole: async (role, token) => {
    return request('/api/v1/auth/switch-role', {
      method: 'POST',
      token,
      body: JSON.stringify({ role }),
    });
  },

  getCurrentUser: async (token) => {
    return request('/api/v1/auth/me', { method: 'GET', token });
  },

  // 2. Kart ve Defter (Yolcu)
  getCard: async (cardId, token) => {
    return request(`/api/v1/cards/${cardId}`, { method: 'GET', token });
  },

  getCardSnapshot: async (cardId, token) => {
    return request(`/api/v1/cards/${cardId}/snapshot`, { method: 'GET', token });
  },

  rechargeCard: async (cardId, amountKurus, description, token, idempotencyKey) => {
    return request(`/api/v1/cards/${cardId}/recharge`, {
      method: 'POST',
      token,
      headers: {
        'Idempotency-Key': idempotencyKey || `REC_${Date.now()}_${Math.random().toString(16).slice(2)}`,
      },
      body: JSON.stringify({ amount_kurus: amountKurus, description: description || 'Kiosk Dolumu' }),
    });
  },

  getLedger: async (cardId, token) => {
    return request(`/api/v1/cards/${cardId}/ledger`, { method: 'GET', token });
  },

  // 3. Turnike ve Validatör İşlemleri
  tapTurnstile: async (payload, headers = {}) => {
    return request('/api/v1/validators/tap', {
      method: 'POST',
      headers,
      body: JSON.stringify(payload),
    });
  },

  // 4. Kontrolör Ceza Tutanakları
  createPenalty: async (cardId, amountKurus, reason, token) => {
    return request('/api/v1/penalties', {
      method: 'POST',
      token,
      body: JSON.stringify({
        card_id: cardId,
        penalty_amount_kurus: amountKurus,
        reason: reason || 'kartsiz_binis',
      }),
    });
  },

  getPenalties: async (token) => {
    return request('/api/v1/penalties', { method: 'GET', token });
  },

  // 5. Muhasebe ve Dönem Kapanışı
  getSettlements: async (token) => {
    return request('/api/v1/settlements', { method: 'GET', token });
  },

  getAdminUsers: async (token) => request('/api/v1/admin/users', { method: 'GET', token }),

  closeSettlementPeriod: async (periodName, token) => {
    return request('/api/v1/settlements/close', {
      method: 'POST',
      token,
      body: JSON.stringify({ period_name: periodName || '2026-EKIM', period_id: '00000000-0000-0000-0000-000000000001' }),
    });
  },

  reopenSettlementPeriod: async (periodName, reason, token) => {
    return request('/api/v1/settlements/reopen', {
      method: 'POST',
      token,
      body: JSON.stringify({
        period_name: periodName || '2026-EKIM',
        reason: reason || 'Yetkili denetim yeniden açma',
      }),
    });
  },

  stornoSettlement: async (referenceId, reason, amountKurus = -25000, token, idempotencyKey) => {
    const key = idempotencyKey || `STORNO_${Date.now()}_${Math.random().toString(16).slice(2)}`;
    return request('/api/v1/settlements/storno', {
      method: 'POST',
      token,
      headers: {
        'Idempotency-Key': key,
      },
      body: JSON.stringify({
        period_id: '00000000-0000-0000-0000-000000000001',
        operator: 'ozel_halk',
        amount_kurus: amountKurus,
        reason: reason || 'Hak ediş storno düzeltme kaydı',
        reference_id: referenceId || 'REF_REC_500T_001',
      }),
    });
  },

  addSettlementSubsidy: async (adjustment, token, idempotencyKey) => request('/api/v1/settlements/adjustments/subsidy', {
    method: 'POST',
    token,
    headers: { 'Idempotency-Key': idempotencyKey },
    body: JSON.stringify(adjustment),
  }),

  // 6. Hat Yöneticisi & Optimizasyon
  getLineFinancials: async (token) => {
    return request('/api/v1/reports/line-financials', { method: 'GET', token });
  },

  simulateHeadway: async (params, token) => {
    return request('/api/v1/optimization/headway-scenario', {
      method: 'POST',
      token,
      body: JSON.stringify(params),
    });
  },

  approveOptimization: async (params, token, idempotencyKey) => {
    const key = idempotencyKey || `OPT_${params?.line_id || 'LINE'}_${Date.now()}_${Math.random().toString(16).slice(2)}`;
    return request('/api/v1/optimization/approve', {
      method: 'POST',
      token,
      headers: {
        'Idempotency-Key': key,
      },
      body: JSON.stringify(params),
    });
  },

  // 7. Operasyon & Harita
  getLiveBuses: async (lineNo) => {
    return request(`/api/v1/transit/live-buses?line=${encodeURIComponent(lineNo || '800')}`, { method: 'GET' });
  },

  getHeatmap: async () => {
    return request('/api/v1/reports/heatmap', { method: 'GET' });
  },

  getAnomalies: async (token) => {
    return request('/api/v1/anomalies', { method: 'GET', token });
  },

  // 8. Sistem Sağlığı
  getHealth: async () => {
    return request('/health', { method: 'GET' });
  },
};

