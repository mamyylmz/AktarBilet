import { randomUUID } from 'node:crypto';
import { test, expect } from '@playwright/test';
import { apiURL, bearer, login } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

const seededPeriod = '00000000-0000-0000-0000-000000000001';

test('settlement audit failures roll back; close is serialized; reopen rejects bad audit SQL', async ({ request }) => {
  test.setTimeout(45_000);
  const { token: accountant } = await login(request, 'muhasebe');
  const { token: admin } = await login(request, 'admin');

  const triggerFunction = `CREATE OR REPLACE FUNCTION qa_reject_close_audit() RETURNS trigger LANGUAGE plpgsql AS $$
    BEGIN IF NEW.action = 'SETTLEMENT_CLOSE' THEN RAISE EXCEPTION 'qa injected audit failure'; END IF; RETURN NEW; END; $$`;
  const closeFailurePeriod = randomUUID();
  const concurrentPeriod = randomUUID();
  const reason = `QA reopen ${randomUUID()}`;

  sql(`INSERT INTO jw_settlement_periods(id,period_name,start_date,end_date,is_closed)
       VALUES ('${closeFailurePeriod}','QA-ATOMIC-${closeFailurePeriod}',NOW()-INTERVAL '1 day',NOW(),false),
              ('${concurrentPeriod}','QA-CONCURRENT-${concurrentPeriod}',NOW()-INTERVAL '1 day',NOW(),false)`);
  sql('DROP TRIGGER IF EXISTS qa_reject_close_audit ON jw_audit_logs');
  sql(triggerFunction);
  sql(`CREATE TRIGGER qa_reject_close_audit BEFORE INSERT ON jw_audit_logs FOR EACH ROW EXECUTE FUNCTION qa_reject_close_audit()`);
  try {
    const rejected = await request.post(`${apiURL}/api/v1/settlements/close`, {
      headers: bearer(accountant), data: { period_id: closeFailurePeriod }
    });
    expect(rejected.status()).toBe(500);
    expect(scalar(`SELECT is_closed::text FROM jw_settlement_periods WHERE id='${closeFailurePeriod}'`)).toBe('false');
    expect(scalar(`SELECT COUNT(*) FROM jw_audit_logs WHERE action='SETTLEMENT_CLOSE' AND record_id='${closeFailurePeriod}'`)).toBe('0');
  } finally {
    sql('DROP TRIGGER IF EXISTS qa_reject_close_audit ON jw_audit_logs');
    sql('DROP FUNCTION IF EXISTS qa_reject_close_audit()');
  }

  const simultaneous = await Promise.all([
    request.post(`${apiURL}/api/v1/settlements/close`, { headers: bearer(accountant), data: { period_id: concurrentPeriod } }),
    request.post(`${apiURL}/api/v1/settlements/close`, { headers: bearer(accountant), data: { period_id: concurrentPeriod } }),
  ]);
  expect(simultaneous.map((response) => response.status()).sort()).toEqual([200, 409]);
  expect(scalar(`SELECT COUNT(*) FROM jw_audit_logs WHERE action='SETTLEMENT_CLOSE' AND record_id='${concurrentPeriod}'`)).toBe('1');

  const seededWasClosed = scalar(`SELECT is_closed::text FROM jw_settlement_periods WHERE id='${seededPeriod}'`) === 'true';
  sql(`UPDATE jw_settlement_periods SET is_closed=false,closed_at=NULL,closed_by=NULL WHERE id='${seededPeriod}'`);
  const closed = await request.post(`${apiURL}/api/v1/settlements/close`, {
    headers: bearer(accountant), data: { period_id: seededPeriod }
  });
  expect(closed.status()).toBe(200);
  const closeAuditCount = scalar(`SELECT COUNT(*) FROM jw_audit_logs WHERE action='SETTLEMENT_CLOSE' AND record_id='${seededPeriod}'`);
  expect(closeAuditCount).not.toBe('0');

  const reopened = await request.post(`${apiURL}/api/v1/settlements/reopen`, {
    headers: bearer(admin), data: { period_name: '2026-EKIM', reason }
  });
  expect(reopened.status(), await reopened.text()).toBe(200);
  expect(scalar(`SELECT is_closed::text FROM jw_settlement_periods WHERE id='${seededPeriod}'`)).toBe('false');
  expect(scalar(`SELECT COUNT(*) FROM jw_audit_logs WHERE action='SETTLEMENT_REOPEN' AND record_id='${seededPeriod}' AND new_values->>'reason'='${reason}'`)).toBe('1');

  if (seededWasClosed) {
    sql(`UPDATE jw_settlement_periods SET is_closed=true WHERE id='${seededPeriod}'`);
  } else {
    sql(`UPDATE jw_settlement_periods SET is_closed=false,closed_at=NULL,closed_by=NULL WHERE id='${seededPeriod}'`);
  }
});
