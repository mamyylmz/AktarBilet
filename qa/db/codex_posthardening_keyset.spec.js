import { randomUUID } from 'node:crypto';
import { test, expect } from '@playwright/test';
import { apiURL, bearer, login } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

test('DB keysets keep timestamp ties, exact-page boundaries, and reject invalid cursors', async ({ request }) => {
  test.setTimeout(60_000);
  const { token } = await login(request, 'admin');
  const timestamp = '2099-12-31T23:59:59Z';
  const cardID = 'CARD_YOLCU_001';
  const inspectorID = scalar("SELECT id FROM jw_tf_users WHERE email='seed.sedir@ddo-case.invalid'");
  const anomalyIDs = Array.from({ length: 4 }, () => randomUUID());
  const penaltyIDs = Array.from({ length: 5 }, () => randomUUID());

  sql(anomalyIDs.map((id) => `INSERT INTO jw_anomalies(id,card_id,anomaly_type,description,detected_at)
    VALUES ('${id}','${cardID}','CLONE_CARD_DETECTED','QA keyset fixture','${timestamp}')`).join(';'));
  sql(penaltyIDs.map((id) => `INSERT INTO jw_penalties(id,inspector_id,card_id,penalty_amount_kurus,reason,issued_at,notes)
    VALUES ('${id}','${inspectorID}','${cardID}',1000,'diger','${timestamp}','QA keyset fixture')`).join(';'));

  const pages = async (path, keyset) => {
    let cursor = '';
    const ids = [];
    const hasMoreValues = [];
    const pageSizes = [];
    for (let i = 0; i < 100; i += 1) {
      const url = `${apiURL}${path}?limit=2${cursor ? `&cursor=${cursor}` : ''}`;
      const response = await request.get(url, { headers: bearer(token) });
      expect(response.status()).toBe(200);
      const rows = await response.json();
      pageSizes.push(rows.length);
      ids.push(...rows.map((row) => row.id));
      hasMoreValues.push(response.headers()['x-has-more']);
      cursor = response.headers()['x-next-cursor'] || '';
      if (response.headers()['x-has-more'] !== 'true') break;
    }
    expect.soft(ids.filter((id) => keyset.includes(id)).sort()).toEqual([...keyset].sort());
    expect.soft(new Set(ids).size).toBe(ids.length);
    expect.soft(pageSizes.at(-1)).toBeGreaterThan(0);
    return { hasMoreValues, pageSizes };
  };

  const anomalyMore = await pages('/api/v1/anomalies', anomalyIDs);
  const penaltyMore = await pages('/api/v1/penalties', penaltyIDs);
  console.log(JSON.stringify({ anomalyIDs, anomalyMore, penaltyIDs, penaltyMore }));
  expect.soft(anomalyMore.hasMoreValues.at(-1)).toBe('false');
  expect.soft(penaltyMore.hasMoreValues.at(-1)).toBe('false');

  for (const path of ['/api/v1/anomalies', '/api/v1/penalties']) {
    const malformed = await request.get(`${apiURL}${path}?limit=2&cursor=not-a-valid-cursor`, { headers: bearer(token) });
    expect.soft(malformed.status(), `${path} malformed cursor should be rejected`).toBe(400);
  }
});
