import { defineConfig, devices } from '@playwright/test';

const configuredBaseURL = process.env.PLAYWRIGHT_BASE_URL;
const baseURL = configuredBaseURL || 'http://127.0.0.1:4173';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI
    ? [['github'], ['line'], ['html', { open: 'never' }]]
    : 'list',
  outputDir: 'test-results',
  snapshotPathTemplate: '{testDir}/__screenshots__/{testFilePath}/{arg}{ext}',
  expect: {
    toHaveScreenshot: {
      animations: 'disabled',
      maxDiffPixelRatio: 0.01,
    },
  },
  use: {
    baseURL,
    colorScheme: 'light',
    locale: 'en-US',
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },
  webServer: configuredBaseURL
    ? undefined
    : {
        command:
          'yarn build && exec node node_modules/serve/build/main.js -s build -l 4173',
        env: {
          BROWSER: 'none',
          CI: 'false',
          DISABLE_ESLINT_PLUGIN: 'true',
        },
        reuseExistingServer: false,
        timeout: 180_000,
        url: `${baseURL}/auth/signin`,
      },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
