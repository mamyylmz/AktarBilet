import { test, expect } from '@playwright/test';
import { apiURL, bearer, login } from '../helpers/http.js';
import { scalar } from '../helpers/db.js';

const roles = ['yolcu', 'kontrolor', 'hat_yoneticisi', 'muhasebe', 'operasyon', 'admin'];

test.describe('final endpoint-by-role audit', () => {
  for (const role of roles) {
    test(`settlement close role policy: ${role}`, async ({ request }) => {
      const { token } = await login(request, role);
      const response = await request.post(`${apiURL}/api/v1/settlements/close`, {
        headers: bearer(token), data: { period_id: 'invalid' }
      });
      if (['muhasebe', 'admin'].includes(role)) expect(response.status()).toBe(400);
      else expect(response.status()).toBe(403);
    });

    test(`anomaly role policy: ${role}`, async ({ request }) => {
      const { token } = await login(request, role);
      const response = await request.get(`${apiURL}/api/v1/anomalies`, { headers: bearer(token) });
      if (['operasyon', 'admin'].includes(role)) expect(response.status()).toBe(200);
      else expect(response.status()).toBe(403);
    });

    test(`penalty create role policy: ${role}`, async ({ request }) => {
      const { token } = await login(request, role);
      const response = await request.post(`${apiURL}/api/v1/penalties`, {
        headers: bearer(token), data: {}
      });
      if (['kontrolor', 'admin'].includes(role)) expect(response.status()).toBe(400);
      else expect(response.status()).toBe(403);
    });
  }

  for (const role of ['yolcu', 'hat_yoneticisi', 'muhasebe', 'operasyon']) {
    test(`penalty records are not exposed to unrelated role: ${role}`, async ({ request }) => {
      const { token } = await login(request, role);
      const response = await request.get(`${apiURL}/api/v1/penalties`, { headers: bearer(token) });
      expect(response.status()).toBe(403);
    });
  }

  for (const role of ['yolcu', 'kontrolor', 'muhasebe', 'operasyon']) {
    test(`fare simulation is denied to unrelated role: ${role}`, async ({ request }) => {
      const { token } = await login(request, role);
      const response = await request.post(`${apiURL}/api/v1/simulation/fare-impact`, {
        headers: bearer(token), data: { base_change: 1, student_change: 1 }
      });
      expect(response.status()).toBe(403);
    });
  }

  test('an ordinary passenger JWT is not a validator device credential', async ({ request }) => {
    const { token } = await login(request, 'yolcu');
    const before = scalar("SELECT balance_kurus FROM jw_cards WHERE id='CARD_YOLCU_001'");
    const response = await request.post(`${apiURL}/api/v1/validators/tap`, {
      headers: { ...bearer(token), 'Idempotency-Key': `RBAC-USER-AS-DEVICE-${Date.now()}` },
      data: {
        validator_id: 'VAL_500T_01', sequence_no: Date.now(), card_id: 'CARD_YOLCU_001',
        line_id: 'LINE_500T', station_id: 'STAT_AYRILIK', direction: 1,
        tapped_at: new Date().toISOString()
      }
    });
    expect.soft([401, 403]).toContain(response.status());
    expect(scalar("SELECT balance_kurus FROM jw_cards WHERE id='CARD_YOLCU_001'")).toBe(before);
  });

  test('line manager has a real server-side line management operation', async ({ request }) => {
    const { token } = await login(request, 'hat_yoneticisi');
    const response = await request.post(`${apiURL}/api/v1/lines`, {
      headers: bearer(token), data: { id: 'QA_NOT_COMMITTED', name: 'QA', mode: 'bus', operator: 'belediye' }
    });
    expect([200, 201, 400, 409, 422]).toContain(response.status());
  });
});
