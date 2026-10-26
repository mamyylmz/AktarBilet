import { test, expect } from '@playwright/test';
import { apiURL, bearer, login } from '../helpers/http.js';
import { scalar } from '../helpers/db.js';

test.describe('Codex cost/optimization contract audit', () => {
  test('line-financials response matches the array consumed by the frontend', async ({ request }) => {
    const { token } = await login(request, 'muhasebe');
    const response = await request.get(`${apiURL}/api/v1/reports/line-financials`, {
      headers: bearer(token)
    });
    expect(response.status()).toBe(200);
    expect(Array.isArray(await response.json()), 'App.jsx discards any non-array response').toBe(true);
  });

  test('the frontend headway DTO preserves the selected target', async ({ request }) => {
    const { token } = await login(request, 'hat_yoneticisi');
    const response = await request.post(`${apiURL}/api/v1/optimization/headway-scenario`, {
      headers: bearer(token),
      data: {
        line_id: '800', current_headway_mins: 15, target_headway_mins: 8,
        fleet_capacity_per_vehicle: 100, peak_hour: true
      }
    });
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.target_headway_minutes, 'UI-selected 8 minutes must not silently become the backend default').toBe(8);
    for (const field of ['current_vehicles', 'proposed_vehicles', 'proposed_headway_mins', 'vehicle_delta', 'delta_cost_kurus', 'demand_change_basis_points', 'projected_net_margin_kurus', 'model_basis']) {
      expect(body, `frontend renders response field ${field}`).toHaveProperty(field);
    }
  });

  test('fleet capacity materially changes an otherwise identical optimization', async ({ request }) => {
    const { token } = await login(request, 'hat_yoneticisi');
    const run = async (capacity) => {
      const response = await request.post(`${apiURL}/api/v1/optimization/headway-scenario`, {
        headers: bearer(token),
        data: {
          line_id: '800', current_headway_minutes: 15, target_headway_minutes: 8,
          fleet_capacity_per_vehicle: capacity
        }
      });
      expect(response.status()).toBe(200);
      return response.json();
    };

    const lowCapacity = await run(50);
    const highCapacity = await run(200);
    expect(lowCapacity, 'capacity is currently accepted but ignored').not.toEqual(highCapacity);
  });

  test('an authorized optimization approval is durably persisted', async ({ request }) => {
    const { token } = await login(request, 'hat_yoneticisi');
    const notes = `CODEX-PERSIST-${Date.now()}`;
    const before = Number(scalar('SELECT COUNT(*) FROM jw_optimization_approvals'));

    const response = await request.post(`${apiURL}/api/v1/optimization/approve`, {
      headers: { ...bearer(token), 'Idempotency-Key': `CODEX-PERSIST-IDEMP-${Date.now()}` },
      data: {
        line_id: 'LINE_500T', current_headway_minutes: 15, target_headway_minutes: 8,
        fleet_capacity: 75, vehicle_delta: 5, delta_cost_kurus: 5200000,
        projected_net_margin_kurus: 1400000, notes
      }
    });

    expect(response.status()).toBe(201);
    const body = await response.json();
    expect(Number(scalar('SELECT COUNT(*) FROM jw_optimization_approvals'))).toBe(before + 1);
    expect(scalar(`SELECT COUNT(*) FROM jw_optimization_approvals WHERE id='${body.id}'`)).toBe('1');
    expect(scalar(`SELECT approved_by::text FROM jw_optimization_approvals WHERE id='${body.id}'`)).toBe(body.approved_by);
    expect(scalar(`SELECT COUNT(*) FROM jw_optimization_approvals WHERE id='${body.id}' AND approved_at IS NOT NULL`)).toBe('1');
  });

  test('a passenger cannot persist an optimization approval', async ({ request }) => {
    const { token } = await login(request, 'yolcu');
    const notes = `CODEX-DENY-${Date.now()}`;
    const before = scalar('SELECT COUNT(*) FROM jw_optimization_approvals');
    const response = await request.post(`${apiURL}/api/v1/optimization/approve`, {
      headers: { ...bearer(token), 'Idempotency-Key': `CODEX-DENY-IDEMP-${Date.now()}` },
      data: {
        line_id: 'LINE_500T', current_headway_minutes: 15, target_headway_minutes: 8,
        fleet_capacity: 75, vehicle_delta: 5, delta_cost_kurus: 5200000,
        projected_net_margin_kurus: 1400000, notes
      }
    });

    expect(response.status()).toBe(403);
    expect(scalar('SELECT COUNT(*) FROM jw_optimization_approvals')).toBe(before);
  });

  test('retrying the same approval key creates one durable decision', async ({ request }) => {
    const { token } = await login(request, 'hat_yoneticisi');
    const key = `CODEX-APPROVAL-IDEMP-${Date.now()}`;
    const notes = `CODEX-APPROVAL-RETRY-${Date.now()}`;
    const before = Number(scalar('SELECT COUNT(*) FROM jw_optimization_approvals'));
    const data = {
      line_id: 'LINE_500T', current_headway_minutes: 15, target_headway_minutes: 8,
      fleet_capacity: 75, vehicle_delta: 5, delta_cost_kurus: 5200000,
      projected_net_margin_kurus: 1400000, notes
    };
    const headers = { ...bearer(token), 'Idempotency-Key': key };

    const first = await request.post(`${apiURL}/api/v1/optimization/approve`, { headers, data });
    const second = await request.post(`${apiURL}/api/v1/optimization/approve`, { headers, data });

    expect(first.status()).toBe(201);
    expect([200, 201]).toContain(second.status());
    expect(Number(scalar('SELECT COUNT(*) FROM jw_optimization_approvals'))).toBe(before + 1);
    expect(scalar(`SELECT COUNT(*) FROM jw_optimization_approvals WHERE notes='${notes}'`)).toBe('1');
  });

  test('invalid negative approval DTO is rejected without persistence', async ({ request }) => {
    const { token } = await login(request, 'hat_yoneticisi');
    const notes = `CODEX-INVALID-APPROVAL-${Date.now()}`;
    const before = scalar('SELECT COUNT(*) FROM jw_optimization_approvals');
    const response = await request.post(`${apiURL}/api/v1/optimization/approve`, {
      headers: { ...bearer(token), 'Idempotency-Key': `CODEX-INVALID-IDEMP-${Date.now()}` },
      data: {
        line_id: 'LINE_500T', current_headway_minutes: -15, target_headway_minutes: -8,
        fleet_capacity: -75, vehicle_delta: -5, delta_cost_kurus: -1,
        projected_net_margin_kurus: -1, notes
      }
    });

    expect([400, 422]).toContain(response.status());
    expect(scalar('SELECT COUNT(*) FROM jw_optimization_approvals')).toBe(before);
    expect(scalar(`SELECT COUNT(*) FROM jw_optimization_approvals WHERE notes='${notes}'`)).toBe('0');
  });
});
