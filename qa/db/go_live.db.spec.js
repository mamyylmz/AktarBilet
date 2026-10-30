import { test, expect } from '@playwright/test';
import { apiURL, bearer, login, deviceAuthHeaders } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

function unique(prefix) {
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

test.describe('go-live database and transaction invariants', () => {
  test('DATA-02 validator event direction is constrained in PostgreSQL', () => {
    const key = unique('DB-BAD-DIRECTION');
    expect(() => sql(`BEGIN;
      INSERT INTO jw_validator_events(idempotency_key,validator_id,sequence_no,card_id,line_id,station_id,direction,tapped_at,fare_charged_kurus)
      VALUES ('${key}','VAL_500T_01',${Date.now()},'CARD_TAM_001','LINE_5','STAT_ESHOT_10177',9,NOW(),0);
      ROLLBACK;`)).toThrow();
  });

  test('DATA-03 financial mutations produce an append-only audit record', async ({ request }) => {
    const { token } = await login(request, 'admin');
    const key = unique('DB-AUDIT-RECHARGE');
    const before = Number(scalar('SELECT COUNT(*) FROM jw_audit_logs'));
    const response = await request.post(`${apiURL}/api/v1/cards/CARD_TAM_001/recharge`, {
      headers: { ...bearer(token), 'Idempotency-Key': key },
      data: { amount_kurus: 17, description: 'audit trail probe' }
    });
    expect(response.status()).toBe(200);
    expect(Number(scalar('SELECT COUNT(*) FROM jw_audit_logs'))).toBeGreaterThan(before);
  });

  test('CORE-04 out-of-order delivery yields the same fare as chronological delivery', async ({ request }) => {
    const suffix = `${Date.now()}${Math.floor(Math.random() * 1000)}`;
    const cardReverse = `QA_OOO_R_${suffix}`;
    const cardOrdered = `QA_OOO_O_${suffix}`;
    for (const card of [cardReverse, cardOrdered]) {
      sql(`INSERT INTO jw_cards(id,card_type,balance_kurus) VALUES ('${card}','tam',10000)`);
      sql(`INSERT INTO jw_ledger(card_id,entry_type,amount_kurus,balance_after_kurus,reference_id)
           VALUES ('${card}','dolum',10000,10000,'SEED-${card}')`);
    }
    const t1 = new Date(Date.now() - 60 * 60 * 1000).toISOString();
    const t2 = new Date(Date.now() - 30 * 60 * 1000).toISOString();
    const tap = (card, time, station, line) => {
      const validatorID = line === 'LINE_15' ? 'VAL_T1_01' : 'VAL_500T_01';
      const thisSeq = Number(scalar(`SELECT COALESCE(MAX(sequence_no), 0) + 1 FROM jw_device_requests WHERE validator_id='${validatorID}'`));
      const idemp = unique(`OOO-${card}`);
      const data = { validator_id: validatorID, sequence_no: thisSeq, card_id: card,
        line_id: line, station_id: station, direction: 1, tapped_at: time };
      return request.post(`${apiURL}/api/v1/validators/tap`, {
        headers: { ...deviceAuthHeaders(validatorID, thisSeq, data, idemp), 'Idempotency-Key': idemp },
        data
      });
    };
    expect((await tap(cardReverse, t2, 'STAT_ESHOT_10008', 'LINE_15')).status()).toBe(200);
    expect((await tap(cardReverse, t1, 'STAT_ESHOT_10177', 'LINE_5')).status()).toBe(200);
    expect((await tap(cardOrdered, t1, 'STAT_ESHOT_10177', 'LINE_5')).status()).toBe(200);
    expect((await tap(cardOrdered, t2, 'STAT_ESHOT_10008', 'LINE_15')).status()).toBe(200);
    expect(scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${cardReverse}'`))
      .toBe(scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${cardOrdered}'`));
  });

  test('CORE-02 one hundred concurrent recharges preserve ledger equality', async ({ request }) => {
    const { token } = await login(request, 'admin');
    const card = `QA_CONC_${Date.now()}`;
    sql(`INSERT INTO jw_cards(id,card_type,balance_kurus) VALUES ('${card}','tam',0)`);
    const responses = await Promise.all(Array.from({ length: 100 }, (_, i) =>
      request.post(`${apiURL}/api/v1/cards/${card}/recharge`, {
        headers: { ...bearer(token), 'Idempotency-Key': `${card}-${i}` },
        data: { amount_kurus: 1, description: 'concurrent QA recharge' }
      })
    ));
    expect(responses.every((response) => response.status() === 200)).toBe(true);
    expect(scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${card}'`)).toBe('100');
    expect(scalar(`SELECT COALESCE(SUM(amount_kurus),0) FROM jw_ledger WHERE card_id='${card}'`)).toBe('100');
  });

  test('CORE-05 settlement close is a one-way idempotent transition', async ({ request }) => {
    const { token } = await login(request, 'muhasebe');
    const period = scalar(`INSERT INTO jw_settlement_periods(period_name,start_date,end_date)
      VALUES ('${unique('DB-CLOSE')}',NOW()-INTERVAL '1 day',NOW()) RETURNING id`);
    const first = await request.post(`${apiURL}/api/v1/settlements/close`, {
      headers: bearer(token), data: { period_id: period }
    });
    const closedAt = scalar(`SELECT closed_at::text FROM jw_settlement_periods WHERE id='${period}'`);
    await new Promise((resolve) => setTimeout(resolve, 25));
    const second = await request.post(`${apiURL}/api/v1/settlements/close`, {
      headers: bearer(token), data: { period_id: period }
    });
    expect(first.status()).toBe(200);
    expect.soft([409, 422]).toContain(second.status());
    expect(scalar(`SELECT closed_at::text FROM jw_settlement_periods WHERE id='${period}'`)).toBe(closedAt);
  });

  test('CORE-05 storno retries create exactly one correction', async ({ request }) => {
    const { token } = await login(request, 'muhasebe');
    const period = scalar(`INSERT INTO jw_settlement_periods(period_name,start_date,end_date,is_closed,closed_at)
      VALUES ('${unique('DB-STORNO')}',NOW()-INTERVAL '1 day',NOW(),true,NOW()) RETURNING id`);
    const key = unique('DB-STORNO-IDEMP');
    const payload = { period_id: period, operator: 'belediye', amount_kurus: -123, reason: 'QA retry' };
    const first = await request.post(`${apiURL}/api/v1/settlements/storno`, {
      headers: { ...bearer(token), 'Idempotency-Key': key }, data: payload
    });
    const second = await request.post(`${apiURL}/api/v1/settlements/storno`, {
      headers: { ...bearer(token), 'Idempotency-Key': key }, data: payload
    });
    expect(first.status()).toBe(200);
    expect(second.status()).toBe(200);
    expect(scalar(`SELECT COUNT(*) FROM jw_settlement_ledger WHERE period_id='${period}' AND storno_reason='QA retry'`)).toBe('1');
  });
});
