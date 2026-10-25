export const apiURL = process.env.QA_API_URL || 'http://127.0.0.1:18080';

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
