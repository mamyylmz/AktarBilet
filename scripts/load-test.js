import http from 'k6/http';
import { check, sleep } from 'k6';

// k6 Yük Testi Yapılandırması (AktarBilet Yüksek Başarım ve Düşük Gecikme Testi)
export const options = {
  stages: [
    { duration: '10s', target: 50 },  // 50 eşzamanlı kullanıcıya ısınma
    { duration: '30s', target: 200 }, // 200 eşzamanlı kullanıcı altında pik yük
    { duration: '10s', target: 0 },   // Soğuma
  ],
  thresholds: {
    http_req_duration: ['p(95)<50', 'p(99)<150'], // p95 < 50ms, p99 < 150ms
    http_req_failed: ['rate<0.01'],               // Hata oranı < %1
  },
};

const BASE_URL = __ENV.BACKEND_URL || 'http://localhost:8080';

export default function () {
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-DDO-Stamp': 'f85f26',
    },
  };

  // 1. Sağlık Ucu Kontrolü
  let resHealth = http.get(`${BASE_URL}/health`, params);
  check(resHealth, {
    'health status 200': (r) => r.status === 200,
    'has ddo stamp': (r) => r.headers['X-Ddo-Stamp'] === 'f85f26',
  });

  // 2. Hatlar Listesi Sorgusu (Okuma Performansı)
  let resLines = http.get(`${BASE_URL}/api/v1/lines`, params);
  check(resLines, {
    'lines status 200': (r) => r.status === 200,
  });

  // 3. Doluluk Isı Haritası Sorgusu (Materialized View / Özet Tablo Performansı)
  let resHeatmap = http.get(`${BASE_URL}/api/v1/reports/heatmap`, params);
  check(resHeatmap, {
    'heatmap status 200': (r) => r.status === 200,
  });

  // 4. Kart Okutma Simülasyonu (Yazma & Eşzamanlılık)
  const tapPayload = JSON.stringify({
    validator_id: `VAL_K6_${__VU}`,
    sequence_no: __ITER,
    card_id: 'CARD_TAM_001',
    line_id: 'LINE_500T',
    station_id: 'STAT_KADIKOY',
    direction: 1,
    tapped_at: new Date().toISOString(),
  });

  let resTap = http.post(`${BASE_URL}/api/v1/validators/tap`, tapPayload, params);
  check(resTap, {
    'tap status 200': (r) => r.status === 200,
  });

  sleep(0.1);
}
