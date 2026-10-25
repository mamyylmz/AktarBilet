import crypto from 'node:crypto';

function encode(value) {
  return Buffer.from(JSON.stringify(value)).toString('base64url');
}

export function signedToken(payload, algorithm = 'HS256') {
  const header = encode({ alg: algorithm, typ: 'JWT' });
  const body = encode(payload);
  const digest = algorithm === 'HS512' ? 'sha512' : 'sha256';
  const signature = crypto
    .createHmac(digest, 'qa-only-jwt-secret-at-least-thirty-two-bytes')
    .update(`${header}.${body}`)
    .digest('base64url');
  return `${header}.${body}.${signature}`;
}
