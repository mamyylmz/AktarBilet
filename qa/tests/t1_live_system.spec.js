import { test, expect } from '@playwright/test';
import { apiURL, bearer, deviceAuthHeaders, login } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

function unique(prefix) {
  return `${prefix}_${Date.now()}_${Math.random().toString(16).slice(2)}`;
}

function createCardWithType(cardId, cardType, balanceKurus) {
  sql(`INSERT INTO jw_cards(id, card_type, balance_kurus, daily_taps_count, monthly_capping_spent_kurus) 
       VALUES ('${cardId}', '${cardType}', ${balanceKurus}, 0, 0)`);
  if (balanceKurus > 0) {
    sql(`INSERT INTO jw_ledger(card_id, entry_type, amount_kurus, balance_after_kurus, reference_id, description)
         VALUES ('${cardId}', 'dolum', ${balanceKurus}, ${balanceKurus}, 'SEED-${cardId}', 't1 live seed')`);
  }
}

function assertLedgerEquality(cardId) {
  const cardBal = scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${cardId}'`);
  const ledgerSum = scalar(`SELECT COALESCE(SUM(amount_kurus), 0) FROM jw_ledger WHERE card_id='${cardId}'`);
  expect(cardBal, `Card ${cardId} balance must equal sum of ledger`).toBe(ledgerSum);
}

function sendTap(request, { validatorId, sequenceNo, cardId, lineId, stationId, direction = 1, tappedAt, idempKey }) {
  const payload = {
    validator_id: validatorId,
    sequence_no: sequenceNo,
    card_id: cardId,
    line_id: lineId,
    station_id: stationId,
    direction: direction,
    tapped_at: tappedAt || new Date().toISOString()
  };
  const body = JSON.stringify(payload);
  const key = idempKey || `IDEMP-${validatorId}-${sequenceNo}`;
  const authHeaders = deviceAuthHeaders(validatorId, sequenceNo, body, `nonce-${key}`);

  return request.post(`${apiURL}/api/v1/validators/tap`, {
    headers: {
      ...authHeaders,
      'Idempotency-Key': key,
      'Content-Type': 'application/json'
    },
    data: body
  });
}

test.describe('T-1 Fare Engine Live Database & API Verification', () => {

  // ==========================================================================
  // T1-01: Kart Türleri ve Başlangıç Bakiyesi
  // ==========================================================================
  test('T1-01: Negative tolerance floor (-50 TL) is preserved and excessive deficit rejected', async ({ request }) => {
    const cardId = unique('CARD_NEG');
    // Start with 500 kuruş (5 TL)
    createCardWithType(cardId, 'tam', 500);

    // Ride 1: 2000 kuruş -> Balance becomes -1500 kuruş (Within -50 TL tolerance)
    const baseSeq = Date.now();
    const res1 = await sendTap(request, {
      validatorId: 'VAL_500T_01', sequenceNo: baseSeq, cardId,
      lineId: 'LINE_5', stationId: 'STAT_ESHOT_10177', direction: 1
    });
    expect(res1.status()).toBe(200);
    assertLedgerEquality(cardId);
    expect(scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${cardId}'`)).toBe('-1500');

    // Ride 2: 2000 kuruş -> Balance would be -3500 kuruş (Within -50 TL tolerance)
    const res2 = await sendTap(request, {
      validatorId: 'VAL_14ES_01', sequenceNo: baseSeq + 1, cardId,
      lineId: 'LINE_10', stationId: 'STAT_ESHOT_10145', direction: 1
    });
    expect(res2.status()).toBe(200);
    assertLedgerEquality(cardId);

    // Blacklist check: Once card goes into negative tolerance, does it get blocked if pushed past -5000?
    sql(`UPDATE jw_cards SET balance_kurus = -4500 WHERE id='${cardId}'`);
    sql(`INSERT INTO jw_ledger(card_id, entry_type, amount_kurus, balance_after_kurus, reference_id, description)
         VALUES ('${cardId}', 'binis', -3000, -4500, '${unique('REF')}', 'manual test deficit')`);

    // Next tap of 2000 kuruş would drop balance to -6500 kuruş (< -5000 kuruş) -> Must reject!
    const res3 = await sendTap(request, {
      validatorId: 'VAL_T1_01', sequenceNo: baseSeq + 2, cardId,
      lineId: 'LINE_15', stationId: 'STAT_ESHOT_10008', direction: 1
    });
    // Rejection: either HTTP 200 with status "REJECTED" or error status
    if (res3.status() === 200) {
      const data = await res3.json();
      const eventStatus = data.event?.status || data.status;
      const rejReason = data.event?.rejection_reason || data.rejection_reason;
      expect(eventStatus).toBe('REJECTED');
      expect(rejReason).toContain('Yetersiz bakiye');
    } else {
      expect([400, 402, 403, 422]).toContain(res3.status());
    }
  });

  // ==========================================================================
  // T1-04: Aynı Hatta Geri Dönüş ve Farklı Hatlar
  // ==========================================================================
  test('T1-04: Same line return charges full fare while different line gets transfer discount', async ({ request }) => {
    const cardId = unique('CARD_REV');
    createCardWithType(cardId, 'tam', 20000);
    const baseSeq = Date.now();
    const t0 = new Date();

    // 1. Initial ride on Line 500T (Direction 1) -> 2000 kuruş
    const res1 = await sendTap(request, {
      validatorId: 'VAL_500T_01', sequenceNo: baseSeq, cardId,
      lineId: 'LINE_5', stationId: 'STAT_ESHOT_10177', direction: 1,
      tappedAt: t0.toISOString()
    });
    expect(res1.status()).toBe(200);
    expect(scalar(`SELECT fare_charged_kurus FROM jw_validator_events WHERE card_id='${cardId}' ORDER BY tapped_at DESC LIMIT 1`)).toBe('2000');

    // 2. 20 min later: Transfer to Line 14ES (Direction 1) -> %50 discount (1000 kuruş)
    const t1 = new Date(t0.getTime() + 20 * 60 * 1000);
    const res2 = await sendTap(request, {
      validatorId: 'VAL_14ES_01', sequenceNo: baseSeq + 1, cardId,
      lineId: 'LINE_10', stationId: 'STAT_ESHOT_10145', direction: 1,
      tappedAt: t1.toISOString()
    });
    expect(res2.status()).toBe(200);
    expect(scalar(`SELECT fare_charged_kurus FROM jw_validator_events WHERE card_id='${cardId}' ORDER BY tapped_at DESC LIMIT 1`)).toBe('1000');
    expect(scalar(`SELECT is_transfer FROM jw_validator_events WHERE card_id='${cardId}' ORDER BY tapped_at DESC LIMIT 1`)).toBe('t');

    // 3. 40 min later: Same line 500T return -> Reverse Direction (2000 kuruş, NO DISCOUNT!)
    const t2 = new Date(t0.getTime() + 40 * 60 * 1000);
    const res3 = await sendTap(request, {
      validatorId: 'VAL_500T_01', sequenceNo: baseSeq + 2, cardId,
      lineId: 'LINE_5', stationId: 'STAT_ESHOT_10177', direction: 2,
      tappedAt: t2.toISOString()
    });
    expect(res3.status()).toBe(200);
    expect(scalar(`SELECT fare_charged_kurus FROM jw_validator_events WHERE card_id='${cardId}' ORDER BY tapped_at DESC LIMIT 1`)).toBe('2000');
    expect(scalar(`SELECT is_transfer FROM jw_validator_events WHERE card_id='${cardId}' ORDER BY tapped_at DESC LIMIT 1`)).toBe('f');
  });

  // ==========================================================================
  // T1-08: Sırasız Olay Determinizmi (3 Olayın Ters Sırada Gelmesi)
  // ==========================================================================
  test('T1-08: 3 events arriving in reverse order must result in exact same total debits as chronological', async ({ request }) => {
    // We will run TWO cards:
    // Card A: Chronological delivery (E1, then E2, then E3)
    // Card B: Reverse delivery (E3, then E2, then E1)
    const cardChrono = unique('CARD_CHRONO');
    const cardReverse = unique('CARD_REVERSE');
    createCardWithType(cardChrono, 'tam', 20000);
    createCardWithType(cardReverse, 'tam', 20000);

    const tBase = new Date(Date.now() - 3600 * 1000); // 1 hour ago
    const t1 = new Date(tBase.getTime());
    const t2 = new Date(tBase.getTime() + 20 * 60 * 1000);
    const t3 = new Date(tBase.getTime() + 40 * 60 * 1000);

    const baseSeq = Date.now();

    // --- CARD A: Chronological Delivery ---
    // E1
    await sendTap(request, {
      validatorId: 'VAL_500T_01', sequenceNo: baseSeq + 1, cardId: cardChrono,
      lineId: 'LINE_5', stationId: 'STAT_ESHOT_10177', direction: 1, tappedAt: t1.toISOString()
    });
    // E2
    await sendTap(request, {
      validatorId: 'VAL_14ES_01', sequenceNo: baseSeq + 2, cardId: cardChrono,
      lineId: 'LINE_10', stationId: 'STAT_ESHOT_10145', direction: 1, tappedAt: t2.toISOString()
    });
    // E3
    await sendTap(request, {
      validatorId: 'VAL_T1_01', sequenceNo: baseSeq + 3, cardId: cardChrono,
      lineId: 'LINE_15', stationId: 'STAT_ESHOT_10008', direction: 1, tappedAt: t3.toISOString()
    });

    assertLedgerEquality(cardChrono);
    const chronoBalance = scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${cardChrono}'`);
    // Chronological balance: 20000 - 2000 - 1000 - 500 = 16500 kuruş
    expect(chronoBalance, 'Card A chronological balance').toBe('16500');

    // --- CARD B: Reverse Delivery (E3 first, then E2, then E1) ---
    // Send E3 (at t3)
    await sendTap(request, {
      validatorId: 'VAL_T1_01', sequenceNo: baseSeq + 10, cardId: cardReverse,
      lineId: 'LINE_15', stationId: 'STAT_ESHOT_10008', direction: 1, tappedAt: t3.toISOString()
    });
    // Send E2 (at t2)
    await sendTap(request, {
      validatorId: 'VAL_14ES_01', sequenceNo: baseSeq + 11, cardId: cardReverse,
      lineId: 'LINE_10', stationId: 'STAT_ESHOT_10145', direction: 1, tappedAt: t2.toISOString()
    });
    // Send E1 (at t1)
    await sendTap(request, {
      validatorId: 'VAL_500T_01', sequenceNo: baseSeq + 12, cardId: cardReverse,
      lineId: 'LINE_5', stationId: 'STAT_ESHOT_10177', direction: 1, tappedAt: t1.toISOString()
    });

    assertLedgerEquality(cardReverse);
    const reverseBalance = scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${cardReverse}'`);

    // In a truly deterministic fare engine, reverse delivery must yield identical balance: 16500 kuruş!
    expect(reverseBalance, `[FAIL T1-08] Reverse delivery balance (${reverseBalance}) must equal chronological balance (${chronoBalance})`).toBe(chronoBalance);
  });

  // ==========================================================================
  // Concurrency: 10 Parallel Taps on Single Card
  // ==========================================================================
  test('Concurrency: 10 parallel taps on single card preserve ledger equality without lost updates', async ({ request }) => {
    const cardId = unique('CARD_PARALLEL');
    createCardWithType(cardId, 'tam', 50000);
    const baseSeq = Date.now();

    const promises = Array.from({ length: 10 }, (_, i) => {
      return sendTap(request, {
        validatorId: 'VAL_500T_01',
        sequenceNo: baseSeq + i,
        cardId,
        lineId: 'LINE_5',
        stationId: 'STAT_ESHOT_10177',
        direction: 1,
        idempKey: `IDEMP_PARALLEL_${cardId}_${i}`
      });
    });

    const responses = await Promise.all(promises);
    for (const res of responses) {
      expect(res.status()).toBe(200);
    }

    assertLedgerEquality(cardId);
    expect(scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE card_id='${cardId}'`)).toBe('10');
  });

  // ==========================================================================
  // T1-12: Duplicate Idempotency on Real API
  // ==========================================================================
  test('T1-12: Replaying identical tap produces identical HTTP response and zero extra ledger mutation', async ({ request }) => {
    const cardId = unique('CARD_IDEMP');
    createCardWithType(cardId, 'tam', 10000);
    const seq = Date.now();
    const key = `KEY_${cardId}`;

    const payload = {
      validator_id: 'VAL_500T_01', sequence_no: seq, card_id: cardId,
      line_id: 'LINE_5', station_id: 'STAT_ESHOT_10177', direction: 1,
      tapped_at: new Date().toISOString()
    };
    const body = JSON.stringify(payload);
    const headers = {
      ...deviceAuthHeaders('VAL_500T_01', seq, body, `nonce-${key}`),
      'Idempotency-Key': key,
      'Content-Type': 'application/json'
    };

    const res1 = await request.post(`${apiURL}/api/v1/validators/tap`, { headers, data: body });
    expect(res1.status()).toBe(200);

    const balAfterFirst = scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${cardId}'`);
    const ledgerCountFirst = scalar(`SELECT COUNT(*) FROM jw_ledger WHERE card_id='${cardId}'`);

    // Retry with EXACT SAME request, exact same headers (same timestamp, signature, nonce)
    const res2 = await request.post(`${apiURL}/api/v1/validators/tap`, { headers, data: body });
    expect(res2.status()).toBe(200);

    const balAfterSecond = scalar(`SELECT balance_kurus FROM jw_cards WHERE id='${cardId}'`);
    const ledgerCountSecond = scalar(`SELECT COUNT(*) FROM jw_ledger WHERE card_id='${cardId}'`);

    expect(balAfterSecond).toBe(balAfterFirst);
    expect(ledgerCountSecond).toBe(ledgerCountFirst);
    assertLedgerEquality(cardId);
  });
});
