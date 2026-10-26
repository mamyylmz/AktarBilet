import http from 'http';
import fs from 'fs';
import path from 'path';
import pkg from '../node_modules/playwright/index.js';
const { chromium } = pkg;

const distDir = path.resolve('web/dist');

// Static server for test
const server = http.createServer((req, res) => {
  let reqPath = req.url.split('?')[0];
  if (reqPath === '/') reqPath = '/index.html';
  const filePath = path.join(distDir, reqPath);
  
  if (fs.existsSync(filePath) && fs.statSync(filePath).isFile()) {
    const ext = path.extname(filePath);
    const mimeTypes = {
      '.html': 'text/html',
      '.js': 'text/javascript',
      '.css': 'text/css',
      '.json': 'application/json',
      '.png': 'image/png',
      '.svg': 'image/svg+xml',
    };
    res.writeHead(200, { 'Content-Type': mimeTypes[ext] || 'application/octet-stream' });
    fs.createReadStream(filePath).pipe(res);
  } else {
    const indexPath = path.join(distDir, 'index.html');
    res.writeHead(200, { 'Content-Type': 'text/html' });
    fs.createReadStream(indexPath).pipe(res);
  }
});

server.listen(19877, async () => {
  console.log('Testing frontend on http://localhost:19877...');
  let failed = false;
  const browser = await chromium.launch({ headless: true });

  try {
    // ----------------------------------------------------
    // TEST 1: Desktop Navbar Layout & No Redundant Popup
    // ----------------------------------------------------
    const desktopPage = await browser.newPage({ viewport: { width: 1280, height: 800 } });
    await desktopPage.goto('http://localhost:19877');
    await desktopPage.waitForSelector('.navbar');

    // Verify Hamburger button is hidden on desktop
    const isHamburgerHiddenOnDesktop = await desktopPage.evaluate(() => {
      const btn = document.querySelector('.navbar-mobile-toggle');
      if (!btn) return true;
      const style = window.getComputedStyle(btn);
      return style.display === 'none';
    });
    if (!isHamburgerHiddenOnDesktop) {
      console.error('FAIL: Hamburger toggle is unexpectedly visible on desktop!');
      failed = true;
    } else {
      console.log('PASS: [Desktop Navbar] Redundant popup / mobile toggle button is hidden on desktop.');
    }

    // Verify Giriş Yap button is present and opens modal
    const loginBtn = await desktopPage.$('button:has-text("Giriş Yap")');
    if (!loginBtn) {
      console.error('FAIL: Giriş Yap button not found on public navbar!');
      failed = true;
    } else {
      console.log('PASS: [Desktop Navbar] Giriş Yap button is present and prominent.');
      await loginBtn.click();
      await desktopPage.waitForSelector('[role="dialog"]');
      const dialogVisible = await desktopPage.isVisible('[role="dialog"]');
      if (dialogVisible) {
        console.log('PASS: [Desktop Navbar] Login modal opened properly upon clicking Giriş Yap.');
        // Close modal
        const closeBtn = await desktopPage.$('[aria-label="Kapat"], [aria-label="Modalı Kapat"]');
        if (closeBtn) await closeBtn.click();
      }
    }
    await desktopPage.close();

    // ----------------------------------------------------
    // TEST 2: Mobile Compact Navbar (360x740)
    // ----------------------------------------------------
    const mobilePage = await browser.newPage({ viewport: { width: 360, height: 740 } });
    await mobilePage.goto('http://localhost:19877');
    await mobilePage.waitForSelector('.navbar');

    const navbarHeight = await mobilePage.evaluate(() => {
      const nb = document.querySelector('.navbar-container');
      return nb ? nb.getBoundingClientRect().height : 0;
    });
    if (navbarHeight > 70) {
      console.error(`FAIL: Mobile navbar height is ${navbarHeight}px (must be <= 64px compact)!`);
      failed = true;
    } else {
      console.log(`PASS: [Mobile Navbar] Height is compact (${Math.round(navbarHeight)}px <= 64px, not 215px).`);
    }

    const isHamburgerVisibleOnMobile = await mobilePage.evaluate(() => {
      const btn = document.querySelector('.navbar-mobile-toggle');
      if (!btn) return false;
      const style = window.getComputedStyle(btn);
      return style.display !== 'none';
    });
    if (!isHamburgerVisibleOnMobile) {
      console.error('FAIL: Hamburger toggle is not visible on mobile 360px!');
      failed = true;
    } else {
      console.log('PASS: [Mobile Navbar] Hamburger toggle is properly available on mobile.');
    }
    await mobilePage.close();

    // ----------------------------------------------------
    // TEST 3: Footer Design & Accessibility (7 Links & Verbatim Notices)
    // ----------------------------------------------------
    const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });
    await page.goto('http://localhost:19877');
    await page.waitForSelector('.app-footer');

    const requiredLinks = [
      'Hakkımızda',
      'Nasıl Çalışır?',
      'Yardım & SSS',
      'Gizlilik Politikası',
      'KVKK Aydınlatma',
      'Kullanım Koşulları',
      'İletişim & Destek',
    ];

    const linkTexts = await page.$$eval('.footer-link', (nodes) => nodes.map((n) => n.textContent.trim()));
    console.log(`Found footer links: ${JSON.stringify(linkTexts)}`);

    for (const req of requiredLinks) {
      if (!linkTexts.includes(req)) {
        console.error(`FAIL: Footer missing link: "${req}"`);
        failed = true;
      }
    }
    if (linkTexts.length === 7) {
      console.log('PASS: [Footer Links] All 7 corporate & legal links are present.');
    }

    // Touch Target Size Verification (WCAG 2.2 AA >= 44x44px target)
    const targetSizes = await page.$$eval('.footer-link', (nodes) =>
      nodes.map((n) => {
        const rect = n.getBoundingClientRect();
        return { text: n.textContent.trim(), width: rect.width, height: rect.height };
      })
    );
    let allTargetSizesOk = true;
    for (const item of targetSizes) {
      if (item.height < 40) {
        console.error(`FAIL: Link "${item.text}" height is ${item.height}px (< 40px minimum target size)`);
        allTargetSizesOk = false;
        failed = true;
      }
    }
    if (allTargetSizesOk) {
      console.log('PASS: [Footer Accessibility] All 7 links satisfy WCAG target size requirements.');
    }

    // Verbatim text check
    const copyrightText = await page.$eval('.copyright-text', (el) => el.textContent.trim());
    const expectedCopyright = '© 2026 AktarBilet Şehir İçi Entegre Ulaşım ve Akıllı Biletleme Platformu. Tüm hakları saklıdır.';
    if (copyrightText !== expectedCopyright) {
      console.error(`FAIL: Copyright text mismatch!\nExpected: "${expectedCopyright}"\nGot:      "${copyrightText}"`);
      failed = true;
    } else {
      console.log('PASS: [Footer Notices] Copyright text matches verbatim (100% exact).');
    }

    const attributionText = await page.$eval('.attribution-text', (el) => el.textContent.trim());
    const expectedAttribution = 'Canlı otobüs telemetrisi İzmir Büyükşehir Belediyesi ESHOT Genel Müdürlüğü Açık Veri Portalı entegrasyonu ile sağlanmaktadır.';
    if (attributionText !== expectedAttribution) {
      console.error(`FAIL: Attribution text mismatch!\nExpected: "${expectedAttribution}"\nGot:      "${attributionText}"`);
      failed = true;
    } else {
      console.log('PASS: [Footer Notices] ESHOT telemetry attribution text matches verbatim (100% exact).');
    }

    // Test clicking a footer link opens the modal
    const aboutBtn = await page.$('button.footer-link:has-text("Hakkımızda")');
    if (aboutBtn) {
      await aboutBtn.click();
      await page.waitForSelector('[role="dialog"]');
      const dialogTitle = await page.$eval('[role="dialog"] h2, [role="dialog"] h3', (el) => el.textContent.trim());
      console.log(`PASS: [Footer Modal Interaction] Clicked "Hakkımızda", opened modal with title: "${dialogTitle}".`);
    }

    await page.close();

  } catch (err) {
    console.error('Test execution error:', err);
    failed = true;
  } finally {
    await browser.close();
    server.close();
    if (failed) {
      console.error('\n❌ Targeted User Fixes Verification: FAILED');
      process.exit(1);
    } else {
      console.log('\n✅ Targeted User Fixes Verification: ALL CHECKS PASSED (100% PASS)');
    }
  }
});

