import { test, expect } from '@playwright/test';

async function loginAs(page, roleName) {
  await page.getByRole('button', { name: /Giriş Yap/ }).click();
  const dialog = page.getByRole('dialog', { name: /AktarBilet Giriş/ });
  await dialog.getByRole('button', { name: roleName }).click();
  await dialog.getByRole('button', { name: /Giriş Yap/ }).click();
  await expect(dialog).toBeHidden();
}

test.describe('browser UAT against the real frontend', () => {
  for (const viewport of [
    { name: 'desktop', width: 1440, height: 900 },
    { name: 'tablet', width: 768, height: 1024 },
    { name: 'mobile', width: 390, height: 844 }
  ]) {
    test(`@smoke ${viewport.name} renders without horizontal overflow`, async ({ page }) => {
      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      await page.goto('/');
      await expect(page.getByText('AktarBilet', { exact: true })).toBeVisible();
      const dimensions = await page.evaluate(() => ({
        scrollWidth: document.documentElement.scrollWidth,
        clientWidth: document.documentElement.clientWidth
      }));
      expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth + 1);
    });
  }

  test('role dashboards require login instead of client-side role switching', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('button', { name: /Kontrolör/ })).toHaveCount(0);
    await expect(page.getByRole('button', { name: /Muhasebe/ })).toHaveCount(0);
    await expect(page.getByRole('button', { name: /Operasyon/ })).toHaveCount(0);
    await expect(page.getByRole('button', { name: /Giriş/ })).toBeVisible();
  });

  test('recharge failure never changes displayed balance or emits success', async ({ page }) => {
    await page.route('**/api/v1/cards/**/recharge', (route) => route.abort('connectionfailed'));
    await page.goto('/');
    await loginAs(page, /Yolcu \(Mehmet Demir\)/);
    const balance = page.locator('.card-balance-value');
    await expect(balance).not.toContainText('Yükleniyor');
    const before = await balance.textContent();
    await page.getByRole('button', { name: '+50 TL Yükle' }).click();
    await expect(balance).toHaveText(before);
    await expect(page.getByText(/başarıyla yüklendi/i)).toHaveCount(0);
    await expect(page.getByText(/tamamlanamadı|hata|ulaşılamadı|başarısız/i)).toBeVisible();
  });

  test('turnstile action calls the validator API before showing success', async ({ page }) => {
    const tapRequest = page.waitForRequest((request) =>
      request.url().includes('/api/v1/validators/tap') && request.method() === 'POST',
    { timeout: 5_000 });
    await page.goto('/');
    await loginAs(page, /Yolcu \(Mehmet Demir\)/);
    await page.getByRole('button', { name: '5 ESHOT', exact: true }).click();
    await tapRequest;
    await expect(page.getByText(/okutuldu/)).toBeVisible();
  });

  test('penalty creation performs a server request', async ({ page }) => {
    await page.goto('/');
    await loginAs(page, /Kontrolör \(Ali Saha\)/);
    const penaltyRequest = page.waitForRequest((request) =>
      request.url().includes('/api/v1/penalties') && request.method() === 'POST',
    { timeout: 5_000 });
    await page.getByLabel('İhlal Yapan Kart ID').fill('CARD_YOLCU_001');
    await page.getByLabel('Ceza Tutarı (TL)').fill('10');
    await page.getByRole('button', { name: 'Ceza Tutanağı Düzenle' }).click();
    await penaltyRequest;
  });

  test('settlement close performs a server request and survives refresh', async ({ page }) => {
    await page.goto('/');
    await loginAs(page, /Muhasebe \(Canan Finans\)/);
    const closeRequest = page.waitForRequest((request) =>
      request.url().includes('/api/v1/settlements/close') && request.method() === 'POST',
    { timeout: 5_000 });
    await page.getByRole('button', { name: /Dönemi Kapat ve Mutabakatı Kesinleştir/ }).click();
    await closeRequest;
    await page.reload();
    await expect(page.getByText(/kapalı dönem/i)).toBeVisible();
  });

  test('heatmap backend outage produces an error state, not fabricated data', async ({ page }) => {
    await page.route('**/api/v1/reports/heatmap', (route) => route.abort('connectionfailed'));
    await page.goto('/');
    await page.getByRole('button', { name: /Isı Haritası/ }).click();
    await expect(page.getByText(/veri.*alınamadı|hata|yeniden dene/i)).toBeVisible();
    await expect(page.getByText('2450')).toHaveCount(0);
  });

  test('successful recharge is reloaded from the server after refresh', async ({ page }) => {
    await page.goto('/');
    await loginAs(page, /Yolcu \(Mehmet Demir\)/);
    const balance = page.locator('.card-balance-value');
    await expect(balance).not.toContainText('Yükleniyor');
    const toKurus = (value) => Number(value.replace(/\./g, '').replace(/[^\d,]/g, '').replace(',', ''));
    const before = toKurus(await balance.textContent());
    const rechargeRequest = page.waitForResponse((response) => response.url().includes('/recharge') && response.request().method() === 'POST');
    await page.getByRole('button', { name: '+50 TL Yükle' }).click();
    expect((await rechargeRequest).ok()).toBe(true);
    await page.reload();
    await expect.poll(async () => toKurus(await balance.textContent())).toBe(before + 5000);
  });
});
