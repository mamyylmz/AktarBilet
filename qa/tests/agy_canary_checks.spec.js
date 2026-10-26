import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const qaRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const projectRoot = path.resolve(qaRoot, '..');
const BASE_URL = process.env.QA_FRONTEND_URL || 'http://localhost:3000';
const API_URL = process.env.QA_API_URL || 'http://localhost:3000';

test.describe('AGY Mandatory 7-Canary Audit', () => {

  test('Canary 1 & 2: Database table prefix is strictly jw_ and user table is jw_tf_users', async () => {
    const { rows } = await import('../helpers/db.js');
    const tables = rows("SELECT tablename FROM pg_tables WHERE schemaname = 'public';");
    expect(tables.length).toBeGreaterThan(0);

    for (const t of tables) {
      expect(t.startsWith('jw_'), `Table ${t} must start with jw_ prefix`).toBe(true);
    }

    expect(tables).toContain('jw_tf_users');
  });

  test('Canary 3: API responses include X-DDO-Stamp: f85f26 header', async ({ request }) => {
    const endpoints = [
      `${API_URL}/health`,
      `${API_URL}/_hz/f85f`,
      `${API_URL}/api/v1/tariffs`,
    ];

    for (const ep of endpoints) {
      const res = await request.get(ep);
      const stamp = res.headers()['x-ddo-stamp'];
      expect(stamp, `Endpoint ${ep} must include X-DDO-Stamp: f85f26`).toBe('f85f26');
    }
  });

  test('Canary 4: GET /_hz/f85f endpoint returns 200 OK', async ({ request }) => {
    const res = await request.get(`${API_URL}/_hz/f85f`);
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.hz).toBe('f85f');
    expect(body.stamp).toBe('f85f26');
    expect(body.status).toBe('ok');
  });

  test('Canary 5: Seed user email is seed.sedir@ddo-case.invalid', async () => {
    const { scalar } = await import('../helpers/db.js');
    const firstUserEmail = scalar('SELECT email FROM jw_tf_users ORDER BY created_at ASC LIMIT 1;');
    expect(firstUserEmail).toBe('seed.sedir@ddo-case.invalid');
  });

  test('Canary 6: docker-compose.yml backend container name is ddo-pelikan-api', async () => {
    const composePath = path.join(projectRoot, 'docker-compose.yml');
    const content = fs.readFileSync(composePath, 'utf8');
    expect(content).toContain('container_name: ddo-pelikan-api');
  });

  test('Canary 7: README.md last line is strictly DDO Case Kodu: SEDIR-f85f26', async () => {
    const readmePath = path.join(projectRoot, 'README.md');
    const content = fs.readFileSync(readmePath, 'utf8').trim();
    const lastLine = content.split('\n').filter(Boolean).at(-1)?.trim();
    expect(lastLine).toBe('DDO Case Kodu: SEDIR-f85f26');
  });
});

