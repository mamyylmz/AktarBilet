import fs from 'node:fs';
import path from 'node:path';
import { performance } from 'node:perf_hooks';

const baseURL = process.env.QA_API_URL || 'http://127.0.0.1:18080';
const durationSeconds = Number(process.env.QA_SUSTAINED_SECONDS || 600);
const batchesPerSecond = Number(process.env.QA_BATCHES_PER_SECOND || 15);
const batchSize = Number(process.env.QA_BATCH_SIZE || 20);
const runID = process.env.QA_RUN_ID || `codex-${Date.now()}`;
const sequenceBase = Date.now() * 1000;
const latencies = [];
let submittedEvents = 0;
let acceptedEvents = 0;
let failedBatches = 0;
let unexpectedResponses = 0;

function percentile(sorted, ratio) {
  if (sorted.length === 0) return null;
  return sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * ratio))];
}

function makeBatch(startIndex) {
  const now = new Date().toISOString();
  return Array.from({ length: batchSize }, (_, offset) => {
    const index = startIndex + offset;
    return {
      idempotency_key: `CODEX-SUSTAINED-${runID}-${index}`,
      validator_id: 'VAL_500T_01',
      sequence_no: sequenceBase + index,
      card_id: `CARD_SIM_${String((index % 200000) + 1).padStart(6, '0')}`,
      line_id: 'LINE_5',
      station_id: 'STAT_ESHOT_10177',
      direction: 1,
      tapped_at: now,
    };
  });
}

async function sendBatch(batch) {
  const started = performance.now();
  submittedEvents += batch.length;
  try {
    const response = await fetch(`${baseURL}/api/v1/validators/batch-sync`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Validator-ID': 'VAL_500T_01',
      },
      body: JSON.stringify(batch),
    });
    const body = await response.json().catch(() => ({}));
    if (!response.ok || body.processed_count !== batch.length) {
      failedBatches += 1;
      unexpectedResponses += 1;
      return;
    }
    acceptedEvents += batch.length;
  } catch {
    failedBatches += 1;
  } finally {
    latencies.push(performance.now() - started);
  }
}

const wallStartedAt = new Date();
const started = performance.now();
for (let second = 0; second < durationSeconds; second += 1) {
  const work = [];
  for (let batch = 0; batch < batchesPerSecond; batch += 1) {
    const startIndex = (second * batchesPerSecond + batch) * batchSize;
    work.push(sendBatch(makeBatch(startIndex)));
  }
  await Promise.all(work);
  if ((second + 1) % 60 === 0) {
    console.log(JSON.stringify({ elapsed_seconds: second + 1, submitted_events: submittedEvents, accepted_events: acceptedEvents, failed_batches: failedBatches }));
  }
  const nextWindow = started + (second + 1) * 1000;
  const remaining = nextWindow - performance.now();
  if (remaining > 0) await new Promise((resolve) => setTimeout(resolve, remaining));
}

const elapsedSeconds = (performance.now() - started) / 1000;
latencies.sort((a, b) => a - b);
const result = {
  run_id: runID,
  target: `${baseURL}/api/v1/validators/batch-sync`,
  started_at: wallStartedAt.toISOString(),
  ended_at: new Date().toISOString(),
  configured_duration_seconds: durationSeconds,
  elapsed_seconds: Number(elapsedSeconds.toFixed(3)),
  batch_size: batchSize,
  batches_per_second: batchesPerSecond,
  submitted_events: submittedEvents,
  accepted_events: acceptedEvents,
  failed_batches: failedBatches,
  unexpected_responses: unexpectedResponses,
  accepted_events_per_second: Number((acceptedEvents / elapsedSeconds).toFixed(2)),
  batch_latency_ms: {
    p50: Number(percentile(latencies, 0.50)?.toFixed(2)),
    p95: Number(percentile(latencies, 0.95)?.toFixed(2)),
    p99: Number(percentile(latencies, 0.99)?.toFixed(2)),
  },
  caveat: 'Batch endpoint accepts prefix-based anonymous device identity; this measurement is not a validator-security pass.',
};

const output = path.resolve('artifacts/codex/sustained-ingest-client.json');
fs.mkdirSync(path.dirname(output), { recursive: true });
fs.writeFileSync(output, `${JSON.stringify(result, null, 2)}\n`);
console.log(JSON.stringify(result, null, 2));
process.exit(result.accepted_events_per_second >= 278 && failedBatches === 0 ? 0 : 1);
