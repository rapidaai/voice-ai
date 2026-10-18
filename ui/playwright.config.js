const { defineConfig, devices } = require('@playwright/test');

const defaultPort = process.env.PORT || '3010';
const baseURL =
  process.env.PLAYWRIGHT_BASE_URL || `http://127.0.0.1:${defaultPort}`;
const serverURL = new URL(baseURL);

module.exports = defineConfig({
  testDir: './e2e',
  testMatch: '**/*.spec.js',
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['list'],
    ['html', { open: 'never', outputFolder: 'e2e/reports/playwright' }],
  ],
  outputDir: './test-results/e2e',
  snapshotPathTemplate: '{testDir}/screenshots/{arg}{ext}',
  timeout: 45_000,
  expect: {
    timeout: 10_000,
    // Filter platform-specific glyph rasterization while preserving area limits.
    toHaveScreenshot: {
      threshold: 0.5,
    },
  },
  use: {
    baseURL,
    timezoneId: 'UTC',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    reducedMotion: 'reduce',
  },
  webServer: process.env.PLAYWRIGHT_SKIP_WEB_SERVER
    ? undefined
    : {
        command: 'yarn e2e:server',
        url: baseURL,
        reuseExistingServer: false,
        timeout: 180_000,
        env: {
          HOST: serverURL.hostname,
          PORT: serverURL.port || defaultPort,
        },
      },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
