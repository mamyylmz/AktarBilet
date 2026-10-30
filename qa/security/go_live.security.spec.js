import { test, expect } from '@playwright/test';
import { apiURL, bearer, login } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

function nonce(prefix) {
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function createCard(id, balance = 0) {
  sql(`INSERT INTO jw_cards(id,card_type,balance_kurus) VALUES ('${id}','tam',${balance})`);
  if (balance !== 0) {
    sql(`INSERT INTO jw_ledger(card_id,entry_type,amount_kurus,balance_after_kurus,reference_id,description)
         VALUES ('${id}','dolum',${balance},${balance},'SEED-${id}','QA security seed')`);
  }
}

test.describe('go-live negative security controls', () => {
  test('SEC-03 passenger cannot recharge an unowned card with an arbitrary payload', async ({ request }) => {
    const { token } = await login(request, 'yolcu');
    const key = nonce('SEC-IDOR');
    const before = Number(scalar("SELECT balance_kurus FROM jw_cards WHERE id='CARD_TAM_001'"));
    const response = await request.post(`${apiURL}/api/v1/cards/CARD_TAM_001/recharge`, {
      headers: { ...bearer(token), 'Idempotency-Key': key },
      data: { amount_kurus: 137, description: 'ordinary kiosk recharge' }
    });
    const after = Number(scalar("SELECT balance_kurus FROM jw_cards WHERE id='CARD_TAM_001'"));

    expect.soft([403, 404]).toContain(response.status());
    expect(after, 'an IDOR attempt must have no financial side effect').toBe(before);
  });

  test('SEC-05 a VAL_ prefix is not a device credential and failure is atomic', async ({ request }) => {
    const suffix = `${Date.now()}${Math.floor(Math.random() * 1000)}`;
    const card = `QA_SEC_${suffix}`;
    const validator = `VAL_ATTACKER_${suffix}`;
    const key = `SEC-FAKE-VALIDATOR-${suffix}`;
    createCard(card, 5000);

    const response = await request.post(`${apiURL}/api/v1/validators/tap`, {
      headers: { 'Idempotency-Key': key },
      data: {
        validator_id: validator, sequence_no: Number(suffix.slice(-12)), card_id: card,
        line_id: 'LINE_5', station_id: 'STAT_ESHOT_10177', direction: 1,
        tapped_at: new Date().toISOString()
      }
    });

    expect.soft([401, 403]).toContain(response.status());
    expect.soft(scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${card}'`)).toBe('5000');
    expect.soft(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE card_id='${card}'`)).toBe('1');
    expect(scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE idempotency_key='${key}'`)).toBe('0');
  });

  test('SEC-05 even a seeded validator requires an authenticated device credential', async ({ request }) => {
    const suffix = `${Date.now()}${Math.floor(Math.random() * 1000)}`;
    const card = `QA_AUTH_${suffix}`;
    createCard(card, 5000);
    const response = await request.post(`${apiURL}/api/v1/validators/tap`, {
      headers: { 'Idempotency-Key': `SEC-NO-DEVICE-CREDENTIAL-${suffix}` },
      data: {
        validator_id: 'VAL_500T_01', sequence_no: Number(suffix.slice(-12)), card_id: card,
        line_id: 'LINE_5', station_id: 'STAT_ESHOT_10177', direction: 1,
        tapped_at: new Date().toISOString()
      }
    });
    expect([401, 403]).toContain(response.status());
  });

  test('SEC-02 refresh token cannot be used as an access token', async ({ request }) => {
    const { body } = await login(request, 'yolcu');
    expect(body.refresh_token).toBeTruthy();
    const response = await request.get(`${apiURL}/api/v1/auth/me`, {
      headers: bearer(body.refresh_token)
    });
    expect(response.status()).toBe(401);
  });

  test('SEC-06 conflicting reuse of an idempotency key is rejected', async ({ request }) => {
    const { token } = await login(request, 'admin');
    const key = nonce('SEC-IDEMP-CONFLICT');
    const first = await request.post(`${apiURL}/api/v1/cards/CARD_TAM_001/recharge`, {
      headers: { ...bearer(token), 'Idempotency-Key': key },
      data: { amount_kurus: 101, description: 'first payload' }
    });
    const second = await request.post(`${apiURL}/api/v1/cards/CARD_TAM_001/recharge`, {
      headers: { ...bearer(token), 'Idempotency-Key': key },
      data: { amount_kurus: 9999, description: 'conflicting payload' }
    });
    expect(first.status()).toBe(200);
    expect([409, 422]).toContain(second.status());
    expect(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE reference_id='${key}'`)).toBe('1');
  });
});
