import { test, expect } from '@playwright/test';
import { apiURL, deviceAuthHeaders } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

test('T5 impossible-travel anomaly is durably linked to its validator event', async ({ request }) => {
  const suffix = `${Date.now()}${Math.floor(Math.random() * 1000)}`;
  const card = `QA_ANOM_${suffix}`;
  const firstTime = new Date(Date.now() - 120_000);
  const secondTime = new Date(firstTime.getTime() + 60_000);
  sql(`INSERT INTO jw_cards(id,card_type,balance_kurus) VALUES ('${card}','tam',10000);
       INSERT INTO jw_ledger(card_id,entry_type,amount_kurus,balance_after_kurus,reference_id,description)
       VALUES ('${card}','dolum',10000,10000,'SEED-${card}','final anomaly audit seed')`);

  const sendTap = (validatorID, lineID, station, tappedAt, offset) => {
    const tapSeq = Number(scalar(`SELECT COALESCE(MAX(sequence_no), 0) + 1 FROM jw_device_requests WHERE validator_id='${validatorID}'`));
    const idemp = `${card}-${offset}`;
    const data = {
      validator_id: validatorID, sequence_no: tapSeq, card_id: card,
      line_id: lineID, station_id: station, direction: 1,
      tapped_at: tappedAt.toISOString()
    };
    return request.post(`${apiURL}/api/v1/validators/tap`, {
      headers: { ...deviceAuthHeaders(validatorID, tapSeq, data, idemp), 'Idempotency-Key': idemp },
      data
    });
  };

  expect((await sendTap('VAL_T1_01', 'LINE_T1', 'STAT_KABATAS', firstTime, 1)).status()).toBe(200);
  const second = await sendTap('VAL_14ES_01', 'LINE_14ES', 'STAT_KADIKOY', secondTime, 2);
  expect(second.status()).toBe(200);
  const body = await second.json();
  expect(body.anomaly.IsAnomaly).toBe(true);
  expect(body.anomaly.AnomalyType).toBe('IMPOSSIBLE_TRAVEL_SPEED');

  expect(scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE card_id='${card}'`)).toBe('2');
  expect(scalar(`SELECT COUNT(*) FROM jw_anomalies WHERE card_id='${card}'`)).toBe('1');
  expect(scalar(`SELECT COUNT(*) FROM jw_anomalies a JOIN jw_validator_events e ON e.id=a.event_id WHERE a.card_id='${card}'`)).toBe('1');
});
