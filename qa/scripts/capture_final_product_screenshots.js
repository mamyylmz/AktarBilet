import { chromium } from '@playwright/test';
import path from 'node:path';
import fs from 'node:fs';

const OUT_DIR = path.resolve('artifacts/final-product');
if (!fs.existsSync(OUT_DIR)) {
  fs.mkdirSync(OUT_DIR, { recursive: true });
}

const VIEWPORTS = [
  { name: '360x800', width: 360, height: 800 },
  { name: '390x844', width: 390, height: 844 },
  { name: '768x1024', width: 768, height: 1024 },
  { name: '1440x900', width: 1440, height: 900 },
  { name: '1920x1080', width: 1920, height: 1080 },
];

const THEMES = ['light', 'dark'];

const ROLES = [
  { role: 'yolcu', email: 'yolcu@aktarbilet.gov.tr', pass: 'YolcuPassword123!' },
  { role: 'kontrolor', email: 'kontrolor@aktarbilet.gov.tr', pass: 'KontrolorPassword123!' },
  { role: 'hat_yoneticisi', email: 'hatyoneticisi@aktarbilet.gov.tr', pass: 'HatYoneticisi123!' },
  { role: 'muhasebe', email: 'muhasebe@aktarbilet.gov.tr', pass: 'MuhasebePassword123!' },
  { role: 'operasyon', email: 'operasyon@aktarbilet.gov.tr', pass: 'OperasyonPassword123!' },
  { role: 'admin', email: 'seed.sedir@ddo-case.invalid', pass: 'SedirPassword2026!' },
];

async function main() {
  const browser = await chromium.launch({ headless: true });
  console.log('Starting comprehensive visual verification screenshot generator...');

  for (const theme of THEMES) {
    console.log(`\n=== Generating Theme: ${theme.toUpperCase()} ===`);

    for (const vp of VIEWPORTS) {
      console.log(`Viewport: ${vp.name}...`);
      const context = await browser.newContext({
        viewport: { width: vp.width, height: vp.height },
      });
      const page = await context.newPage();

      // 1. Home Page (Anonymous)
      await page.goto('http://127.0.0.1:3000/');
      await page.evaluate((t) => {
        localStorage.clear();
        localStorage.setItem('aktarbilet_theme', t);
        document.documentElement.setAttribute('data-theme', t);
      }, theme);
      await page.reload();
      await page.waitForTimeout(300);
      await page.screenshot({
        path: path.join(OUT_DIR, `01_homepage_${vp.name}_${theme}.png`),
        fullPage: true,
      });

      // 2. Login Dialog
      await page.getByRole('button', { name: /Giriş/i }).first().click();
      await page.waitForTimeout(200);
      await page.screenshot({
        path: path.join(OUT_DIR, `02_login_modal_${vp.name}_${theme}.png`),
      });
      await page.keyboard.press('Escape');
      await page.waitForTimeout(100);

      // 3. Authenticated Portals
      for (const r of ROLES) {
        const authRes = await page.request.post('http://127.0.0.1:3000/api/v1/auth/login', {
          data: { email: r.email, password: r.pass },
        });
        if (authRes.ok()) {
          const authData = await authRes.json();
          await page.evaluate(({ token, user, role, theme }) => {
            localStorage.setItem('auth_token', token);
            localStorage.setItem('auth_user', JSON.stringify(user));
            localStorage.setItem('active_role', role);
            localStorage.setItem('aktarbilet_theme', theme);
            document.documentElement.setAttribute('data-theme', theme);
          }, { token: authData.access_token, user: authData.user, role: r.role, theme });

          await page.goto('http://127.0.0.1:3000/');
          await page.waitForTimeout(300);

          await page.screenshot({
            path: path.join(OUT_DIR, `portal_${r.role}_${vp.name}_${theme}.png`),
            fullPage: true,
          });
        }
      }

      await context.close();
    }
  }

  await browser.close();
  console.log(`\nVisual snapshots completed in ${OUT_DIR}`);
}

main().catch((err) => {
  console.error('Snapshot error:', err);
  process.exit(1);
});

