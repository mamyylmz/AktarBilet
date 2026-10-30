import { randomUUID } from 'node:crypto';
import { test, expect } from '@playwright/test';
import { apiURL, bearer, login } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

const periodID = '00000000-0000-0000-0000-000000000001';

test('settlement report applies signed income and storno rows to revenue and operator share', async ({ request }) => {
  const { token } = await login(request, 'muhasebe');
  const read = async () => {
    const response = await request.get(`${apiURL}/api/v1/settlements`, { headers: bearer(token) });
    expect(response.status()).toBe(200);
    return response.json();
  };
  const before = await read();
  const originalID = randomUUID();
  const correctionKey = `QA-REPORT-STORNO-${randomUUID()}`;
  sql(`INSERT INTO jw_settlement_ledger(id,period_id,operator,amount_kurus,is_storno,storno_reason,idempotency_key,created_at)
       VALUES ('${originalID}','${periodID}','belediye',1750,false,'QA income movement','${`QA-INCOME-${originalID}`}',NOW())`);

  const afterIncome = await read();
  expect.soft(afterIncome.total_revenue_kurus).toBe(before.total_revenue_kurus + 1750);
  const municipalityBefore = before.operator_shares.find((share) => share.operator.startsWith('İzmir Büyükşehir Belediyesi'));
  const municipalityAfterIncome = afterIncome.operator_shares.find((share) => share.operator.startsWith('İzmir Büyükşehir Belediyesi'));
  expect.soft(municipalityAfterIncome.share_kurus).toBe(municipalityBefore.share_kurus + 1750);

  const closed = await request.post(`${apiURL}/api/v1/settlements/close`, {
    headers: bearer(token), data: { period_id: periodID },
  });
  expect(closed.status()).toBe(200);

  const stornoPayload = { period_id: periodID, operator: 'belediye', amount_kurus: -500, reason: 'QA negative correction' };
  const storno = await request.post(`${apiURL}/api/v1/settlements/storno`, {
    headers: { ...bearer(token), 'Idempotency-Key': correctionKey }, data: stornoPayload,
  });
  expect(storno.status()).toBe(200);
  const stornoID = (await storno.json()).id;
  const repeated = await request.post(`${apiURL}/api/v1/settlements/storno`, {
    headers: { ...bearer(token), 'Idempotency-Key': correctionKey }, data: stornoPayload,
  });
  expect(repeated.status()).toBe(200);
  expect((await repeated.json()).id).toBe(stornoID);
  expect(scalar(`SELECT COUNT(*) FROM jw_settlement_ledger WHERE idempotency_key='${correctionKey}'`)).toBe('1');
  expect(scalar(`SELECT COUNT(*) FROM jw_settlement_ledger WHERE id='${originalID}' AND amount_kurus=1750 AND is_storno=false`)).toBe('1');

  const afterStorno = await read();
  expect.soft(afterStorno.total_revenue_kurus).toBe(before.total_revenue_kurus + 1250);
  const municipalityAfterStorno = afterStorno.operator_shares.find((share) => share.operator.startsWith('İzmir Büyükşehir Belediyesi'));
  expect.soft(municipalityAfterStorno.share_kurus).toBe(municipalityBefore.share_kurus + 1250);
});
