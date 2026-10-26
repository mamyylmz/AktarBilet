// Test runner for requests.http and demo accounts against the live backend
import crypto from 'node:crypto';

const BASE_URL = process.env.QA_API_URL || 'http://127.0.0.1:18080';

console.log('================================================================');
console.log('       AktarBilet requests.http & Postman Doğrulama Testi      ');
console.log('       Hedef Sunucu:', BASE_URL);
console.log('================================================================\n');

const demoAccounts = {
  admin: { email: 'seed.sedir@ddo-case.invalid', password: 'SedirPassword2026!' },
  yolcu: { email: 'yolcu@aktarbilet.gov.tr', password: 'YolcuPassword123!' },
  kontrolor: { email: 'kontrolor@aktarbilet.gov.tr', password: 'KontrolorPassword123!' },
  hatYoneticisi: { email: 'hatyoneticisi@aktarbilet.gov.tr', password: 'HatYoneticisi123!' },
  muhasebe: { email: 'muhasebe@aktarbilet.gov.tr', password: 'MuhasebePassword123!' },
  operasyon: { email: 'operasyon@aktarbilet.gov.tr', password: 'OperasyonPassword123!' },
};

const tokens = {};
let allPassed = true;
let totalTests = 0;
let passedTests = 0;

function assert(condition, name, details = '') {
  totalTests++;
  if (condition) {
    passedTests++;
    console.log(`  ✅ [PASS] ${name} ${details ? '(' + details + ')' : ''}`);
  } else {
    allPassed = false;
    console.error(`  ❌ [FAIL] ${name} ${details ? '(' + details + ')' : ''}`);
  }
}

async function run() {
  // 1. Sistem Sağlık Uçları
  console.log('--- 1. Sistem & Sağlık Uçları ---');
  {
    const res = await fetch(`${BASE_URL}/health`);
    const body = await res.json();
    assert(res.status === 200 && body.status === 'UP', 'GET /health', `status: ${res.status}, stamp: ${res.headers.get('x-ddo-stamp')}`);
  }
  {
    const res = await fetch(`${BASE_URL}/_hz/f85f`);
    const body = await res.json();
    assert(res.status === 200 && body.hz === 'f85f', 'GET /_hz/f85f (Canary 4)', `status: ${res.status}`);
  }
  {
    const res = await fetch(`${BASE_URL}/metrics`);
    assert(res.status === 200 && res.headers.get('content-type')?.includes('text/plain'), 'GET /metrics (Prometheus)', `status: ${res.status}`);
  }

  // 2. Demo Hesap Girişleri
  console.log('\n--- 2. Demo Hesap Girişleri (Tüm Roller) ---');
  for (const [role, creds] of Object.entries(demoAccounts)) {
    const res = await fetch(`${BASE_URL}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(creds),
    });
    const body = await res.json();
    tokens[role] = body.access_token;
    assert(res.status === 200 && !!body.access_token, `POST /auth/login [${role}] (${creds.email})`, `token_length: ${body.access_token?.length}`);
  }

  // 2.7. Profil ve Rol Değişimi
  console.log('\n--- 2.7. Profil & Rol İşlemleri ---');
  {
    const res = await fetch(`${BASE_URL}/api/v1/auth/me`, {
      headers: { Authorization: `Bearer ${tokens.admin}` },
    });
    const body = await res.json();
    assert(res.status === 200 && body.email === demoAccounts.admin.email, 'GET /auth/me', `role: ${body.role}`);
  }
  {
    const res = await fetch(`${BASE_URL}/api/v1/auth/switch-role`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${tokens.admin}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ role: 'admin' }),
    });
    assert(res.status === 200, 'POST /auth/switch-role', `status: ${res.status}`);
  }

  // 3. F-1: Kart ve Bakiye Yönetimi
  console.log('\n--- 3. F-1: Kart ve Bakiye Yönetimi ---');
  {
    const res = await fetch(`${BASE_URL}/api/v1/cards/CARD_YOLCU_001`, {
      headers: { Authorization: `Bearer ${tokens.yolcu}` },
    });
    const body = await res.json();
    assert(res.status === 200 && body.balance_kurus !== undefined, 'GET /cards/CARD_YOLCU_001', `bakiye: ${body.balance_tl}`);
  }
  {
    const res = await fetch(`${BASE_URL}/api/v1/cards/CARD_YOLCU_001/ledger`, {
      headers: { Authorization: `Bearer ${tokens.yolcu}` },
    });
    assert(res.status === 200, 'GET /cards/CARD_YOLCU_001/ledger', `status: ${res.status}`);
  }
  {
    const idemp = `COLLECTION_RECHARGE_${Date.now()}`;
    const res = await fetch(`${BASE_URL}/api/v1/cards/CARD_YOLCU_001/recharge`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${tokens.yolcu}`,
        'Content-Type': 'application/json',
        'Idempotency-Key': idemp,
      },
      body: JSON.stringify({ amount_kurus: 1000, description: 'Test Koleksiyonu Dolumu' }),
    });
    assert(res.status === 200, 'POST /cards/CARD_YOLCU_001/recharge (Idempotent)', `status: ${res.status}`);
  }

  // 4. F-3: Hat, Durak ve Tarife Kataloğu
  console.log('\n--- 4. F-3: Hat, Durak ve Tarife Kataloğu ---');
  {
    const res = await fetch(`${BASE_URL}/api/v1/lines`);
    const body = await res.json();
    assert(res.status === 200 && Array.isArray(body) && body.length >= 40, 'GET /lines', `hat sayısı: ${body.length}`);
  }
  {
    const lineId = `LINE_COL_${Date.now().toString(16).slice(-6).toUpperCase()}`;
    const res = await fetch(`${BASE_URL}/api/v1/lines`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${tokens.hatYoneticisi}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ id: lineId, name: 'Koleksiyon Test Hattı', mode: 'bus' }),
    });
    assert(res.status === 201, 'POST /lines (Hat Yöneticisi)', `created: ${lineId}`);
  }
  {
    const res = await fetch(`${BASE_URL}/api/v1/stations`);
    const body = await res.json();
    assert(res.status === 200 && Array.isArray(body), 'GET /stations', `durak sayısı: ${body.length}`);
  }
  {
    const res = await fetch(`${BASE_URL}/api/v1/tariffs`);
    const body = await res.json();
    assert(res.status === 200 && Array.isArray(body), 'GET /tariffs', `tarife sayısı: ${body.length}`);
  }

  // 5. F-6: Saha Denetimi ve Cezalar
  console.log('\n--- 5. F-6: Saha Denetimi ve Ceza Tutanakları ---');
  {
    const res = await fetch(`${BASE_URL}/api/v1/penalties`, {
      headers: { Authorization: `Bearer ${tokens.kontrolor}` },
    });
    assert(res.status === 200, 'GET /penalties (Kontrolör)', `status: ${res.status}`);
  }
  {
    const res = await fetch(`${BASE_URL}/api/v1/penalties`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${tokens.kontrolor}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        card_id: 'CARD_TAM_001',
        penalty_amount_kurus: 5000,
        reason: 'kartsiz_binis',
        notes: 'requests.http deneme tutanağı',
      }),
    });
    assert(res.status === 201, 'POST /penalties (Kontrolör)', `status: ${res.status}`);
  }

  // 6. F-5 / T-4: Gelir Paylaşımı ve Mutabakat
  console.log('\n--- 6. F-5 / T-4: Gelir Paylaşımı ve Mutabakat ---');
  {
    const res = await fetch(`${BASE_URL}/api/v1/settlements`, {
      headers: { Authorization: `Bearer ${tokens.muhasebe}` },
    });
    assert(res.status === 200, 'GET /settlements (Muhasebe)', `status: ${res.status}`);
  }
  const testPeriodId = 'a0000000-0000-0000-0000-' + Date.now().toString(16).padStart(12, '0');
  {
    const res = await fetch(`${BASE_URL}/api/v1/settlements/close`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${tokens.muhasebe}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        period_id: testPeriodId,
        period_name: `PERIOD-COL-${Date.now()}`,
      }),
    });
    assert(res.status === 200 || res.status === 409, 'POST /settlements/close (Muhasebe)', `status: ${res.status}`);
  }
  {
    const res = await fetch(`${BASE_URL}/api/v1/settlements/reopen`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${tokens.admin}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        period_id: testPeriodId,
        reason: 'requests.http deneme düzeltmesi',
      }),
    });
    assert(res.status === 200, 'POST /settlements/reopen (Admin)', `status: ${res.status}`);
  }
  {
    const res = await fetch(`${BASE_URL}/api/v1/settlements/storno`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${tokens.muhasebe}`,
        'Content-Type': 'application/json',
        'Idempotency-Key': `STORNO_COL_${Date.now()}`,
      },
      body: JSON.stringify({
        period_id: testPeriodId,
        operator: 'ozel_halk',
        amount_kurus: -500,
        reason: 'requests.http storno denemesi',
      }),
    });
    assert(res.status === 200, 'POST /settlements/storno (Muhasebe)', `status: ${res.status}`);
  }

  // 7. Operasyon ve Güvenlik Anomalileri
  console.log('\n--- 7. Operasyon ve Güvenlik Anomalileri ---');
  {
    const res = await fetch(`${BASE_URL}/api/v1/anomalies`, {
      headers: { Authorization: `Bearer ${tokens.operasyon}` },
    });
    assert(res.status === 200, 'GET /anomalies (Operasyon)', `status: ${res.status}`);
  }

  // 8. Canlı Ulaşım ve ESHOT Entegrasyonu
  console.log('\n--- 8. Canlı Ulaşım ve ESHOT Entegrasyonu ---');
  {
    const res = await fetch(`${BASE_URL}/api/v1/transit/live-buses?line=800`);
    assert(res.status === 200, 'GET /transit/live-buses?line=800', `status: ${res.status}`);
  }
  {
    const res = await fetch(`${BASE_URL}/api/v1/transit/lines`);
    assert(res.status === 200, 'GET /transit/lines', `status: ${res.status}`);
  }

  // 9. Hat Optimizasyonu ve Sefer Sıklığı
  console.log('\n--- 9. Hat Optimizasyonu ve Sefer Sıklığı ---');
  {
    const res = await fetch(`${BASE_URL}/api/v1/optimization/headway-scenario`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${tokens.hatYoneticisi}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        line_id: 'LINE_500T',
        current_headway_minutes: 15,
        target_headway_minutes: 8,
        fleet_capacity_per_vehicle: 75,
        peak_hour: true,
      }),
    });
    assert(res.status === 200, 'POST /optimization/headway-scenario (Hat Yöneticisi)', `status: ${res.status}`);
  }
  {
    const res = await fetch(`${BASE_URL}/api/v1/optimization/approve`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${tokens.hatYoneticisi}`,
        'Content-Type': 'application/json',
        'Idempotency-Key': `OPT_COL_${Date.now()}`,
      },
      body: JSON.stringify({
        line_id: 'LINE_500T',
        current_headway_minutes: 15,
        target_headway_minutes: 8,
        fleet_capacity: 75,
        vehicle_delta: 4,
        delta_cost_kurus: 250000,
        projected_net_margin_kurus: 450000,
        notes: 'Koleksiyon test onayı',
      }),
    });
    assert(res.status === 201, 'POST /optimization/approve (Hat Yöneticisi)', `status: ${res.status}`);
  }

  // 10. Raporlama ve Bonuslar
  console.log('\n--- 10. Raporlama ve Bonuslar ---');
  {
    const res = await fetch(`${BASE_URL}/api/v1/reports/heatmap`);
    assert(res.status === 200, 'GET /reports/heatmap', `status: ${res.status}`);
  }
  {
    const res = await fetch(`${BASE_URL}/api/v1/reports/line-financials`, {
      headers: { Authorization: `Bearer ${tokens.muhasebe}` },
    });
    assert(res.status === 200, 'GET /reports/line-financials (Muhasebe)', `status: ${res.status}`);
  }
  {
    const res = await fetch(`${BASE_URL}/api/v1/simulation/fare-impact`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${tokens.hatYoneticisi}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        base_fare_kurus: 2000,
        transfer_fare_kurus: 1000,
      }),
    });
    assert(res.status === 200, 'POST /simulation/fare-impact (Hat Yöneticisi)', `status: ${res.status}`);
  }

  console.log('\n================================================================');
  console.log(`Sonuç: ${passedTests} / ${totalTests} test başarıyla tamamlandı.`);
  if (allPassed) {
    console.log('✅ TÜM ELLE DENEME VE KOLEKSİYON UÇ NOKTALARI %100 ÇALIŞIR DURUMDA!');
    process.exit(0);
  } else {
    console.error('❌ BAZI İSTEKLER BAŞARISIZ OLDU!');
    process.exit(1);
  }
}

run().catch((err) => {
  console.error('Kritik test hatası:', err);
  process.exit(1);
});
