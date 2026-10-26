import crypto from 'node:crypto';
import { scalar } from './db.js';

export const apiURL = process.env.QA_API_URL || 'http://127.0.0.1:18080';
export function deviceSecret(validatorID) {
  const escapedID = validatorID.replaceAll("'", "''");
  const secret = scalar(`SELECT secret_key FROM jw_validator_credentials WHERE validator_id='${escapedID}' AND is_revoked=false`);
  if (!secret) throw new Error(`No active QA device credential for ${validatorID}`);
  return secret;
}

export function deviceAuthHeaders(validatorID, sequenceNo, bodyData, nonce) {
  const ts = Math.floor(Date.now() / 1000);
  const n = nonce || `nonce-${Date.now()}-${Math.random().toString(16).slice(2)}`;
  const bodyBytes = Buffer.from(typeof bodyData === 'string' ? bodyData : JSON.stringify(bodyData));
  const payloadHash = crypto.createHash('sha256').update(bodyBytes).digest('hex');
  const secret = deviceSecret(validatorID);
  const msg = `${validatorID}:${sequenceNo}:${ts}:${n}:${payloadHash}`;
  const sig = crypto.createHmac('sha256', secret).update(msg).digest('hex');

  return {
    'X-Validator-ID': validatorID,
    'X-Device-Signature': sig,
    'X-Device-Timestamp': String(ts),
    'X-Device-Nonce': n,
  };
}

export const accounts = {
  admin: ['seed.sedir@ddo-case.invalid', 'SedirPassword2026!'],
  yolcu: ['yolcu@aktarbilet.gov.tr', 'YolcuPassword123!'],
  kontrolor: ['kontrolor@aktarbilet.gov.tr', 'KontrolorPassword123!'],
  hat_yoneticisi: ['hatyoneticisi@aktarbilet.gov.tr', 'HatYoneticisi123!'],
  muhasebe: ['muhasebe@aktarbilet.gov.tr', 'MuhasebePassword123!'],
  operasyon: ['operasyon@aktarbilet.gov.tr', 'OperasyonPassword123!']
};

export async function login(request, role) {
  const [email, password] = accounts[role];
  const response = await request.post(`${apiURL}/api/v1/auth/login`, {
    data: { email, password }
  });
  const body = await response.json();
  return { response, body, token: body.access_token };
}

export function bearer(token) {
  return { Authorization: `Bearer ${token}` };
}
