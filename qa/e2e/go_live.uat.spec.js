import { test, expect } from '@playwright/test';

test.describe('go-live browser acceptance', () => {
  test('UAT-03 role dashboards remain inaccessible before explicit login', async ({ page }) => {
    await page.goto('/');
    await page.waitForTimeout(800);
    await expect(page.getByRole('button', { name: /Kontrolör/ })).toHaveCount(0);
    await expect(page.getByRole('button', { name: /Muhasebe/ })).toHaveCount(0);
    await expect(page.getByRole('button', { name: /Operasyon/ })).toHaveCount(0);
  });

  test('UAT-01 passenger action does not silently log in with embedded credentials', async ({ page }) => {
    let loginRequests = 0;
    await page.route('**/api/v1/auth/login', async (route) => {
      loginRequests += 1;
      await route.fulfill({ status: 401, contentType: 'application/json', body: '{"error":{"code":"UNAUTHORIZED","message":"login required"}}' });
    });
    await page.goto('/');
    await page.evaluate(() => localStorage.clear());
    await page.reload();
    await page.getByRole('button', { name: 'Yolcu Portalı' }).click();
    expect(loginRequests, 'the browser bundle contains and submits demo credentials').toBe(0);
    await expect(page.getByRole('dialog', { name: /AktarBilet Giriş/ })).toBeVisible();
  });

  test('UAT-04 storno control performs a server-side correction request', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: /Giriş Yap/ }).click();
    const dialog = page.getByRole('dialog', { name: /AktarBilet Giriş/ });
    await dialog.getByRole('button', { name: /Muhasebe \(Canan Finans\)/ }).click();
    await dialog.getByRole('button', { name: /Giriş Yap/ }).click();
    await expect(dialog).toBeHidden();
    const requestPromise = page.waitForRequest((request) =>
      request.url().includes('/api/v1/settlements/storno') && request.method() === 'POST',
    { timeout: 2_000 });
    await page.getByRole('button', { name: /Storno Düzeltme/ }).click();
    await requestPromise;
  });
});
