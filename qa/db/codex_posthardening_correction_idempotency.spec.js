import { randomUUID } from 'node:crypto';
import { test, expect } from '@playwright/test';
import { apiURL, deviceAuthHeaders } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

function tap(request, event) {
  const body = JSON.stringify(event.payload);
  return request.post(`${apiURL}/api/v1/validators/tap`, {
    headers: {
      ...deviceAuthHeaders(event.validator, event.sequence, body, `nonce-${event.idempotencyKey}`),
      'Idempotency-Key': event.idempotencyKey, 'Content-Type': 'application/json',
    }, data: body,
  });
}

test('each legitimate future-event repricing gets one correction; retries add no money', async ({ request }) => {
  const cardID = `CARD_CORRECT_${randomUUID()}`;
  sql(`INSERT INTO jw_cards(id,card_type,balance_kurus,daily_taps_count,monthly_capping_spent_kurus)
       VALUES ('${cardID}','tam',20000,0,0);
       INSERT INTO jw_ledger(card_id,entry_type,amount_kurus,balance_after_kurus,reference_id,description)
       VALUES ('${cardID}','dolum',20000,20000,'SEED-${cardID}','QA correction fixture')`);

  const t1 = new Date(Date.now() - 3_600_000);
  const t2 = new Date(t1.getTime() + 20 * 60_000);
  const t3 = new Date(t1.getTime() + 40 * 60_000);
  const seq = Date.now() * 10;
  const events = [
    { validator: 'VAL_500T_01', sequence: seq + 1, line: 'LINE_5', station: 'STAT_ESHOT_10177', tappedAt: t1 },
    { validator: 'VAL_14ES_01', sequence: seq + 2, line: 'LINE_10', station: 'STAT_ESHOT_10145', tappedAt: t2 },
    { validator: 'VAL_T1_01', sequence: seq + 3, line: 'LINE_15', station: 'STAT_ESHOT_10008', tappedAt: t3 },
  ].map((event, index) => ({
    ...event, idempotencyKey: `QA-CORRECTION-${randomUUID()}-${index}`,
    payload: { validator_id: event.validator, sequence_no: event.sequence, card_id: cardID, line_id: event.line,
      station_id: event.station, direction: 1, tapped_at: event.tappedAt.toISOString() },
  }));

  const futureFirst = await tap(request, events[2]);
  expect(futureFirst.status()).toBe(200);
  const middle = await tap(request, events[1]);
  expect(middle.status()).toBe(200);
  const futureID = scalar(`SELECT id FROM jw_validator_events WHERE idempotency_key='${events[2].idempotencyKey}'`);
  const early = await tap(request, events[0]);
  expect(early.status()).toBe(200);

  const refPrefix = `OOO-REPRICE:${futureID}:`;
  const correctionCount = () => scalar(`SELECT COUNT(*) FROM jw_ledger WHERE reference_id LIKE '${refPrefix}%'`);
  const correctionSum = () => scalar(`SELECT COALESCE(SUM(amount_kurus),0) FROM jw_ledger WHERE reference_id LIKE '${refPrefix}%'`);
  expect(correctionCount()).toBe('2');
  expect(correctionSum()).toBe('1500');
  expect(scalar(`SELECT fare_charged_kurus FROM jw_validator_events WHERE id='${futureID}'`)).toBe('500');

  const retry = await tap(request, events[1]);
  expect(retry.status()).toBe(200);
  expect(correctionCount()).toBe('2');
  expect(correctionSum()).toBe('1500');
  expect(scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${cardID}'`)).toBe('16500');
  expect(scalar(`SELECT SUM(amount_kurus) FROM jw_ledger WHERE card_id='${cardID}'`)).toBe('16500');

  const changedRequestKey = { ...events[1], idempotencyKey: `QA-CORRECTION-ALT-${randomUUID()}` };
  const replay = await tap(request, changedRequestKey);
  expect(replay.status()).toBe(409);
  expect(correctionCount()).toBe('2');
  expect(scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE card_id='${cardID}'`)).toBe('3');
});
