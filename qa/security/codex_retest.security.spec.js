import { test, expect } from '@playwright/test';
import crypto from 'node:crypto';
import { apiURL, bearer, login } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

function unique(prefix) {
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function createFundedCard(cardID, balance = 10000) {
  sql(`INSERT INTO jw_cards(id,card_type,balance_kurus) VALUES ('${cardID}','tam',${balance})`);
  sql(`INSERT INTO jw_ledger(card_id,entry_type,amount_kurus,balance_after_kurus,reference_id,description)
       VALUES ('${cardID}','dolum',${balance},${balance},'SEED-${cardID}','Codex independent retest seed')`);
}

function assertUntouched(cardID, idempotencyKey) {
  expect.soft(scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${cardID}'`)).toBe('10000');
  expect.soft(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE card_id='${cardID}'`)).toBe('1');
  expect.soft(scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE idempotency_key='${idempotencyKey}'`)).toBe('0');
}

function signedDeviceHeaders(rawBody, validatorID, sequence, nonce, exposedCredential) {
  const timestamp = Math.floor(Date.now() / 1000);
  const payloadHash = crypto.createHash('sha256').update(rawBody).digest('hex');
  const message = `${validatorID}:${sequence}:${timestamp}:${nonce}:${payloadHash}`;
  const signature = crypto.createHmac('sha256', exposedCredential).update(message).digest('hex');
  return {
    'Content-Type': 'application/json',
    'X-Validator-ID': validatorID,
    'X-Device-Timestamp': String(timestamp),
    'X-Device-Nonce': nonce,
    'X-Device-Signature': signature
  };
}

test.describe('Codex independent security retest', () => {
  test('a registered validator ID is not an anonymous device credential', async ({ request }) => {
    const suffix = Date.now();
    const cardID = `CODEX_ANON_${suffix}`;
    const key = unique('CODEX-ANON-TAP');
    createFundedCard(cardID);

    const response = await request.post(`${apiURL}/api/v1/validators/tap`, {
      headers: { 'Idempotency-Key': key },
      data: {
        validator_id: 'VAL_500T_01', sequence_no: suffix, card_id: cardID,
        line_id: 'LINE_5', station_id: 'STAT_ESHOT_10177', direction: 1,
        tapped_at: new Date().toISOString()
      }
    });

    expect.soft([401, 403]).toContain(response.status());
    assertUntouched(cardID, key);
  });

  test('the anonymous batch endpoint requires cryptographic device authentication', async ({ request }) => {
    const suffix = Date.now();
    const cardID = `CODEX_BATCH_${suffix}`;
    const key = unique('CODEX-ANON-BATCH');
    createFundedCard(cardID);

    const response = await request.post(`${apiURL}/api/v1/validators/batch-sync`, {
      headers: { 'X-Validator-ID': 'VAL_500T_01' },
      data: [{
        idempotency_key: key, validator_id: 'VAL_500T_01', sequence_no: suffix,
        card_id: cardID, line_id: 'LINE_5', station_id: 'STAT_ESHOT_10177', direction: 1,
        tapped_at: new Date().toISOString()
      }]
    });

    expect.soft([401, 403]).toContain(response.status());
    assertUntouched(cardID, key);
  });

  test('a non-device user JWT cannot authorize validator ingestion', async ({ request }) => {
    const { token } = await login(request, 'kontrolor');
    const suffix = Date.now();
    const cardID = `CODEX_USERJWT_${suffix}`;
    const key = unique('CODEX-USER-JWT');
    createFundedCard(cardID);

    const response = await request.post(`${apiURL}/api/v1/validators/tap`, {
      headers: { ...bearer(token), 'Idempotency-Key': key },
      data: {
        validator_id: 'VAL_500T_01', sequence_no: suffix, card_id: cardID,
        line_id: 'LINE_5', station_id: 'STAT_ESHOT_10177', direction: 1,
        tapped_at: new Date().toISOString()
      }
    });

    expect.soft([401, 403]).toContain(response.status());
    assertUntouched(cardID, key);
  });

  test('an idempotency key cannot bypass passenger card ownership', async ({ request }) => {
    const { token } = await login(request, 'yolcu');
    const cardID = `CODEX_IDOR_${Date.now()}`;
    const key = 'QA-RECHARGE-IDEMPOTENCY-001';
    createFundedCard(cardID);

    const response = await request.post(`${apiURL}/api/v1/cards/${cardID}/recharge`, {
      headers: { ...bearer(token), 'Idempotency-Key': key },
      data: { amount_kurus: 137, description: 'Codex ownership bypass probe' }
    });

    expect.soft([403, 404]).toContain(response.status());
    expect.soft(scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${cardID}'`)).toBe('10000');
    expect(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE card_id='${cardID}'`)).toBe('1');
  });

  test('a random idempotency key cannot bypass passenger card ownership', async ({ request }) => {
    const { token } = await login(request, 'yolcu');
    const cardID = `CODEX_RANDOM_IDOR_${Date.now()}`;
    const key = unique('CODEX-RANDOM-RECHARGE');
    createFundedCard(cardID);

    const response = await request.post(`${apiURL}/api/v1/cards/${cardID}/recharge`, {
      headers: { ...bearer(token), 'Idempotency-Key': key },
      data: { amount_kurus: 137, description: 'Codex randomized ownership probe' }
    });

    expect.soft([403, 404]).toContain(response.status());
    expect.soft(scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${cardID}'`)).toBe('10000');
    expect(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE card_id='${cardID}'`)).toBe('1');
  });

  test('a repository-embedded validator credential cannot authorize an external caller', async ({ request }) => {
    const validatorID = 'VAL_500T_01';
    const exposedCredential = '9a5553a397e6ed44b0dcba6f20da99aeb60bb50c132c985c23a45a3aa4af6600';
    const sequence = Date.now();
    const nonce = unique('CODEX-EXPOSED-CREDENTIAL');
    const cardID = `CODEX_EXPOSED_${sequence}`;
    const key = unique('CODEX-EXPOSED-TAP');
    createFundedCard(cardID);
    const rawBody = JSON.stringify({
      validator_id: validatorID, sequence_no: sequence, card_id: cardID,
      line_id: 'LINE_5', station_id: 'STAT_ESHOT_10177', direction: 1,
      tapped_at: new Date().toISOString()
    });

    const response = await request.fetch(`${apiURL}/api/v1/validators/tap`, {
      method: 'POST',
      headers: {
        ...signedDeviceHeaders(rawBody, validatorID, sequence, nonce, exposedCredential),
        'Idempotency-Key': key
      },
      data: rawBody
    });

    expect.soft([401, 403]).toContain(response.status());
    assertUntouched(cardID, key);
  });

  test('a signed batch principal cannot submit events for another validator', async ({ request }) => {
    const signingValidator = 'VAL_500T_01';
    const eventValidator = 'VAL_14ES_01';
    const exposedCredential = '9a5553a397e6ed44b0dcba6f20da99aeb60bb50c132c985c23a45a3aa4af6600';
    const sequence = Date.now() + 1000;
    const nonce = unique('CODEX-VALIDATOR-CONFUSION');
    const cardID = `CODEX_CONFUSION_${sequence}`;
    const key = unique('CODEX-CONFUSED-BATCH');
    createFundedCard(cardID);
    const rawBody = JSON.stringify([{
      idempotency_key: key, validator_id: eventValidator, sequence_no: sequence,
      card_id: cardID, line_id: 'LINE_10', station_id: 'STAT_ESHOT_10145', direction: 1,
      tapped_at: new Date().toISOString()
    }]);

    const response = await request.fetch(`${apiURL}/api/v1/validators/batch-sync`, {
      method: 'POST',
      headers: signedDeviceHeaders(rawBody, signingValidator, sequence, nonce, exposedCredential),
      data: rawBody
    });

    expect.soft([401, 403]).toContain(response.status());
    assertUntouched(cardID, key);
  });

  test('changing a real JWT signature byte is rejected', async ({ request }) => {
    const { token } = await login(request, 'operasyon');
    const parts = token.split('.');
    const signature = Buffer.from(parts[2], 'base64url');
    signature[0] ^= 0xff;
    parts[2] = signature.toString('base64url');

    const response = await request.get(`${apiURL}/api/v1/anomalies`, {
      headers: bearer(parts.join('.'))
    });
    expect(response.status()).toBe(401);
  });
});
