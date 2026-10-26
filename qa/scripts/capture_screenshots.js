import { chromium } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const outputDir = path.resolve(__dirname, '..', 'artifacts', 'frontend');
const baseURL = process.env.QA_FRONTEND_URL || 'http://127.0.0.1:13000';

async function capture() {
  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });
  const page = await context.newPage();

  console.log('Navigating to', baseURL);
  await page.goto(baseURL);
  await page.waitForTimeout(1500);

  // 1. Passenger Portal (Default View)
  console.log('Capturing 01_passenger_portal_desktop.png...');
  await page.screenshot({ path: path.join(outputDir, '01_passenger_portal_desktop.png'), fullPage: true });

  // 2. Mobile View
  console.log('Capturing 02_passenger_portal_mobile.png...');
  await page.setViewportSize({ width: 390, height: 844 });
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(outputDir, '02_passenger_portal_mobile.png'), fullPage: true });

  // Reset to desktop
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.waitForTimeout(500);

  // 3. Public Home Page
  console.log('Capturing 03_public_home_page.png...');
  await page.getByRole('button', { name: 'Ana Sayfa', exact: true }).click();
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(outputDir, '03_public_home_page.png'), fullPage: true });

  // 4. Login Dialog
  console.log('Capturing 04_login_dialog.png...');
  await page.getByRole('button', { name: 'Giriş Yap', exact: true }).click();
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(outputDir, '04_login_dialog.png'), fullPage: false });
  await page.keyboard.press('Escape');
  await page.waitForTimeout(300);

  // 5. Inspector Portal
  console.log('Capturing 05_inspector_portal.png...');
  await page.getByRole('button', { name: 'Kontrolör' }).click();
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(outputDir, '05_inspector_portal.png'), fullPage: true });

  // 6. Line Manager Portal
  console.log('Capturing 06_linemanager_portal.png...');
  await page.getByRole('button', { name: 'Hat Yöneticisi' }).click();
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(outputDir, '06_linemanager_portal.png'), fullPage: true });

  // 7. Accounting Portal
  console.log('Capturing 07_accounting_portal.png...');
  await page.getByRole('button', { name: 'Muhasebe' }).click();
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(outputDir, '07_accounting_portal.png'), fullPage: true });

  // 8. Operations Portal & Live Map
  console.log('Capturing 08_operations_portal.png...');
  await page.getByRole('button', { name: 'Operasyon' }).click();
  await page.waitForTimeout(1000);
  await page.screenshot({ path: path.join(outputDir, '08_operations_portal.png'), fullPage: true });

  // 9. Admin Portal
  console.log('Capturing 09_admin_portal.png...');
  await page.getByRole('button', { name: 'Admin' }).click();
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(outputDir, '09_admin_portal.png'), fullPage: true });

  console.log('All screenshots successfully saved to', outputDir);
  await browser.close();
}

capture().catch((err) => {
  console.error('Screenshot capture failed:', err);
  process.exit(1);
});
