import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { test, expect } from '@playwright/test';
import { apiURL, bearer, login } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

const project = process.env.QA_PROJECT_NAME || 'aktarbilet-qa';
const composeFile = 'docker-compose.qa.yml';

async function restartBackend(request) {
  execFileSync('docker', ['compose', '--project-name', project, '--file', composeFile, 'restart', 'backend'], {
    cwd: process.cwd(), encoding: 'utf8'
  });
  for (let attempt = 0; attempt < 40; attempt += 1) {
    try {
      if ((await request.get(`${apiURL}/health`, { timeout: 1000 })).status() === 200) return;
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  throw new Error('Backend did not become healthy after restart');
}

test('settlement close/reopen/storno persist across restart and seed reruns', async ({ request }) => {
  test.setTimeout(90_000);
  const { token: accountant } = await login(request, 'muhasebe');
  const { token: admin } = await login(request, 'admin');
  const { token: passenger } = await login(request, 'yolcu');
  const seededPeriod = '00000000-0000-0000-0000-000000000001';

  const seedState = () => scalar(`SELECT is_closed::text || ':' || COALESCE(closed_at::text,'null') FROM jw_settlement_periods WHERE id='${seededPeriod}'`);
  if (seedState().startsWith('true:')) {
    const setupReopen = await request.post(`${apiURL}/api/v1/settlements/reopen`, {
      headers: bearer(admin), data: { period_name: '2026-EKIM', reason: 'QA fixture setup' }
    });
    expect(setupReopen.status()).toBe(200);
  }

  const auditCount = (action) => Number(scalar(`SELECT COUNT(*) FROM jw_audit_logs WHERE action='${action}' AND record_id='${seededPeriod}'`));
  const beforeUnauthorized = seedState();
  const passengerClose = await request.post(`${apiURL}/api/v1/settlements/close`, {
    headers: bearer(passenger), data: { period_id: seededPeriod }
  });
  expect(passengerClose.status()).toBe(403);
  expect(seedState()).toBe(beforeUnauthorized);

  const beforeCloseAudit = auditCount('SETTLEMENT_CLOSE');
  const closed = await request.post(`${apiURL}/api/v1/settlements/close`, {
    headers: bearer(accountant), data: { period_id: seededPeriod }
  });
  expect(closed.status()).toBe(200);
  const closedAt = scalar(`SELECT closed_at::text FROM jw_settlement_periods WHERE id='${seededPeriod}'`);
  expect(scalar(`SELECT is_closed::text FROM jw_settlement_periods WHERE id='${seededPeriod}'`)).toBe('true');
  expect(closedAt).not.toBe('');
  expect(auditCount('SETTLEMENT_CLOSE')).toBeGreaterThan(beforeCloseAudit);

  const duplicateClose = await request.post(`${apiURL}/api/v1/settlements/close`, {
    headers: { ...bearer(accountant), 'Idempotency-Key': `dup-close-${randomUUID()}` },
    data: { period_id: seededPeriod }
  });
  expect(duplicateClose.status()).toBe(409);
  expect(scalar(`SELECT closed_at::text FROM jw_settlement_periods WHERE id='${seededPeriod}'`)).toBe(closedAt);

  await restartBackend(request);
  expect(scalar(`SELECT is_closed::text FROM jw_settlement_periods WHERE id='${seededPeriod}'`)).toBe('true');
  expect(scalar(`SELECT closed_at::text FROM jw_settlement_periods WHERE id='${seededPeriod}'`)).toBe(closedAt);

  const reopenReason = `Independent QA reopen ${randomUUID()}`;
  const beforeReopenAudit = auditCount('SETTLEMENT_REOPEN');
  const reopened = await request.post(`${apiURL}/api/v1/settlements/reopen`, {
    headers: bearer(admin), data: { period_name: '2026-EKIM', reason: reopenReason }
  });
  expect(reopened.status()).toBe(200);
  expect(scalar(`SELECT is_closed::text || ':' || COALESCE(closed_at::text,'null') FROM jw_settlement_periods WHERE id='${seededPeriod}'`)).toBe('false:null');
  expect(auditCount('SETTLEMENT_REOPEN')).toBeGreaterThan(beforeReopenAudit);
  expect(scalar(`SELECT COUNT(*) FROM jw_audit_logs WHERE action='SETTLEMENT_REOPEN' AND record_id='${seededPeriod}' AND new_values->>'reason'='${reopenReason}'`)).toBe('1');

  await restartBackend(request);
  expect(seedState()).toBe('false:null');

  const openPeriod = randomUUID();
  sql(`INSERT INTO jw_settlement_periods(id,period_name,start_date,end_date,is_closed)
       VALUES ('${openPeriod}','QA-STORNO-OPEN-${openPeriod}',NOW()-INTERVAL '1 day',NOW(),false)`);
  const openStorno = await request.post(`${apiURL}/api/v1/settlements/storno`, {
    headers: bearer(accountant), data: { period_id: openPeriod, operator: 'belediye', amount_kurus: -500, reason: 'Must be rejected' }
  });
  expect(openStorno.status()).toBe(400);
  expect(scalar(`SELECT COUNT(*) FROM jw_settlement_ledger WHERE period_id='${openPeriod}'`)).toBe('0');

  const closedPeriod = randomUUID();
  const originalLedgerID = randomUUID();
  sql(`INSERT INTO jw_settlement_periods(id,period_name,start_date,end_date,is_closed)
       VALUES ('${closedPeriod}','QA-STORNO-CLOSED-${closedPeriod}',NOW()-INTERVAL '1 day',NOW(),false);
       INSERT INTO jw_settlement_ledger(id,period_id,operator,amount_kurus,is_storno,created_at)
       VALUES ('${originalLedgerID}','${closedPeriod}','belediye',1750,false,NOW())`);
  expect((await request.post(`${apiURL}/api/v1/settlements/close`, {
    headers: bearer(accountant), data: { period_id: closedPeriod }
  })).status()).toBe(200);

  const stornoKey = `storno-${randomUUID()}`;
  const stornoPayload = { period_id: closedPeriod, operator: 'belediye', amount_kurus: -500, reason: 'QA closed-period correction' };
  const storno = await request.post(`${apiURL}/api/v1/settlements/storno`, {
    headers: { ...bearer(accountant), 'Idempotency-Key': stornoKey }, data: stornoPayload
  });
  expect(storno.status()).toBe(200);
  const stornoID = (await storno.json()).id;
  expect(scalar(`SELECT COUNT(*) FROM jw_settlement_ledger WHERE id='${originalLedgerID}' AND amount_kurus=1750 AND is_storno=false`)).toBe('1');
  expect(scalar(`SELECT COUNT(*) FROM jw_settlement_ledger WHERE id='${stornoID}' AND amount_kurus=-500 AND is_storno=true AND idempotency_key='${stornoKey}'`)).toBe('1');
  expect(scalar(`SELECT SUM(amount_kurus)::text FROM jw_settlement_ledger WHERE period_id='${closedPeriod}'`)).toBe('1250');
  expect(scalar(`SELECT COUNT(*) FROM jw_audit_logs WHERE action='SETTLEMENT_STORNO' AND record_id='${stornoID}'`)).toBe('1');

  const duplicateStorno = await request.post(`${apiURL}/api/v1/settlements/storno`, {
    headers: { ...bearer(accountant), 'Idempotency-Key': stornoKey },
    data: { ...stornoPayload, amount_kurus: -9999, reason: 'Different payload under same key' }
  });
  expect(duplicateStorno.status()).toBe(200);
  expect(scalar(`SELECT COUNT(*) FROM jw_settlement_ledger WHERE idempotency_key='${stornoKey}'`)).toBe('1');

  const beforeDeniedStorno = scalar(`SELECT COUNT(*) FROM jw_settlement_ledger WHERE period_id='${closedPeriod}'`);
  const deniedStorno = await request.post(`${apiURL}/api/v1/settlements/storno`, {
    headers: bearer(passenger), data: stornoPayload
  });
  expect(deniedStorno.status()).toBe(403);
  expect(scalar(`SELECT COUNT(*) FROM jw_settlement_ledger WHERE period_id='${closedPeriod}'`)).toBe(beforeDeniedStorno);

  const badID = await request.post(`${apiURL}/api/v1/settlements/close`, {
    headers: bearer(accountant), data: { period_id: 'not-a-uuid' }
  });
  expect(badID.status()).toBe(400);
  const newPeriodName = `QA-PERIOD-${randomUUID()}`;
  const reopenedNewPeriod = await request.post(`${apiURL}/api/v1/settlements/reopen`, {
    headers: bearer(admin), data: { period_name: newPeriodName, reason: 'QA dynamic period acceptance' }
  });
  expect(reopenedNewPeriod.status()).toBe(200);
  const newPeriod = await reopenedNewPeriod.json();
  expect(newPeriod.period_name).toBe(newPeriodName);
  expect(scalar(`SELECT COUNT(*) FROM jw_settlement_periods WHERE period_name='${newPeriodName}' AND is_closed=false`)).toBe('1');
  expect(scalar(`SELECT COUNT(*) FROM jw_audit_logs WHERE action='SETTLEMENT_REOPEN' AND record_id='${newPeriod.period_id}'`)).toBe('1');

  const closeBeforeDefaultReason = await request.post(`${apiURL}/api/v1/settlements/close`, {
    headers: bearer(accountant), data: { period_id: seededPeriod }
  });
  expect(closeBeforeDefaultReason.status()).toBe(200);
  const emptyReasonReopen = await request.post(`${apiURL}/api/v1/settlements/reopen`, {
    headers: bearer(admin), data: { period_name: '2026-EKIM', reason: '' }
  });
  expect(emptyReasonReopen.status()).toBe(200);
  expect(scalar(`SELECT COUNT(*) FROM jw_audit_logs WHERE action='SETTLEMENT_REOPEN' AND record_id='${seededPeriod}' AND new_values->>'reason'='Yetkili mali işler denetimi kapsamında yeniden açma'`)).toBe('1');
});
