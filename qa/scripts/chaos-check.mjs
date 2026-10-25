import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';

const project = process.env.QA_PROJECT_NAME || 'aktarbilet-qa';
const compose = ['compose', '--project-name', project, '--file', 'docker-compose.qa.yml'];
const result = { project, started_at: new Date().toISOString() };
const docker = (...args) => execFileSync('docker', [...compose, ...args], { encoding: 'utf8' });

async function health() {
  try {
    const response = await fetch('http://127.0.0.1:18080/health');
    return { status: response.status, body: await response.json() };
  } catch (error) {
    return { status: 0, error: error.message };
  }
}

async function waitForStatus(status, attempts = 30) {
  for (let i = 0; i < attempts; i += 1) {
    const current = await health();
    if (current.status === status) return current;
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  return health();
}

let exitCode = 0;
try {
  result.before = await health();
  docker('stop', 'postgres');
  result.during_outage = await waitForStatus(503);
  docker('start', 'postgres');
  result.after_recovery = await waitForStatus(200);
  const mismatch = docker('exec', '-T', 'postgres', 'psql', '-X', '-q', '-A', '-t',
    '-U', 'aktarbilet_qa', '-d', 'aktarbilet_qa', '-c', `SELECT COUNT(*) FROM (
      SELECT c.id FROM jw_cards c LEFT JOIN jw_ledger l ON l.card_id=c.id
      GROUP BY c.id,c.balance_kurus HAVING c.balance_kurus<>COALESCE(SUM(l.amount_kurus),0)
    ) q;`).trim();
  result.balance_ledger_mismatches = Number(mismatch);
  if (result.before.status !== 200 || result.during_outage.status !== 503 ||
      result.after_recovery.status !== 200 || result.balance_ledger_mismatches !== 0) exitCode = 1;
} catch (error) {
  result.error = error.message;
  exitCode = 1;
} finally {
  try { docker('start', 'postgres'); } catch {}
  result.completed_at = new Date().toISOString();
  result.passed = exitCode === 0;
  fs.mkdirSync(path.resolve('artifacts'), { recursive: true });
  fs.writeFileSync(path.resolve('artifacts/chaos-results.json'), `${JSON.stringify(result, null, 2)}\n`);
  console.log(JSON.stringify(result, null, 2));
}
process.exit(exitCode);
