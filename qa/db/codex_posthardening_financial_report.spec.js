import { randomUUID } from 'node:crypto';
import { test, expect } from '@playwright/test';
import { apiURL, deviceAuthHeaders, login } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

test('a persisted fare event updates settlement revenue and operator report totals', async ({ request }) => {
  const { token } = await login(request, 'muhasebe');
  const readReport = async () => {
    const response = await request.get(`${apiURL}/api/v1/settlements`, { headers: { Authorization: `Bearer ${token}` } });
    expect(response.status()).toBe(200);
    return response.json();
  };
  const before = await readReport();
  const cardID = `CARD_REPORT_${randomUUID()}`;
  const idempotencyKey = `QA-REPORT-${randomUUID()}`;
  sql(`INSERT INTO jw_cards(id,card_type,balance_kurus,daily_taps_count,monthly_capping_spent_kurus)
       VALUES ('${cardID}','tam',10000,0,0);
       INSERT INTO jw_ledger(card_id,entry_type,amount_kurus,balance_after_kurus,reference_id,description)
       VALUES ('${cardID}','dolum',10000,10000,'SEED-${cardID}','QA report fixture')`);

  const payload = {
    validator_id: 'VAL_500T_01', sequence_no: Date.now() * 1000,
    card_id: cardID, line_id: 'LINE_5', station_id: 'STAT_ESHOT_10177',
    direction: 1, tapped_at: new Date().toISOString(),
  };
  const body = JSON.stringify(payload);
  const response = await request.post(`${apiURL}/api/v1/validators/tap`, {
    headers: {
      ...deviceAuthHeaders(payload.validator_id, payload.sequence_no, body, `nonce-${idempotencyKey}`),
      'Idempotency-Key': idempotencyKey,
      'Content-Type': 'application/json',
    },
    data: body,
  });
  expect(response.status()).toBe(200);
  const tap = await response.json();
  const fare = tap.event.fare_charged_kurus;
  expect(scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE idempotency_key='${idempotencyKey}' AND status='PROCESSED'`)).toBe('1');
  expect(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE card_id='${cardID}' AND amount_kurus=-${fare}`)).toBe('1');

  const after = await readReport();
  expect.soft(after.total_trips).toBe(before.total_trips + 1);
  expect.soft(after.total_revenue_kurus).toBe(before.total_revenue_kurus + fare);
  const municipalityBefore = before.operator_shares.find((share) => share.operator.startsWith('İzmir Büyükşehir Belediyesi'));
  const municipalityAfter = after.operator_shares.find((share) => share.operator.startsWith('İzmir Büyükşehir Belediyesi'));
  expect.soft(municipalityAfter.share_kurus).toBe(municipalityBefore.share_kurus + fare);
});
