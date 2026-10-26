import { test, expect } from '@playwright/test';
import { apiURL, accounts, bearer, login, deviceAuthHeaders } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

test.describe('real API and PostgreSQL integration', () => {
  test('@smoke health proves PostgreSQL readiness and canary headers', async ({ request }) => {
    const response = await request.get(`${apiURL}/health`);
    expect(response.status()).toBe(200);
    expect(response.headers()['x-ddo-stamp']).toBe('f85f26');
    const body = await response.json();
    expect(body.status).toBe('UP');
    expect(body.database).toBe('healthy');
    expect(scalar('SELECT 1')).toBe('1');
  });

  test('@smoke mandatory health canary is present', async ({ request }) => {
    const response = await request.get(`${apiURL}/_hz/f85f`);
    expect(response.status()).toBe(200);
    expect(response.headers()['x-ddo-stamp']).toBe('f85f26');
    expect(await response.json()).toMatchObject({ status: 'ok', stamp: 'f85f26' });
  });

  for (const role of Object.keys(accounts)) {
    test(`real seeded ${role} account can log in with its database role`, async ({ request }) => {
      const { response, body } = await login(request, role);
      expect(response.status()).toBe(200);
      expect(body.access_token).toBeTruthy();
      expect(body.user.role).toBe(role);
      expect(scalar(`SELECT role FROM jw_tf_users WHERE email = '${accounts[role][0]}'`)).toBe(role);
    });
  }

  test('wrong password and unknown user are rejected without token leakage', async ({ request }) => {
    const wrong = await request.post(`${apiURL}/api/v1/auth/login`, {
      data: { email: accounts.yolcu[0], password: 'definitely-wrong' }
    });
    expect(wrong.status()).toBe(401);
    expect(await wrong.json()).not.toHaveProperty('access_token');

    const unknown = await request.post(`${apiURL}/api/v1/auth/login`, {
      data: { email: 'unknown@qa.invalid', password: 'irrelevant' }
    });
    expect(unknown.status()).toBe(401);
    expect(await unknown.json()).not.toHaveProperty('access_token');
  });

  test('authenticated identity endpoint returns the logged-in role', async ({ request }) => {
    const { token } = await login(request, 'yolcu');
    const response = await request.get(`${apiURL}/api/v1/auth/me`, { headers: bearer(token) });
    expect(response.status()).toBe(200);
    expect((await response.json()).role).toBe('yolcu');
  });

  test('anonymous recharge is denied and leaves card and ledger unchanged', async ({ request }) => {
    const beforeBalance = scalar("SELECT balance_kurus FROM jw_cards WHERE id='CARD_TAM_001'");
    const beforeCount = scalar("SELECT COUNT(*) FROM jw_ledger WHERE card_id='CARD_TAM_001'");
    const response = await request.post(`${apiURL}/api/v1/cards/CARD_TAM_001/recharge`, {
      headers: { 'Idempotency-Key': 'QA-ANON-RECHARGE' },
      data: { amount_kurus: 5000, description: 'anonymous QA attempt' }
    });
    expect(response.status()).toBe(401);
    expect(scalar("SELECT balance_kurus FROM jw_cards WHERE id='CARD_TAM_001'")).toBe(beforeBalance);
    expect(scalar("SELECT COUNT(*) FROM jw_ledger WHERE card_id='CARD_TAM_001'")).toBe(beforeCount);
  });

  test('recharge idempotency returns one effect for the same key', async ({ request }) => {
    const { token, body } = await login(request, 'yolcu');
    sql(`UPDATE jw_cards SET user_id='${body.user.id}' WHERE id='CARD_TAM_001'`);
    try {
      const key = 'QA-RECHARGE-IDEMPOTENCY-001';
      const before = Number(scalar("SELECT balance_kurus FROM jw_cards WHERE id='CARD_TAM_001'"));
      const first = await request.post(`${apiURL}/api/v1/cards/CARD_TAM_001/recharge`, {
        headers: { ...bearer(token), 'Idempotency-Key': key },
        data: { amount_kurus: 5000, description: 'QA idempotency' }
      });
      const second = await request.post(`${apiURL}/api/v1/cards/CARD_TAM_001/recharge`, {
        headers: { ...bearer(token), 'Idempotency-Key': key },
        data: { amount_kurus: 5000, description: 'QA idempotency' }
      });
      expect(first.status()).toBe(200);
      expect(second.status()).toBe(200);
      expect(await second.json()).toEqual(await first.json());
      expect(Number(scalar("SELECT balance_kurus FROM jw_cards WHERE id='CARD_TAM_001'"))).toBe(before + 5000);
      expect(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE reference_id='${key}'`)).toBe('1');
    } finally {
      sql(`UPDATE jw_cards SET user_id=NULL WHERE id='CARD_TAM_001'`);
    }
  });

  test('real validator tap persists one event and one matching ledger debit', async ({ request }) => {
    const key = `QA-TAP-REAL-${Date.now()}`;
    const nextSeq = Math.max(900001, Number(scalar("SELECT COALESCE(MAX(last_sequence_no), 0) FROM jw_validator_credentials WHERE validator_id='VAL_500T_01'") || 0) + 1);
    const tapData = {
      validator_id: 'VAL_500T_01', sequence_no: nextSeq, card_id: 'CARD_TAM_001',
      line_id: 'LINE_500T', station_id: 'STAT_AYRILIK', direction: 1,
      tapped_at: '2026-10-24T14:00:00Z'
    };
    const response = await request.post(`${apiURL}/api/v1/validators/tap`, {
      headers: { ...deviceAuthHeaders('VAL_500T_01', nextSeq, tapData, key), 'Idempotency-Key': key },
      data: tapData
    });
    expect(response.status()).toBe(200);
    expect((await response.json()).event.status).toBe('PROCESSED');
    expect(scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE idempotency_key='${key}'`)).toBe('1');
    expect(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE reference_id='SYNC:VAL_500T_01:${nextSeq}' AND amount_kurus < 0`)).toBe('1');
  });

  test('invalid penalty reference cannot report success without a database row', async ({ request }) => {
    const { token } = await login(request, 'kontrolor');
    const response = await request.post(`${apiURL}/api/v1/penalties`, {
      headers: bearer(token),
      data: { card_id: 'CARD_DOES_NOT_EXIST', penalty_amount_kurus: 5000, reason: 'kartsiz_binis' }
    });
    expect(response.status()).toBeGreaterThanOrEqual(400);
    expect(scalar("SELECT COUNT(*) FROM jw_penalties WHERE card_id='CARD_DOES_NOT_EXIST'")).toBe('0');
  });

  test('settlement close mutates PostgreSQL rather than process memory', async ({ request }) => {
    const { token } = await login(request, 'muhasebe');
    const periodID = scalar("INSERT INTO jw_settlement_periods(period_name,start_date,end_date) VALUES ('QA-PERIOD-1',NOW()-INTERVAL '1 day',NOW()) ON CONFLICT(period_name) DO UPDATE SET is_closed=false,closed_at=NULL RETURNING id");
    const response = await request.post(`${apiURL}/api/v1/settlements/close`, {
      headers: bearer(token), data: { period_id: periodID }
    });
    expect(response.status()).toBe(200);
    expect(scalar(`SELECT is_closed FROM jw_settlement_periods WHERE id='${periodID}'`)).toBe('t');
  });
});
