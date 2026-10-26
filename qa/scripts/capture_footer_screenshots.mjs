import http from 'http';
import fs from 'fs';
import path from 'path';
import pkg from '../node_modules/playwright/index.js';
const { chromium } = pkg;

const distDir = path.resolve('web/dist');

// Simple static server
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
    // SPA fallback
    const indexPath = path.join(distDir, 'index.html');
    res.writeHead(200, { 'Content-Type': 'text/html' });
    fs.createReadStream(indexPath).pipe(res);
  }
});

server.listen(19876, async () => {
  console.log('Serving on http://localhost:19876');
  const browser = await chromium.launch({ headless: true });

  const outDir = path.resolve('qa/artifacts/screenshots');
  if (!fs.existsSync(outDir)) {
    fs.mkdirSync(outDir, { recursive: true });
  }

  // 1. Desktop Light (1280x800)
  const pageDesktop = await browser.newPage({ viewport: { width: 1280, height: 800 } });
  await pageDesktop.goto('http://localhost:19876');
  await pageDesktop.waitForSelector('.app-footer');
  await pageDesktop.evaluate(() => document.documentElement.setAttribute('data-theme', 'light'));
  
  const footerElem = await pageDesktop.$('.app-footer');
  if (footerElem) {
    await footerElem.screenshot({ path: path.join(outDir, 'footer-desktop-light.png') });
  }
  const navbarElem = await pageDesktop.$('.navbar');
  if (navbarElem) {
    await navbarElem.screenshot({ path: path.join(outDir, 'navbar-desktop-public.png') });
  }

  // 2. Desktop Dark (1280x800)
  await pageDesktop.evaluate(() => document.documentElement.setAttribute('data-theme', 'dark'));
  if (footerElem) {
    await footerElem.screenshot({ path: path.join(outDir, 'footer-desktop-dark.png') });
  }

  await pageDesktop.close();

  // 3. Mobile Light (360x740)
  const pageMobile = await browser.newPage({ viewport: { width: 360, height: 740 } });
  await pageMobile.goto('http://localhost:19876');
  await pageMobile.waitForSelector('.app-footer');
  await pageMobile.evaluate(() => document.documentElement.setAttribute('data-theme', 'light'));

  const footerMobileElem = await pageMobile.$('.app-footer');
  if (footerMobileElem) {
    await footerMobileElem.screenshot({ path: path.join(outDir, 'footer-mobile-light.png') });
  }
  const navbarMobileElem = await pageMobile.$('.navbar');
  if (navbarMobileElem) {
    await navbarMobileElem.screenshot({ path: path.join(outDir, 'navbar-mobile-public.png') });
  }

  // 4. Mobile Dark (360x740)
  await pageMobile.evaluate(() => document.documentElement.setAttribute('data-theme', 'dark'));
  if (footerMobileElem) {
    await footerMobileElem.screenshot({ path: path.join(outDir, 'footer-mobile-dark.png') });
  }

  await pageMobile.close();
  await browser.close();
  server.close();
  console.log('Screenshots captured successfully in qa/artifacts/screenshots/');
});
