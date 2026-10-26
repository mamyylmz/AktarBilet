import { chromium } from 'playwright';
import fs from 'node:fs';
import path from 'node:path';

const BASE_URL = process.env.QA_FRONTEND_URL || 'http://127.0.0.1:13000';
const OUT_DIR = path.resolve('artifacts/agy-final-qa/screenshots');
const RESULTS_FILE = path.resolve('artifacts/agy-final-qa/viewport_audit_results.json');

fs.mkdirSync(OUT_DIR, { recursive: true });

const viewports = [
  { name: 'iphone_se_320x568', width: 320, height: 568 },
  { name: 'android_360x800', width: 360, height: 800 },
  { name: 'iphone_390x844', width: 390, height: 844 },
  { name: 'tablet_port_768x1024', width: 768, height: 1024 },
  { name: 'tablet_land_1024x768', width: 1024, height: 768 },
  { name: 'desktop_1440x900', width: 1440, height: 900 },
  { name: 'desktop_fhd_1920x1080', width: 1920, height: 1080 },
];

async function run() {
  const browser = await chromium.launch({ headless: true });
  const results = [];

  for (const vp of viewports) {
    for (const theme of ['light', 'dark']) {
      const context = await browser.newContext({
        viewport: { width: vp.width, height: vp.height },
      });
      const page = await context.newPage();

      // Pre-set theme in localStorage
      await page.addInitScript((th) => {
        localStorage.setItem('aktarbilet_theme', th);
        document.documentElement.setAttribute('data-theme', th);
      }, theme);

      await page.goto(BASE_URL, { waitUntil: 'networkidle' });

      // Check header height
      const headerBox = await page.locator('header').boundingBox();
      const headerHeight = headerBox ? headerBox.height : 0;

      // Check horizontal overflow
      const overflow = await page.evaluate(() => {
        const doc = document.documentElement;
        return {
          scrollWidth: doc.scrollWidth,
          clientWidth: doc.clientWidth,
          hasOverflow: doc.scrollWidth > doc.clientWidth,
        };
      });

      // Check layout details
      const layoutDetails = await page.evaluate(() => {
        const gridEl = document.querySelector('.grid');
        const gridDisplay = gridEl ? window.getComputedStyle(gridEl).display : null;
        const navMenu = document.querySelector('.nav-desktop-menu');
        const navMenuDisplay = navMenu ? window.getComputedStyle(navMenu).display : null;
        const hamburger = document.querySelector('.mobile-menu-trigger');
        const hamburgerDisplay = hamburger ? window.getComputedStyle(hamburger).display : null;

        return {
          gridDisplay,
          navMenuDisplay,
          hamburgerDisplay,
        };
      });

      const shotName = `${vp.name}_${theme}.png`;
      const shotPath = path.join(OUT_DIR, shotName);
      await page.screenshot({ path: shotPath, fullPage: true });

      const itemResult = {
        viewport: vp.name,
        width: vp.width,
        height: vp.height,
        theme,
        headerHeight: Math.round(headerHeight),
        horizontalOverflow: overflow.hasOverflow,
        scrollWidth: overflow.scrollWidth,
        clientWidth: overflow.clientWidth,
        layoutDetails,
        screenshot: shotName,
      };

      console.log(`[VIEWPORT] ${vp.name} (${theme}): Header=${Math.round(headerHeight)}px, Overflow=${overflow.hasOverflow}, Grid=${layoutDetails.gridDisplay}, Hamburger=${layoutDetails.hamburgerDisplay}`);
      results.push(itemResult);

      await context.close();
    }
  }

  await browser.close();

  fs.writeFileSync(RESULTS_FILE, JSON.stringify(results, null, 2));
  console.log(`\nAudit complete. Saved ${results.length} viewport records to ${RESULTS_FILE}`);
}

run().catch((err) => {
  console.error('Error during viewport audit:', err);
  process.exit(1);
});

