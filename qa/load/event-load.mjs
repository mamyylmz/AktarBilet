import fs from 'node:fs';
import path from 'node:path';
import { performance } from 'node:perf_hooks';

const baseURL = process.env.QA_API_URL || 'http://127.0.0.1:18080';
const totalEvents = Number(process.env.QA_EVENT_LOAD_TOTAL || 3000);
const batchSize = Number(process.env.QA_EVENT_LOAD_BATCH || 20);
const concurrency = Number(process.env.QA_EVENT_LOAD_CONCURRENCY || 20);
const runID = `${Date.now()}-${Math.random().toString(16).slice(2, 8)}`;
const batches = Math.ceil(totalEvents / batchSize);
const latencies = [];
let nextBatch = 0;
let failedBatches = 0;
let acceptedEvents = 0;

function event(index) {
  return {
    idempotency_key: `LOAD-${runID}-${index}`,
    validator_id: 'VAL_500T_01',
    sequence_no: Number(`${Date.now()}`.slice(-10)) * 10000 + index,
    card_id: `CARD_SIM_${String(index + 1).padStart(6, '0')}`,
    line_id: 'LINE_5',
    station_id: 'STAT_ESHOT_10177',
    direction: 1,
    tapped_at: new Date(Date.now() - index).toISOString()
  };
}

async function worker() {
  while (true) {
    const batchIndex = nextBatch++;
    if (batchIndex >= batches) return;
    const startIndex = batchIndex * batchSize;
    const count = Math.min(batchSize, totalEvents - startIndex);
    const payload = Array.from({ length: count }, (_, offset) => event(startIndex + offset));
    const started = performance.now();
    try {
      const response = await fetch(`${baseURL}/api/v1/validators/batch-sync`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Validator-ID': 'VAL_500T_01' },
        body: JSON.stringify(payload)
      });
      const body = await response.json().catch(() => ({}));
      if (!response.ok || body.processed_count !== count) failedBatches += 1;
      else acceptedEvents += count;
    } catch {
      failedBatches += 1;
    }
    latencies.push(performance.now() - started);
  }
}

const started = performance.now();
await Promise.all(Array.from({ length: concurrency }, worker));
const elapsedSeconds = (performance.now() - started) / 1000;
latencies.sort((a, b) => a - b);
const percentile = (p) => latencies[Math.min(latencies.length - 1, Math.floor(latencies.length * p))];
const result = {
  target: `${baseURL}/api/v1/validators/batch-sync`, run_id: runID,
  requested_events: totalEvents, accepted_events: acceptedEvents,
  batch_size: batchSize, concurrency, failed_batches: failedBatches,
  elapsed_seconds: Number(elapsedSeconds.toFixed(3)),
  events_per_second: Number((acceptedEvents / elapsedSeconds).toFixed(2)),
  error_rate: Number((failedBatches / batches).toFixed(6)),
  batch_latency_ms: {
    p50: Number(percentile(0.50).toFixed(2)),
    p95: Number(percentile(0.95).toFixed(2)),
    p99: Number(percentile(0.99).toFixed(2))
  },
  acceptance: { min_events_per_second: 278, max_error_rate: 0.01 }
};
fs.mkdirSync(path.resolve('artifacts'), { recursive: true });
fs.writeFileSync(path.resolve('artifacts/event-load-results.json'), `${JSON.stringify(result, null, 2)}\n`);
console.log(JSON.stringify(result, null, 2));
process.exit(result.events_per_second >= 278 && result.error_rate <= 0.01 ? 0 : 1);
