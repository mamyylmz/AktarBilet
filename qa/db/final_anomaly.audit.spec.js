import { test, expect } from '@playwright/test';
import { apiURL } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

test('T5 impossible-travel anomaly is durably linked to its validator event', async ({ request }) => {
  const suffix = `${Date.now()}${Math.floor(Math.random() * 1000)}`;
  const card = `QA_ANOM_${suffix}`;
  const firstTime = new Date(Date.now() - 120_000);
  const secondTime = new Date(firstTime.getTime() + 60_000);
  const sequence = Number(suffix.slice(-10)) * 10;

  sql(`INSERT INTO jw_cards(id,card_type,balance_kurus) VALUES ('${card}','tam',10000);
       INSERT INTO jw_ledger(card_id,entry_type,amount_kurus,balance_after_kurus,reference_id,description)
       VALUES ('${card}','dolum',10000,10000,'SEED-${card}','final anomaly audit seed')`);

  const sendTap = (station, tappedAt, offset) => request.post(`${apiURL}/api/v1/validators/tap`, {
    headers: { 'Idempotency-Key': `${card}-${offset}` },
    data: {
      validator_id: 'VAL_500T_01', sequence_no: sequence + offset, card_id: card,
      line_id: 'LINE_500T', station_id: station, direction: 1,
      tapped_at: tappedAt.toISOString()
    }
  });

  expect((await sendTap('STAT_KADIKOY', firstTime, 1)).status()).toBe(200);
  const second = await sendTap('STAT_SARIYER', secondTime, 2);
  expect(second.status()).toBe(200);
  const body = await second.json();
  expect(body.anomaly.IsAnomaly).toBe(true);
  expect(body.anomaly.AnomalyType).toBe('IMPOSSIBLE_TRAVEL_SPEED');

  expect(scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE card_id='${card}'`)).toBe('2');
  expect(scalar(`SELECT COUNT(*) FROM jw_anomalies WHERE card_id='${card}'`)).toBe('1');
  expect(scalar(`SELECT COUNT(*) FROM jw_anomalies a JOIN jw_validator_events e ON e.id=a.event_id WHERE a.card_id='${card}'`)).toBe('1');
});
