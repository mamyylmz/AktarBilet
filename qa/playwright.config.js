import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  testMatch: [
    'tests/**/*.spec.js',
    'api/**/*.spec.js',
    'db/**/*.spec.js',
    'e2e/**/*.spec.js',
    'security/**/*.spec.js'
  ],
  fullyParallel: false,
  workers: 1,
  timeout: 30_000,
  expect: { timeout: 5_000 },
  outputDir: './artifacts/playwright',
  reporter: [
    ['list'],
    ['junit', { outputFile: './artifacts/results.xml' }],
    ['json', { outputFile: './artifacts/results.json' }],
    ['html', { outputFolder: './artifacts/html-report', open: 'never' }]
  ],
  use: {
    baseURL: process.env.QA_FRONTEND_URL || 'http://127.0.0.1:3000',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure'
  }
});
