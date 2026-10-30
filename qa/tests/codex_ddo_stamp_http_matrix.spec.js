import { test, expect } from '@playwright/test';

const apiURL = process.env.QA_API_URL || 'http://127.0.0.1:18080';
const frontendURL = process.env.QA_FRONTEND_URL || 'http://127.0.0.1:13000';
const accounts = {
  admin: ['seed.sedir@ddo-case.invalid', 'SedirPassword2026!'],
  passenger: ['yolcu@aktarbilet.gov.tr', 'YolcuPassword123!'],
  accounting: ['muhasebe@aktarbilet.gov.tr', 'MuhasebePassword123!'],
};

async function postLogin(request, account, password = accounts[account][1]) {
  return request.post(`${apiURL}/api/v1/auth/login`, {
    data: { email: accounts[account][0], password },
  });
}

async function expectStamp(response, label) {
  expect(response.headers()['x-ddo-stamp'], `${label} (${response.status()})`).toBe('f85f26');
}

test('X-DDO-Stamp is present on backend, error, auth, CORS, and frontend-proxied responses', async ({ request }) => {
  const observed = [];
  const capture = async (label, promise, expectedStatus) => {
    const response = await promise;
    expect(response.status(), label).toBe(expectedStatus);
    await expectStamp(response, label);
    observed.push(`${label}=${response.status()}`);
    return response;
  };

  await capture('health', request.get(`${apiURL}/health`), 200);
  await capture('canary', request.get(`${apiURL}/_hz/f85f`), 200);
  await capture('catalog', request.get(`${apiURL}/api/v1/tariffs`), 200);
  await capture('frontend-proxy', request.get(`${frontendURL}/api/v1/tariffs`), 200);
  await capture('not-found', request.get(`${apiURL}/api/v1/no-such-route`), 404);
  await capture('method-not-allowed', request.patch(`${apiURL}/health`), 405);

  const preflight = await capture('cors-preflight', request.fetch(`${apiURL}/api/v1/auth/me`, {
    method: 'OPTIONS',
    headers: {
      Origin: 'http://localhost:3000',
      'Access-Control-Request-Method': 'GET',
      'Access-Control-Request-Headers': 'authorization',
    },
  }), 200);
  expect(preflight.headers()['access-control-allow-origin']).toBe('http://localhost:3000');

  await capture('bad-password', postLogin(request, 'passenger', 'incorrect-password'), 401);
  await capture('missing-auth', request.get(`${apiURL}/api/v1/auth/me`), 401);
  await capture('invalid-token', request.get(`${apiURL}/api/v1/auth/me`, {
    headers: { Authorization: 'Bearer invalid.jwt.value' },
  }), 401);

  const passengerLogin = await capture('passenger-login', postLogin(request, 'passenger'), 200);
  const passengerTokens = await passengerLogin.json();
  await capture('refresh-as-access', request.get(`${apiURL}/api/v1/auth/me`, {
    headers: { Authorization: `Bearer ${passengerTokens.refresh_token}` },
  }), 401);
  await capture('forbidden-role', request.get(`${apiURL}/api/v1/settlements`, {
    headers: { Authorization: `Bearer ${passengerTokens.access_token}` },
  }), 403);

  const accountingLogin = await capture('accounting-login', postLogin(request, 'accounting'), 200);
  const accountingTokens = await accountingLogin.json();
  await capture('invalid-body', request.post(`${apiURL}/api/v1/settlements/close`, {
    headers: { Authorization: `Bearer ${accountingTokens.access_token}` },
    data: {},
  }), 400);

  console.log(`STAMP_MATRIX ${observed.join(' ')}`);
});
