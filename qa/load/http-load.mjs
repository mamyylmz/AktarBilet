import fs from 'node:fs';
import path from 'node:path';
import { performance } from 'node:perf_hooks';

const baseURL = process.env.QA_API_URL || 'http://127.0.0.1:18080';
const requests = Number(process.env.QA_LOAD_REQUESTS || 2000);
const concurrency = Number(process.env.QA_LOAD_CONCURRENCY || 40);
const latencies = [];
let next = 0;
let failures = 0;

async function worker() {
  while (true) {
    const index = next++;
    if (index >= requests) return;
    const started = performance.now();
    try {
      const response = await fetch(`${baseURL}/api/v1/lines`);
      if (!response.ok) failures += 1;
      await response.arrayBuffer();
    } catch {
      failures += 1;
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
  target: `${baseURL}/api/v1/lines`, requests, concurrency,
  elapsed_seconds: Number(elapsedSeconds.toFixed(3)),
  requests_per_second: Number((requests / elapsedSeconds).toFixed(2)),
  error_rate: Number((failures / requests).toFixed(6)), failures,
  latency_ms: { p50: Number(percentile(0.50).toFixed(2)), p95: Number(percentile(0.95).toFixed(2)), p99: Number(percentile(0.99).toFixed(2)) },
  acceptance: { min_rps: 278, max_error_rate: 0.01 }
};
fs.mkdirSync(path.resolve('artifacts'), { recursive: true });
fs.writeFileSync(path.resolve('artifacts/load-results.json'), `${JSON.stringify(result, null, 2)}\n`);
console.log(JSON.stringify(result, null, 2));
process.exit(result.requests_per_second >= 278 && result.error_rate <= 0.01 ? 0 : 1);
