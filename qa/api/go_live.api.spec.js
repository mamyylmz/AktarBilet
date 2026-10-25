import { execFileSync } from 'node:child_process';
import { test, expect } from '@playwright/test';
import { apiURL } from '../helpers/http.js';

const project = process.env.QA_PROJECT_NAME || 'aktarbilet-qa';
const compose = ['compose', '--project-name', project, '--file', 'docker-compose.qa.yml'];

function docker(...args) {
  return execFileSync('docker', [...compose, ...args], { cwd: process.cwd(), encoding: 'utf8' });
}

async function status(request, path = '/health') {
  try { return (await request.get(`${apiURL}${path}`, { timeout: 2_000 })).status(); }
  catch { return 0; }
}

async function waitFor(request, wanted, attempts = 30) {
  for (let i = 0; i < attempts; i += 1) {
    if (await status(request) === wanted) return true;
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  return false;
}

async function waitForAnyHTTP(request, attempts = 30) {
  for (let i = 0; i < attempts; i += 1) {
    const current = await status(request);
    if (current !== 0) return current;
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  return 0;
}

test.describe('go-live API protocol checks', () => {
  test('SYS-03 stamp and structured errors survive success, 400, 401 and 404', async ({ request }) => {
    const probes = [
      await request.get(`${apiURL}/health`),
      await request.post(`${apiURL}/api/v1/auth/login`, { data: { email: 'bad', password: 'bad' } }),
      await request.get(`${apiURL}/api/v1/auth/me`),
      await request.get(`${apiURL}/definitely-not-a-route`)
    ];
    for (const response of probes) expect(response.headers()['x-ddo-stamp']).toBe('f85f26');
    for (const response of probes.slice(1, 3)) {
      const body = await response.json();
      expect(body).toHaveProperty('error.code');
      expect(body).toHaveProperty('error.message');
    }
  });

  test('SEC-04 SQL injection payload cannot authenticate', async ({ request }) => {
    const response = await request.post(`${apiURL}/api/v1/auth/login`, {
      data: { email: "' OR 1=1 --", password: "' OR 1=1 --" }
    });
    expect(response.status()).toBe(401);
    expect(await response.json()).not.toHaveProperty('access_token');
  });

  test('ARCH-04 metrics endpoint exposes Prometheus text', async ({ request }) => {
    const response = await request.get(`${apiURL}/metrics`);
    expect(response.status()).toBe(200);
    expect(response.headers()['content-type']).toContain('text/plain');
    expect(await response.text()).toContain('go_gc_duration_seconds');
  });

  test('ARCH-04 a fresh backend must not report healthy without PostgreSQL', async ({ request }) => {
    test.setTimeout(60_000);
    let observed = 0;
    try {
      docker('stop', 'postgres');
      docker('up', '--detach', '--force-recreate', '--no-deps', 'backend');
      observed = await waitForAnyHTTP(request);
    } finally {
      docker('start', 'postgres');
      docker('up', '--detach', '--force-recreate', '--no-deps', 'backend');
      expect(await waitFor(request, 200), 'QA backend did not recover after the outage probe').toBe(true);
      // Nginx resolves the backend name at startup; refresh it after recreating backend.
      docker('up', '--detach', '--force-recreate', '--no-deps', 'frontend');
    }
    expect(observed).toBe(503);
  });
});
