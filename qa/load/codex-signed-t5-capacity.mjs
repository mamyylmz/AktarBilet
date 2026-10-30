import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { performance } from 'node:perf_hooks';
import { deviceSecret } from '../helpers/http.js';
import { scalar, sql } from '../helpers/db.js';

const apiURL = process.env.QA_API_URL || 'http://127.0.0.1:18080';
const validatorID = 'VAL_500T_01';
const durationSeconds = Number(process.env.QA_T5_SECONDS || 60);
const batchesPerSecond = Number(process.env.QA_T5_BATCHES_PER_SECOND || 15);
const batchSize = Number(process.env.QA_T5_BATCH_SIZE || 20);
const concurrency = Number(process.env.QA_T5_CONCURRENCY || 10);
const runID = `codex-${Date.now()}`;
const targetEvents = Math.ceil(durationSeconds * batchesPerSecond) * batchSize;
const secret = deviceSecret(validatorID);

if (!Number.isFinite(durationSeconds) || durationSeconds < 10 || durationSeconds > 180 || batchSize < 1 || batchSize > 50) {
  throw new Error('QA_T5_SECONDS must be 10..180 and QA_T5_BATCH_SIZE must be 1..50');
}

const funded = Number(scalar(`SELECT COUNT(*) FROM jw_cards WHERE id LIKE 'CARD_SIM_%' AND is_blacklisted=false AND balance_kurus=0`));
if (funded < targetEvents) throw new Error(`Insufficient zero-balance QA simulator cards: ${funded} < ${targetEvents}`);

sql(`WITH chosen AS (
       SELECT id FROM jw_cards WHERE id LIKE 'CARD_SIM_%' AND is_blacklisted=false AND balance_kurus=0 ORDER BY id LIMIT ${targetEvents}
     ), updated AS (
       UPDATE jw_cards c SET balance_kurus=100000 FROM chosen WHERE c.id=chosen.id RETURNING c.id
     )
     INSERT INTO jw_ledger(id,card_id,entry_type,amount_kurus,balance_after_kurus,reference_id,description,created_at)
     SELECT gen_random_uuid(),id,'dolum',100000,100000,'QA-T5-FUND:'||id,'Isolated signed T-5 QA card funding',NOW() FROM updated`);

const cardIDs = sql(`SELECT id FROM jw_cards WHERE id LIKE 'CARD_SIM_%' AND balance_kurus=100000 ORDER BY id LIMIT ${targetEvents}`).split('\n').filter(Boolean);
if (cardIDs.length !== targetEvents) throw new Error(`Funded fixture mismatch: ${cardIDs.length} != ${targetEvents}`);
const reportTokenResponse = await fetch(`${apiURL}/api/v1/auth/login`, {
  method: 'POST', headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email: 'muhasebe@aktarbilet.gov.tr', password: 'MuhasebePassword123!' }),
});
if (!reportTokenResponse.ok) throw new Error(`Accountant login failed: HTTP ${reportTokenResponse.status}`);
const reportToken = (await reportTokenResponse.json()).access_token;
async function readSummary() {
  const response = await fetch(`${apiURL}/api/v1/settlements`, { headers: { Authorization: `Bearer ${reportToken}` } });
  if (!response.ok) throw new Error(`Settlement report failed: HTTP ${response.status}`);
  const body = await response.json();
  return { totalTrips: body.total_trips, totalRevenueKurus: body.total_revenue_kurus, totalNetKurus: body.total_net_kurus };
}
const summaryBefore = await readSummary();
// Keep the values within JavaScript's exact integer range; prior verifier runs may use UnixNano int64 values.
let sequenceBase = Date.now() * 1000;
while (Number(scalar(`SELECT COUNT(*) FROM jw_device_requests WHERE validator_id='${validatorID}' AND sequence_no BETWEEN ${sequenceBase + 1} AND ${sequenceBase + targetEvents}`)) > 0) {
  sequenceBase += 1000000;
}
const latencies = [];
const errors = [];
let launchedBatches = 0;
let apiAcceptedEvents = 0;
let active = 0;
const start = performance.now();
const durationMs = durationSeconds * 1000;
const batchIntervalMs = 1000 / batchesPerSecond;
const pending = new Set();

function makeBatch(batchIndex) {
  const first = batchIndex * batchSize;
  const tappedAt = new Date().toISOString();
  return Array.from({ length: batchSize }, (_, offset) => {
    const index = first + offset;
    return {
      idempotency_key: `T5CAP:${runID}:${index}`,
      validator_id: validatorID,
      sequence_no: sequenceBase + index + 1,
      card_id: cardIDs[index],
      line_id: 'LINE_5',
      station_id: 'STAT_ESHOT_10177',
      direction: 1,
      tapped_at: tappedAt,
    };
  });
}

// Launch at a bounded rate and cap in-flight requests; drain active work before measuring final persistence.
while (performance.now() - start < durationMs) {
  const elapsed = performance.now() - start;
  if (active < concurrency && elapsed >= launchedBatches * batchIntervalMs) {
    const batchIndex = launchedBatches++;
    active += 1;
    const batch = makeBatch(batchIndex);
    const promise = (async () => {
      const body = JSON.stringify(batch);
      const timestamp = Math.floor(Date.now() / 1000);
      const nonce = `t5-${runID}-${batchIndex}`;
      const payloadHash = crypto.createHash('sha256').update(body).digest('hex');
      const signingSequence = batch.at(-1).sequence_no;
      const message = `${validatorID}:${signingSequence}:${timestamp}:${nonce}:${payloadHash}`;
      const signature = crypto.createHmac('sha256', secret).update(message).digest('hex');
      const started = performance.now();
      try {
        const response = await fetch(`${apiURL}/api/v1/validators/batch-sync`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-Validator-ID': validatorID,
            'X-Device-Timestamp': String(timestamp), 'X-Device-Nonce': nonce, 'X-Device-Signature': signature },
          body,
        });
        const result = await response.json().catch(() => ({}));
        if (!response.ok || result.processed_count !== batch.length) errors.push({ batchIndex, status: response.status, code: result?.error?.code || 'UNEXPECTED_RESPONSE' });
        else apiAcceptedEvents += result.processed_count;
      } catch (error) {
        errors.push({ batchIndex, code: error?.name || 'NETWORK_ERROR' });
      } finally {
        latencies.push(performance.now() - started);
        active -= 1;
      }
    })();
    pending.add(promise);
  }
  await new Promise((resolve) => setTimeout(resolve, 2));
}
await Promise.all(pending);
const elapsedSeconds = (performance.now() - start) / 1000;
const submittedEvents = launchedBatches * batchSize;
const firstSequence = sequenceBase + 1;
const lastSequence = sequenceBase + submittedEvents;
const runPattern = `T5CAP:${runID}:%`;
const statusRows = sql(`SELECT status||'|'||COUNT(*) FROM jw_validator_events WHERE idempotency_key LIKE '${runPattern}' GROUP BY status ORDER BY status`);
const persistedEvents = Number(scalar(`SELECT COUNT(*) FROM jw_validator_events WHERE idempotency_key LIKE '${runPattern}'`));
const persistedFareKurus = Number(scalar(`SELECT COALESCE(SUM(fare_charged_kurus),0) FROM jw_validator_events WHERE idempotency_key LIKE '${runPattern}' AND status='PROCESSED'`));
const ledgerCount = Number(scalar(`SELECT COUNT(*) FROM jw_ledger WHERE reference_id LIKE 'SYNC:${validatorID}:%' AND reference_id::text IN (SELECT 'SYNC:${validatorID}:'||sequence_no FROM jw_device_requests WHERE validator_id='${validatorID}' AND sequence_no BETWEEN ${firstSequence} AND ${lastSequence})`));
const summaryAfter = await readSummary();
latencies.sort((a, b) => a - b);
const percentile = (ratio) => latencies.length ? Number(latencies[Math.min(latencies.length - 1, Math.floor(latencies.length * ratio))].toFixed(2)) : null;
const eventsPerSecond = Number((persistedEvents / elapsedSeconds).toFixed(2));
const reportDeltaKurus = summaryAfter.totalRevenueKurus - summaryBefore.totalRevenueKurus;
const expectedRevenueDeltaKurus = persistedFareKurus;
const reportAccurate = reportDeltaKurus === expectedRevenueDeltaKurus;
const result = {
  runID, apiURL, validatorID, durationSeconds, elapsedSeconds: Number(elapsedSeconds.toFixed(3)),
  launchRateBatchesPerSecond: batchesPerSecond, batchSize, concurrency,
  generatedEvents: targetEvents, sentEvents: submittedEvents, apiAcceptedEvents,
  persistedEvents, processedEvents: Number((statusRows.match(/PROCESSED\|(\d+)/)?.[1]) || 0),
  rejectedEvents: Number((statusRows.match(/REJECTED\|(\d+)/)?.[1]) || 0), pendingEvents: Number((statusRows.match(/PENDING\|(\d+)/)?.[1]) || 0),
  ledgerDebitRows: ledgerCount, persistedFareKurus, failureBatches: errors.length,
  errors: errors.slice(0, 20), errorRate: Number((errors.length / Math.max(launchedBatches, 1)).toFixed(6)),
  eventsPerSecond, hourlyEquivalent: Math.round(eventsPerSecond * 3600),
  batchLatencyMs: { p50: percentile(0.50), p95: percentile(0.95), p99: percentile(0.99) },
  summaryBefore, summaryAfter, expectedRevenueDeltaKurus, reportDeltaKurus, reportAccurate,
  persistenceReconciles: persistedEvents === submittedEvents && ledgerCount === persistedEvents,
  capacityVerdict: eventsPerSecond >= 278 && errors.length / Math.max(launchedBatches, 1) <= 0.01 && persistedEvents === submittedEvents && ledgerCount === persistedEvents && reportAccurate ? 'T5_CAPACITY_VERIFIED' : 'T5_CAPACITY_FAIL',
};
fs.mkdirSync(path.resolve('artifacts'), { recursive: true });
const artifactPath = path.resolve('artifacts', `codex-signed-t5-${runID}.json`);
fs.writeFileSync(artifactPath, `${JSON.stringify(result, null, 2)}\n`);
console.log(JSON.stringify({ ...result, artifactPath }, null, 2));
if (result.capacityVerdict !== 'T5_CAPACITY_VERIFIED') process.exitCode = 1;
