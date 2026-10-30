import { execFileSync } from 'node:child_process';
import { test, expect } from '@playwright/test';
import { apiURL, bearer, deviceAuthHeaders, login } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

const project = process.env.QA_PROJECT_NAME || 'aktarbilet-qa';
const composeFile = 'docker-compose.qa.yml';

function unique(prefix) {
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function createCard(card, balance) {
  sql(`INSERT INTO jw_cards(id,card_type,balance_kurus) VALUES ('${card}','tam',${balance})`);
  if (balance !== 0) {
    sql(`INSERT INTO jw_ledger(card_id,entry_type,amount_kurus,balance_after_kurus,reference_id,description)
         VALUES ('${card}','dolum',${balance},${balance},'SEED-${card}','final audit seed')`);
  }
}

function assertLedgerEquality(card) {
  expect(scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${card}'`)).toBe(
    scalar(`SELECT COALESCE(SUM(amount_kurus),0) FROM jw_ledger WHERE card_id='${card}'`)
  );
}

function tap(request, card, sequence, key, tappedAt = new Date().toISOString()) {
  const body = JSON.stringify({
    validator_id: 'VAL_500T_01', sequence_no: sequence, card_id: card,
    line_id: 'LINE_5', station_id: 'STAT_ESHOT_10177', direction: 1,
    tapped_at: tappedAt
  });
  return request.post(`${apiURL}/api/v1/validators/tap`, {
    headers: {
      ...deviceAuthHeaders('VAL_500T_01', sequence, body, `nonce-${key}`),
      'Idempotency-Key': key,
      'Content-Type': 'application/json'
    },
    data: body
  });
}

test.describe('final audit PostgreSQL financial workflows', () => {
  test('parallel taps on one card preserve balance and unique effects', async ({ request }) => {
    const card = `QA_TAPS_${Date.now()}`;
    createCard(card, 100000);
    const base = Number(`${Date.now()}`.slice(-10)) * 100;
    const responses = await Promise.all(Array.from({ length: 20 }, (_, i) =>
      tap(request, card, base + i, `${card}-TAP-${i}`)
    ));
    expect(responses.every((response) => response.status() === 200)).toBe(true);
    expect(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE card_id='${card}'`)).toBe('21');
    expect(scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE card_id='${card}'`)).toBe('20');
    expect(scalar(`SELECT COUNT(*)-COUNT(DISTINCT reference_id) FROM jw_ledger WHERE card_id='${card}'`)).toBe('0');
    assertLedgerEquality(card);
  });

  test('concurrent tap and recharge preserve the complete accounting identity', async ({ request }) => {
    const { token } = await login(request, 'admin');
    const card = `QA_MIX_${Date.now()}`;
    createCard(card, 100000);
    const base = Number(`${Date.now()}`.slice(-10)) * 100;
    const operations = [];
    for (let i = 0; i < 15; i += 1) {
      operations.push(tap(request, card, base + i, `${card}-TAP-${i}`));
      operations.push(request.post(`${apiURL}/api/v1/cards/${card}/recharge`, {
        headers: { ...bearer(token), 'Idempotency-Key': `${card}-REC-${i}` },
        data: { amount_kurus: 101, description: 'concurrent final audit' }
      }));
    }
    const responses = await Promise.all(operations);
    expect(responses.every((response) => response.status() === 200)).toBe(true);
    expect(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE card_id='${card}'`)).toBe('31');
    assertLedgerEquality(card);
  });

  test('concurrent duplicate tap produces one event and one debit', async ({ request }) => {
    const card = `QA_DUP_${Date.now()}`;
    createCard(card, 10000);
    const sequence = Number(`${Date.now()}`.slice(-10)) * 100;
    const key = unique('QA-CONCURRENT-DUP');
    const tappedAt = new Date().toISOString();
    const responses = await Promise.all(Array.from({ length: 20 }, () =>
      tap(request, card, sequence, key, tappedAt)
    ));
    expect(responses.every((response) => response.status() === 200)).toBe(true);
    expect(scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE idempotency_key='${key}'`)).toBe('1');
    expect(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE reference_id='SYNC:VAL_500T_01:${sequence}'`)).toBe('1');
    assertLedgerEquality(card);
  });

  test('insufficient balance under contention cannot cross the tolerance limit', async ({ request }) => {
    const card = `QA_LIMIT_${Date.now()}`;
    createCard(card, 0);
    const base = Number(`${Date.now()}`.slice(-10)) * 100;
    const responses = await Promise.all(Array.from({ length: 10 }, (_, i) =>
      tap(request, card, base + i, `${card}-TAP-${i}`)
    ));
    expect(responses.every((response) => response.status() === 200)).toBe(true);
    expect(Number(scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${card}'`))).toBeGreaterThanOrEqual(-5000);
    expect(Number(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE card_id='${card}' AND entry_type='binis'`))).toBeLessThan(10);
    assertLedgerEquality(card);
  });

  test('batch retry has one event and one financial effect per item', async ({ request }) => {
    const suffix = `${Date.now()}`;
    const cards = Array.from({ length: 5 }, (_, i) => `QA_BATCH_${suffix}_${i}`);
    cards.forEach((card) => createCard(card, 10000));
    const base = Number(suffix.slice(-10)) * 100;
    const payload = cards.map((card, i) => ({
      idempotency_key: `${card}-IDEMP`, validator_id: 'VAL_500T_01', sequence_no: base + i,
      card_id: card, line_id: 'LINE_5', station_id: 'STAT_ESHOT_10177', direction: 1,
      tapped_at: new Date(Date.now() + i).toISOString()
    }));
    const body = JSON.stringify(payload);
    const authHeaders = deviceAuthHeaders('VAL_500T_01', base + payload.length - 1, body, `nonce-${suffix}`);
    const send = () => request.post(`${apiURL}/api/v1/validators/batch-sync`, {
      headers: { ...authHeaders, 'Content-Type': 'application/json' }, data: body
    });
    expect((await send()).status()).toBe(200);
    expect((await send()).status()).toBe(200);
    for (let i = 0; i < cards.length; i += 1) {
      expect(scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE idempotency_key='${cards[i]}-IDEMP'`)).toBe('1');
      expect(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE reference_id='SYNC:VAL_500T_01:${base + i}'`)).toBe('1');
      assertLedgerEquality(cards[i]);
    }
  });

  test('recharge survives backend restart and retry remains idempotent', async ({ request }) => {
    test.setTimeout(60_000);
    const { token } = await login(request, 'admin');
    const card = `QA_RESTART_${Date.now()}`;
    const key = `${card}-REC`;
    createCard(card, 1000);
    const payload = { amount_kurus: 777, description: 'restart persistence audit' };
    const first = await request.post(`${apiURL}/api/v1/cards/${card}/recharge`, {
      headers: { ...bearer(token), 'Idempotency-Key': key }, data: payload
    });
    expect(first.status()).toBe(200);
    execFileSync('docker', ['compose', '--project-name', project, '--file', composeFile, 'restart', 'backend'], {
      cwd: process.cwd(), encoding: 'utf8'
    });
    let healthy = false;
    for (let i = 0; i < 40; i += 1) {
      try {
        const response = await request.get(`${apiURL}/health`, { timeout: 1_000 });
        if (response.status() === 200) { healthy = true; break; }
      } catch {}
      await new Promise((resolve) => setTimeout(resolve, 500));
    }
    expect(healthy).toBe(true);
    const second = await request.post(`${apiURL}/api/v1/cards/${card}/recharge`, {
      headers: { ...bearer(token), 'Idempotency-Key': key }, data: payload
    });
    expect(second.status()).toBe(200);
    expect(scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${card}'`)).toBe('1777');
    expect(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE reference_id='${key}'`)).toBe('1');
    assertLedgerEquality(card);
  });
});
