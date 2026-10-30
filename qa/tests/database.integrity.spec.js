import { test, expect } from '@playwright/test';
import { rows, scalar, sql } from '../helpers/db.js';

test.describe('PostgreSQL schema and financial invariants', () => {
  test('@smoke PostgreSQL 16 and mandatory table canaries exist', () => {
    expect(scalar('SHOW server_version')).toMatch(/^16\./);
    const tables = rows("SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename");
    expect(tables.length).toBeGreaterThan(0);
    expect(tables.every((name) => name.startsWith('jw_'))).toBe(true);
    expect(tables).toContain('jw_tf_users');
  });

  test('seed cardinalities meet the declared 200k cards and 40 lines', () => {
    expect(Number(scalar('SELECT COUNT(*) FROM jw_cards'))).toBeGreaterThanOrEqual(200000);
    expect(Number(scalar('SELECT COUNT(*) FROM jw_lines'))).toBeGreaterThanOrEqual(40);
    expect(Number(scalar('SELECT COUNT(*) FROM jw_tf_users'))).toBeGreaterThanOrEqual(6);
  });

  test('card balance equals the complete ledger sum for every card', () => {
    const mismatches = scalar(`
      SELECT COUNT(*) FROM (
        SELECT c.id, c.balance_kurus, COALESCE(SUM(l.amount_kurus),0) AS ledger_sum
        FROM jw_cards c LEFT JOIN jw_ledger l ON l.card_id=c.id
        GROUP BY c.id, c.balance_kurus
        HAVING c.balance_kurus <> COALESCE(SUM(l.amount_kurus),0)
      ) q
    `);
    expect(mismatches).toBe('0');
  });

  test('ledger is database-enforced append-only', () => {
    const id = scalar("SELECT id FROM jw_ledger ORDER BY created_at LIMIT 1");
    expect(id).toBeTruthy();
    expect(() => sql(`BEGIN; UPDATE jw_ledger SET amount_kurus=amount_kurus+1 WHERE id='${id}'; ROLLBACK`)).toThrow();
    expect(() => sql(`BEGIN; DELETE FROM jw_ledger WHERE id='${id}'; ROLLBACK`)).toThrow();
  });

  test('currency and percentage constraints reject invalid values', () => {
    expect(() => sql("BEGIN; UPDATE jw_tariffs SET base_fare_kurus=-1 WHERE id='TARIFF_IZMIR_STD'; ROLLBACK"))
      .toThrow();
    expect(() => sql("BEGIN; UPDATE jw_tariffs SET transfer1_discount_pct=101 WHERE id='TARIFF_IZMIR_STD'; ROLLBACK"))
      .toThrow();
  });

  test('critical idempotency references are unique in the ledger', () => {
    const duplicates = scalar(`
      SELECT COUNT(*) FROM (
        SELECT reference_id FROM jw_ledger
        WHERE reference_id IS NOT NULL
        GROUP BY reference_id HAVING COUNT(*) > 1
      ) d
    `);
    expect(duplicates).toBe('0');
  });

  test('materialized heatmap agrees with current validator events after refresh', () => {
    sql('REFRESH MATERIALIZED VIEW jw_mv_line_heatmap');
    const mismatch = scalar(`
      SELECT COUNT(*) FROM jw_mv_line_heatmap m
      WHERE m.tap_count <> (
        SELECT COUNT(*) FROM jw_validator_events e
        WHERE e.line_id=m.line_id AND e.tapped_at >= NOW()-INTERVAL '24 hours' AND e.status='PROCESSED'
      )
    `);
    expect(mismatch).toBe('0');
  });
});
