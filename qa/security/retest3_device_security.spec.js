import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { apiURL } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

const qaRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = path.resolve(qaRoot, '..');

function unique(prefix) {
  return `${prefix}-${Date.now()}-${crypto.randomBytes(6).toString('hex')}`;
}

function sign(rawBody, validatorID, sequence, nonce, deviceSecret, payloadOverride) {
  const timestamp = Math.floor(Date.now() / 1000);
  const payload = payloadOverride ?? rawBody;
  const payloadHash = crypto.createHash('sha256').update(payload).digest('hex');
  const message = `${validatorID}:${sequence}:${timestamp}:${nonce}:${payloadHash}`;
  return {
    'Content-Type': 'application/json',
    'X-Validator-ID': validatorID,
    'X-Device-Timestamp': String(timestamp),
    'X-Device-Nonce': nonce,
    'X-Device-Signature': crypto.createHmac('sha256', deviceSecret).update(message).digest('hex')
  };
}

function createFundedCard(cardID, balance = 10000) {
  sql(`INSERT INTO jw_cards(id,card_type,balance_kurus) VALUES ('${cardID}','tam',${balance})`);
  sql(`INSERT INTO jw_ledger(card_id,entry_type,amount_kurus,balance_after_kurus,reference_id,description)
       VALUES ('${cardID}','dolum',${balance},${balance},'SEED-${cardID}','Retest 3 isolated security seed')`);
}

function nextSequence(validatorID) {
  return Number(scalar(`SELECT last_sequence_no + 1 FROM jw_validator_credentials WHERE validator_id='${validatorID}'`));
}

function storedDeviceSecret(validatorID) {
  return scalar(`SELECT secret_key FROM jw_validator_credentials WHERE validator_id='${validatorID}'`);
}

function tapBody({ validatorID, sequence, cardID, lineID = 'LINE_500T', stationID = 'STAT_AYRILIK', direction = 1, tappedAt }) {
  return {
    validator_id: validatorID,
    sequence_no: sequence,
    card_id: cardID,
    line_id: lineID,
    station_id: stationID,
    direction,
    tapped_at: tappedAt ?? new Date().toISOString()
  };
}

function snapshot(cardID, validatorID, nonce, idempotencyKey) {
  return {
    balance: scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${cardID}'`),
    ledger: scalar(`SELECT COUNT(*) FROM jw_ledger WHERE card_id='${cardID}'`),
    events: scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE idempotency_key='${idempotencyKey}'`),
    sequence: scalar(`SELECT last_sequence_no FROM jw_validator_credentials WHERE validator_id='${validatorID}'`),
    nonce: scalar(`SELECT COUNT(*) FROM jw_device_nonces WHERE nonce='${nonce}'`)
  };
}

function expectSameState(before, after) {
  expect.soft(after.balance).toBe(before.balance);
  expect.soft(after.ledger).toBe(before.ledger);
  expect.soft(after.events).toBe(before.events);
  expect.soft(after.sequence).toBe(before.sequence);
  expect.soft(after.nonce).toBe(before.nonce);
}

async function postRaw(request, rawBody, headers, idempotencyKey) {
  return request.fetch(`${apiURL}/api/v1/validators/tap`, {
    method: 'POST',
    headers: { ...headers, 'Idempotency-Key': idempotencyKey },
    data: rawBody
  });
}

test.describe('Retest 3 PostgreSQL-backed validator security', () => {
  test('retired repository credential cannot impersonate a provisioned validator', async ({ request }) => {
    const validatorID = 'VAL_500T_01';
    const sequence = nextSequence(validatorID);
    const nonce = unique('R3-REPO-MASTER');
    const idempotencyKey = unique('R3-REPO-MASTER-IDEM');
    const cardID = unique('R3_REPO_CARD').replaceAll('-', '_');
    createFundedCard(cardID);
    const rawBody = JSON.stringify(tapBody({ validatorID, sequence, cardID }));
    const retiredCredential = '9a5553a397e6ed44b0dcba6f20da99aeb60bb50c132c985c23a45a3aa4af6600';
    const headers = sign(rawBody, validatorID, sequence, nonce, retiredCredential);
    const before = snapshot(cardID, validatorID, nonce, idempotencyKey);

    const response = await postRaw(request, rawBody, headers, idempotencyKey);
    const after = snapshot(cardID, validatorID, nonce, idempotencyKey);

    expect.soft([401, 403]).toContain(response.status());
    expectSameState(before, after);
  });

  test('runtime configuration contains no shared validator master credential', () => {
    const paths = [
      '.env.example', 'docker-compose.yml', 'qa/docker-compose.qa.yml',
      'internal/config/config.go', 'cmd/server/main.go', 'cmd/simulator/main.go',
      'cmd/verify/main.go', 'qa/helpers/http.js'
    ];
    for (const relativePath of paths) {
      const source = fs.readFileSync(path.join(repoRoot, relativePath), 'utf8');
      expect.soft(source, relativePath).not.toContain('VALIDATOR_HMAC_SECRET');
      expect.soft(source, relativePath).not.toContain('qa-validator-hmac-secret');
      expect.soft(source, relativePath).not.toContain('aktarbilet-rot2026-validator');
    }
  });

  test('signature binds every tap field, including direction and tapped_at', async ({ request }) => {
    const validatorID = 'VAL_500T_01';
    const sequence = nextSequence(validatorID);
    const nonce = unique('R3-CANONICAL');
    const idempotencyKey = unique('R3-CANONICAL-IDEM');
    const cardID = unique('R3_CANON_CARD').replaceAll('-', '_');
    createFundedCard(cardID);
    const original = tapBody({ validatorID, sequence, cardID, direction: 1, tappedAt: new Date(Date.now() - 60_000).toISOString() });
    const tampered = { ...original, direction: 2, tapped_at: new Date().toISOString() };
    const rawTampered = JSON.stringify(tampered);
    const canonical = `${original.card_id}:${original.line_id}:${original.station_id}`;
    const headers = sign(rawTampered, validatorID, sequence, nonce, storedDeviceSecret(validatorID), canonical);
    const before = snapshot(cardID, validatorID, nonce, idempotencyKey);

    const response = await postRaw(request, rawTampered, headers, idempotencyKey);
    const after = snapshot(cardID, validatorID, nonce, idempotencyKey);

    expect.soft(response.status()).toBe(401);
    expectSameState(before, after);
  });

  test('another provisioned device key cannot authorize the claimed validator', async ({ request }) => {
    const validatorID = 'VAL_500T_01';
    const sequence = nextSequence(validatorID);
    const nonce = unique('R3-CROSS-DEVICE');
    const idempotencyKey = unique('R3-CROSS-DEVICE-IDEM');
    const cardID = unique('R3_CROSS_CARD').replaceAll('-', '_');
    createFundedCard(cardID);
    const rawBody = JSON.stringify(tapBody({ validatorID, sequence, cardID }));
    const headers = sign(rawBody, validatorID, sequence, nonce, storedDeviceSecret('VAL_14ES_01'));
    const before = snapshot(cardID, validatorID, nonce, idempotencyKey);

    const response = await postRaw(request, rawBody, headers, idempotencyKey);
    const after = snapshot(cardID, validatorID, nonce, idempotencyKey);

    expect.soft(response.status()).toBe(401);
    expectSameState(before, after);
  });

  test('same sequence with a different signed payload is rejected', async ({ request }) => {
    const validatorID = 'VAL_500T_01';
    const sequence = nextSequence(validatorID);
    const firstCard = unique('R3_SEQ_A').replaceAll('-', '_');
    const secondCard = unique('R3_SEQ_B').replaceAll('-', '_');
    createFundedCard(firstCard);
    createFundedCard(secondCard);
    const secret = storedDeviceSecret(validatorID);
    const firstNonce = unique('R3-SEQ-FIRST');
    const firstKey = unique('R3-SEQ-FIRST-IDEM');
    const firstRaw = JSON.stringify(tapBody({ validatorID, sequence, cardID: firstCard }));
    const firstResponse = await postRaw(request, firstRaw, sign(firstRaw, validatorID, sequence, firstNonce, secret), firstKey);
    expect(firstResponse.status()).toBe(200);

    const secondNonce = unique('R3-SEQ-SECOND');
    const secondKey = unique('R3-SEQ-SECOND-IDEM');
    const secondRaw = JSON.stringify(tapBody({ validatorID, sequence, cardID: secondCard, direction: 2 }));
    const before = snapshot(secondCard, validatorID, secondNonce, secondKey);
    const response = await postRaw(request, secondRaw, sign(secondRaw, validatorID, sequence, secondNonce, secret), secondKey);
    const after = snapshot(secondCard, validatorID, secondNonce, secondKey);

    expect.soft(response.status()).toBe(409);
    expectSameState(before, after);
  });

  test('an exact signed retry is idempotent and creates one financial effect', async ({ request }) => {
    const validatorID = 'VAL_500T_01';
    const sequence = nextSequence(validatorID);
    const nonce = unique('R3-EXACT');
    const idempotencyKey = unique('R3-EXACT-IDEM');
    const cardID = unique('R3_EXACT_CARD').replaceAll('-', '_');
    createFundedCard(cardID);
    const rawBody = JSON.stringify(tapBody({ validatorID, sequence, cardID }));
    const headers = sign(rawBody, validatorID, sequence, nonce, storedDeviceSecret(validatorID));

    const first = await postRaw(request, rawBody, headers, idempotencyKey);
    const second = await postRaw(request, rawBody, headers, idempotencyKey);

    expect(first.status()).toBe(200);
    expect(second.status()).toBe(200);
    expect(scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE idempotency_key='${idempotencyKey}'`)).toBe('1');
    expect(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE card_id='${cardID}' AND entry_type='binis'`)).toBe('1');
  });

  test('validly signed event for a validator-assignment mismatch is rejected', async ({ request }) => {
    const validatorID = 'VAL_500T_01';
    const sequence = nextSequence(validatorID);
    const nonce = unique('R3-ASSIGNMENT');
    const idempotencyKey = unique('R3-ASSIGNMENT-IDEM');
    const cardID = unique('R3_ASSIGN_CARD').replaceAll('-', '_');
    createFundedCard(cardID);
    const rawBody = JSON.stringify(tapBody({
      validatorID, sequence, cardID, lineID: 'LINE_T1', stationID: 'STAT_KABATAS'
    }));
    const headers = sign(rawBody, validatorID, sequence, nonce, storedDeviceSecret(validatorID));
    const before = snapshot(cardID, validatorID, nonce, idempotencyKey);

    const response = await postRaw(request, rawBody, headers, idempotencyKey);
    const after = snapshot(cardID, validatorID, nonce, idempotencyKey);

    expect.soft([403, 422]).toContain(response.status());
    expectSameState(before, after);
  });

  test('revoked device rejection does not consume nonce or sequence', async ({ request }) => {
    const validatorID = 'VAL_14ES_01';
    const sequence = nextSequence(validatorID);
    const nonce = unique('R3-REVOKED');
    const idempotencyKey = unique('R3-REVOKED-IDEM');
    const cardID = unique('R3_REVOKED_CARD').replaceAll('-', '_');
    createFundedCard(cardID);
    const rawBody = JSON.stringify(tapBody({
      validatorID, sequence, cardID, lineID: 'LINE_14ES', stationID: 'STAT_KADIKOY'
    }));
    const headers = sign(rawBody, validatorID, sequence, nonce, storedDeviceSecret(validatorID));
    sql(`UPDATE jw_validator_credentials SET is_revoked=true WHERE validator_id='${validatorID}'`);
    const before = snapshot(cardID, validatorID, nonce, idempotencyKey);

    try {
      const response = await postRaw(request, rawBody, headers, idempotencyKey);
      const after = snapshot(cardID, validatorID, nonce, idempotencyKey);
      expect.soft(response.status()).toBe(403);
      expectSameState(before, after);
    } finally {
      sql(`UPDATE jw_validator_credentials SET is_revoked=false WHERE validator_id='${validatorID}'`);
    }
  });
});
