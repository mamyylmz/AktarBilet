import { test, expect } from '@playwright/test';
import { apiURL, bearer, login } from '../helpers/http.js';
import { signedToken } from '../helpers/jwt.js';
import { scalar, sql } from '../helpers/db.js';

test.describe('RBAC, ownership and token security', () => {
  const anonymousDenials = [
    ['GET', '/api/v1/auth/me'],
    ['GET', '/api/v1/cards/CARD_TAM_001'],
    ['GET', '/api/v1/cards/CARD_TAM_001/ledger'],
    ['POST', '/api/v1/cards/CARD_TAM_001/recharge'],
    ['POST', '/api/v1/validators/tap'],
    ['POST', '/api/v1/validators/batch-sync'],
    ['GET', '/api/v1/penalties'],
    ['GET', '/api/v1/anomalies'],
    ['POST', '/api/v1/settlements/close'],
    ['POST', '/api/v1/settlements/storno'],
    ['POST', '/api/v1/simulation/fare-impact']
  ];

  for (const [method, endpoint] of anonymousDenials) {
    test(`anonymous ${method} ${endpoint} is denied`, async ({ request }) => {
      const response = await request.fetch(`${apiURL}${endpoint}`, {
        method,
        data: method === 'POST' ? {} : undefined
      });
      expect([401, 403]).toContain(response.status());
    });
  }

  test('tampered and expired JWTs are rejected', async ({ request }) => {
    const { token } = await login(request, 'operasyon');
    const tokenParts = token.split('.');
    const signature = Buffer.from(tokenParts[2], 'base64url');
    signature[0] ^= 0xff;
    tokenParts[2] = signature.toString('base64url');
    const tampered = tokenParts.join('.');
    const tamperedResponse = await request.get(`${apiURL}/api/v1/anomalies`, { headers: bearer(tampered) });
    expect(tamperedResponse.status()).toBe(401);

    const expired = signedToken({
      user_id: '00000000-0000-0000-0000-000000000001', role: 'operasyon',
      exp: Math.floor(Date.now() / 1000) - 60
    });
    const expiredResponse = await request.get(`${apiURL}/api/v1/anomalies`, { headers: bearer(expired) });
    expect(expiredResponse.status()).toBe(401);
  });

  test('JWT validator rejects HMAC algorithms other than the issued HS256', async ({ request }) => {
    const token = signedToken({
      user_id: '00000000-0000-0000-0000-000000000001', email: 'qa@invalid',
      role: 'admin', exp: Math.floor(Date.now() / 1000) + 300
    }, 'HS512');
    const response = await request.get(`${apiURL}/api/v1/anomalies`, { headers: bearer(token) });
    expect(response.status()).toBe(401);
  });

  test('passenger cannot read or mutate a card that is not owned by that passenger', async ({ request }) => {
    const { token, body } = await login(request, 'yolcu');
    expect(scalar(`SELECT COUNT(*) FROM jw_cards WHERE id='CARD_TAM_001' AND user_id='${body.user.id}'`)).toBe('0');
    const read = await request.get(`${apiURL}/api/v1/cards/CARD_TAM_001`, { headers: bearer(token) });
    const write = await request.post(`${apiURL}/api/v1/cards/CARD_TAM_001/recharge`, {
      headers: { ...bearer(token), 'Idempotency-Key': 'QA-IDOR-001' },
      data: { amount_kurus: 1, description: 'IDOR probe' }
    });
    expect([403, 404]).toContain(read.status());
    expect([403, 404]).toContain(write.status());
  });

  test('role change revokes authorization carried by an old access token', async ({ request }) => {
    const { token, body } = await login(request, 'operasyon');
    sql(`UPDATE jw_tf_users SET role='yolcu' WHERE id='${body.user.id}'`);
    try {
      const response = await request.get(`${apiURL}/api/v1/anomalies`, { headers: bearer(token) });
      expect(response.status()).toBe(403);
    } finally {
      sql(`UPDATE jw_tf_users SET role='operasyon' WHERE id='${body.user.id}'`);
    }
  });

  test('only controller/admin may create penalties', async ({ request }) => {
    for (const role of ['yolcu', 'hat_yoneticisi', 'muhasebe', 'operasyon']) {
      const { token } = await login(request, role);
      const response = await request.post(`${apiURL}/api/v1/penalties`, {
        headers: bearer(token),
        data: { card_id: 'CARD_TAM_001', penalty_amount_kurus: 5000, reason: 'kartsiz_binis' }
      });
      expect(response.status(), `${role} unexpectedly reached penalty creation`).toBe(403);
    }
  });

  test('wildcard CORS is not combined with credentialed cross-origin access', async ({ request }) => {
    const response = await request.fetch(`${apiURL}/api/v1/cards/CARD_TAM_001`, {
      method: 'OPTIONS',
      headers: {
        Origin: 'https://attacker.invalid',
        'Access-Control-Request-Method': 'GET',
        'Access-Control-Request-Headers': 'authorization'
      }
    });
    expect(response.headers()['access-control-allow-origin']).not.toBe('https://attacker.invalid');
    expect(response.headers()['access-control-allow-credentials']).not.toBe('true');
  });
});
