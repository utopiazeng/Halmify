import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './tests/e2e/prod-smoke',
  timeout: 120000,
  expect: {
    timeout: 15000
  },
  fullyParallel: false,
  workers: 1,
  reporter: [
    ['list'],
    ['html', { outputFolder: 'playwright-report-prod-smoke', open: 'never' }]
  ],
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL || 'https://crm.halmify.com',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'off',
    userAgent: process.env.E2E_USER_AGENT || 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36'
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: {
          executablePath: process.env.PLAYWRIGHT_CHROMIUM_PATH || '/usr/bin/google-chrome-stable',
          args: ['--no-sandbox']
        }
      }
    }
  ]
})
