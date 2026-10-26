import { test, expect } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const qaRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const axePath = path.join(qaRoot, 'node_modules', 'axe-core', 'axe.min.js');

async function audit(page, label) {
  await page.addScriptTag({ path: axePath });
  const result = await page.evaluate(async () => window.axe.run(document, {
    runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa'] },
    resultTypes: ['violations']
  }));
  const evidence = result.violations.map((v) => ({
    id: v.id,
    impact: v.impact,
    nodes: v.nodes.length,
    targets: v.nodes.slice(0, 3).map((n) => n.target.join(' '))
  }));
  expect(evidence, `${label}: ${JSON.stringify(evidence)}`).toEqual([]);
}

test.describe('Retest 3 WCAG 2.2 AA axe audit', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/auth/login', (route) => route.abort());
  });

  test('desktop passenger portal', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/');
    await audit(page, 'desktop passenger portal');
  });

  test('desktop login dialog', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/');
    await page.getByRole('button', { name: /Giriş Yap/ }).click();
    await audit(page, 'desktop login dialog');
  });

  test('mobile passenger portal', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/');
    await audit(page, 'mobile passenger portal');
  });

  test('map view semantic fallback', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/');
    await page.getByRole('button', { name: /Canlı Harita.*ESHOT/ }).click();
    await audit(page, 'map view');
  });

  test('keyboard opens login dialog and moves focus inside it', async ({ page }) => {
    await page.goto('/');
    const trigger = page.getByRole('button', { name: /Giriş Yap/ });
    await trigger.focus();
    await page.keyboard.press('Enter');
    const dialog = page.getByRole('dialog', { name: /AktarBilet Giriş/ });
    await expect(dialog).toBeVisible();
    expect(await dialog.evaluate((node) => node.contains(document.activeElement))).toBe(true);
  });

  test('login dialog Escape closes and restores trigger focus', async ({ page }) => {
    await page.goto('/');
    const trigger = page.getByRole('button', { name: /Giriş Yap/ });
    await trigger.click();
    await page.keyboard.press('Escape');
    await expect(page.getByRole('dialog', { name: /AktarBilet Giriş/ })).toBeHidden();
    await expect(trigger).toBeFocused();
  });

  test('200 percent zoom does not introduce page-level horizontal overflow', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 900 });
    await page.goto('/');
    await page.evaluate(() => { document.documentElement.style.zoom = '200%'; });
    const dimensions = await page.evaluate(() => ({
      scrollWidth: document.documentElement.scrollWidth,
      clientWidth: document.documentElement.clientWidth
    }));
    expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
  });
});
