import { test, expect } from '@playwright/test';

const BASE_URL = process.env.QA_FRONTEND_URL || 'http://localhost:3000';
const API_URL = process.env.QA_API_URL || 'http://localhost:3000';

test.describe('AGY Zero-Trust 7-Role E2E & Boundary Audit', () => {

  test('ROLE 1 - Anonim: Public access, login dialog access, and 401 rejection on protected endpoints', async ({ page, request }) => {
    // 1. Home page loads
    await page.goto(BASE_URL + '/');
    await expect(page.getByText('AktarBilet', { exact: true })).toBeVisible();

    // 2. Login modal opens
    const loginBtn = page.getByRole('button', { name: /Giriş/i }).first();
    await expect(loginBtn).toBeVisible();
    await loginBtn.click();
    await expect(page.getByRole('dialog', { name: /AktarBilet Giriş/i })).toBeVisible();
    await page.keyboard.press('Escape');

    // 3. Direct protected endpoints return 401
    const protectedEndpoints = [
      { method: 'get', url: `${API_URL}/api/v1/auth/me` },
      { method: 'get', url: `${API_URL}/api/v1/cards/CARD_TAM_001` },
      { method: 'get', url: `${API_URL}/api/v1/cards/CARD_TAM_001/ledger` },
      { method: 'post', url: `${API_URL}/api/v1/cards/CARD_TAM_001/recharge` },
      { method: 'get', url: `${API_URL}/api/v1/penalties` },
      { method: 'get', url: `${API_URL}/api/v1/anomalies` },
      { method: 'post', url: `${API_URL}/api/v1/settlements/close` },
    ];

    for (const ep of protectedEndpoints) {
      const res = await request[ep.method](ep.url);
      expect(res.status(), `Expected 401 for ${ep.method.toUpperCase()} ${ep.url}`).toBe(401);
      const json = await res.json();
      expect(json.error).toBeDefined();
    }
  });

  test('ROLE 2 - Yolcu: Authentication, card/balance, recharge and tariffs', async ({ request }) => {
    // 1. Login as Yolcu
    const loginRes = await request.post(`${API_URL}/api/v1/auth/login`, {
      data: { email: 'yolcu@aktarbilet.gov.tr', password: 'YolcuPassword123!' }
    });
    expect(loginRes.status()).toBe(200);
    const loginData = await loginRes.json();
    const token = loginData.access_token;
    expect(token).toBeTruthy();

    // 2. Fetch Card & Balance
    const cardRes = await request.get(`${API_URL}/api/v1/cards/CARD_YOLCU_001`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(cardRes.status()).toBe(200);
    const cardData = await cardRes.json();
    expect(cardData.card_id || cardData.id).toBe('CARD_YOLCU_001');
    const initialBalance = cardData.balance_kurus;

    // 3. Top-up / Recharge
    const rechargeRes = await request.post(`${API_URL}/api/v1/cards/CARD_YOLCU_001/recharge`, {
      headers: {
        Authorization: `Bearer ${token}`,
        'Idempotency-Key': `AUDIT_REC_${Date.now()}`
      },
      data: { amount_kurus: 5000, description: 'QA Audit Top-up' }
    });
    expect(rechargeRes.status()).toBe(200);
    const rechargeData = await rechargeRes.json();
    expect(rechargeData.balance_after_kurus).toBe(initialBalance + 5000);

    // 4. Verify ledger has new entry
    const ledgerRes = await request.get(`${API_URL}/api/v1/cards/CARD_YOLCU_001/ledger`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(ledgerRes.status()).toBe(200);
    const ledgerData = await ledgerRes.json();
    const items = Array.isArray(ledgerData) ? ledgerData : ledgerData.items;
    expect(items.length).toBeGreaterThan(0);
    expect(items.some(e => e.description === 'QA Audit Top-up')).toBe(true);

    // 5. Tariffs view
    const tariffsRes = await request.get(`${API_URL}/api/v1/tariffs`);
    expect(tariffsRes.status()).toBe(200);
    const tariffs = await tariffsRes.json();
    expect(Array.isArray(tariffs)).toBe(true);
    expect(tariffs.length).toBeGreaterThan(0);
  });

  test('ROLE 3 - Kontrolör: Inspection and penalty ticket creation', async ({ request }) => {
    // 1. Login as Kontrolör
    const loginRes = await request.post(`${API_URL}/api/v1/auth/login`, {
      data: { email: 'kontrolor@aktarbilet.gov.tr', password: 'KontrolorPassword123!' }
    });
    expect(loginRes.status()).toBe(200);
    const token = (await loginRes.json()).access_token;

    // 2. Query Card
    const cardRes = await request.get(`${API_URL}/api/v1/cards/CARD_TAM_001`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(cardRes.status()).toBe(200);

    // 3. Create Penalty Ticket (Backend returns 201 Created)
    const penaltyRes = await request.post(`${API_URL}/api/v1/penalties`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        card_id: 'CARD_TAM_001',
        penalty_amount_kurus: 20000,
        reason: 'kartsiz_binis',
        validator_id: 'VAL_500T_01'
      }
    });
    expect(penaltyRes.status()).toBe(201);
    const penaltyData = await penaltyRes.json();
    expect(penaltyData.penalty_id || penaltyData.id).toBeTruthy();

    // 4. Verify listed in penalties
    const listRes = await request.get(`${API_URL}/api/v1/penalties`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(listRes.status()).toBe(200);
    const list = await listRes.json();
    const penalties = Array.isArray(list) ? list : list.penalties;
    expect(penalties.some(p => p.card_id === 'CARD_TAM_001')).toBe(true);
  });

  test('ROLE 4 - Hat Yöneticisi: Lines, financials and headway optimization simulation', async ({ request }) => {
    // 1. Login as Hat Yöneticisi
    const loginRes = await request.post(`${API_URL}/api/v1/auth/login`, {
      data: { email: 'hatyoneticisi@aktarbilet.gov.tr', password: 'HatYoneticisi123!' }
    });
    expect(loginRes.status()).toBe(200);
    const token = (await loginRes.json()).access_token;

    // 2. Lines & Financials
    const finRes = await request.get(`${API_URL}/api/v1/reports/line-financials`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(finRes.status()).toBe(200);
    const finData = await finRes.json();
    expect(Array.isArray(finData)).toBe(true);
    expect(finData.length).toBeGreaterThan(0);

    // 3. Headway optimization simulation
    const simRes = await request.post(`${API_URL}/api/v1/optimization/headway-scenario`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        line_id: 'LINE_500T',
        current_headway_minutes: 15,
        target_headway_minutes: 8,
        fleet_capacity_per_vehicle: 75,
        peak_hour: true
      }
    });
    expect(simRes.status()).toBe(200);
    const simData = await simRes.json();
    expect(simData.vehicle_delta).toBeDefined();
    expect(simData.delta_cost_kurus).toBeDefined();
  });

  test('ROLE 5 - Muhasebe: Revenue distribution, period closing and storno ledger impact', async ({ request }) => {
    // 1. Login as Muhasebe
    const loginRes = await request.post(`${API_URL}/api/v1/auth/login`, {
      data: { email: 'muhasebe@aktarbilet.gov.tr', password: 'MuhasebePassword123!' }
    });
    expect(loginRes.status()).toBe(200);
    const token = (await loginRes.json()).access_token;

    // 2. Audit check: GET /settlements endpoint returns 200 OK with settlement summary
    const setRes = await request.get(`${API_URL}/api/v1/settlements`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    expect(setRes.status(), 'GET /settlements returns 200 OK').toBe(200);

    // 3. Settlement Close (Real API)
    const testPeriodId = `00000000-0000-4000-8000-${Date.now().toString(16).padStart(12, '0')}`;
    const closeRes = await request.post(`${API_URL}/api/v1/settlements/close`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        period_id: testPeriodId,
        period_name: `2026-EKIM-${Date.now()}`
      }
    });
    expect([200, 409]).toContain(closeRes.status());

    // 4. Storno correction (Real API)
    const stornoRes = await request.post(`${API_URL}/api/v1/settlements/storno`, {
      headers: {
        Authorization: `Bearer ${token}`,
        'Idempotency-Key': `AUDIT_STORNO_${Date.now()}`
      },
      data: {
        period_id: testPeriodId,
        operator: 'ozel_halk',
        amount_kurus: -25000,
        reason: 'QA Audit Storno Correction'
      }
    });
    expect(stornoRes.status()).toBe(200);
  });

  test('ROLE 6 - Operasyon: Live ESHOT transit telemetry and PMTiles vector source', async ({ request, page }) => {
    // 1. Live ESHOT transit buses API
    const busRes = await request.get(`${API_URL}/api/v1/transit/live-buses?line=800`);
    expect(busRes.status()).toBe(200);
    const busData = await busRes.json();
    expect(busData.line_no).toBe('800');
    expect(Array.isArray(busData.buses)).toBe(true);
    expect(busData.buses.length).toBeGreaterThan(0);
    expect(busData.attribution).toContain('ESHOT');

    // 2. PMTiles HTTP 206 Partial Content check via browser
    await page.goto(BASE_URL + '/');
    const pmtilesPromise = page.waitForResponse(res => res.url().includes('izmir.pmtiles') && (res.status() === 206 || res.status() === 200), { timeout: 10000 });
    const mapBtn = page.getByRole('button', { name: /Harita|ESHOT/i }).first();
    await mapBtn.click();
    const pmtilesResponse = await pmtilesPromise;
    expect(pmtilesResponse.headers()['accept-ranges']).toBe('bytes');
  });

  test('ROLE 7 - Admin & RBAC Boundary: Unauthorized passenger cannot access administrative endpoints', async ({ request }) => {
    // 1. Yolcu token
    const yolcuLogin = await request.post(`${API_URL}/api/v1/auth/login`, {
      data: { email: 'yolcu@aktarbilet.gov.tr', password: 'YolcuPassword123!' }
    });
    const yolcuToken = (await yolcuLogin.json()).access_token;

    // 2. Yolcu attempting to access Admin / Privileged endpoints -> MUST return 403 Forbidden
    const forbiddenAttempts = [
      { method: 'post', url: `${API_URL}/api/v1/penalties`, data: { card_id: 'CARD_TAM_001', penalty_amount_kurus: 1000 } },
      { method: 'get', url: `${API_URL}/api/v1/anomalies` },
      { method: 'post', url: `${API_URL}/api/v1/settlements/close`, data: { period_name: 'TEST' } },
      { method: 'post', url: `${API_URL}/api/v1/optimization/approve`, data: { line_id: 'LINE_500T' } }
    ];

    for (const attempt of forbiddenAttempts) {
      const res = await request[attempt.method](attempt.url, {
        headers: { Authorization: `Bearer ${yolcuToken}` },
        data: attempt.data
      });
      expect(res.status(), `Yolcu token must be 403 Forbidden for ${attempt.url}`).toBe(403);
    }

    // 3. Admin token CAN access
    const adminLogin = await request.post(`${API_URL}/api/v1/auth/login`, {
      data: { email: 'seed.sedir@ddo-case.invalid', password: 'SedirPassword2026!' }
    });
    expect(adminLogin.status()).toBe(200);
    const adminToken = (await adminLogin.json()).access_token;

    const adminAnomalyRes = await request.get(`${API_URL}/api/v1/anomalies`, {
      headers: { Authorization: `Bearer ${adminToken}` }
    });
    expect(adminAnomalyRes.status()).toBe(200);
  });
});
