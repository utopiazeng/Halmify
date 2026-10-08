import { expect, test } from '@playwright/test'
import { fetchCaptchaToken } from './helpers/auth.js'

const admin = {
  username: process.env.E2E_USERNAME || 'admin',
  password: process.env.E2E_PASSWORD || '123456'
}

test('滑块通过后才能登录', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('textbox', { name: /账号|Account/i }).fill(admin.username)
  await page.getByRole('textbox', { name: /密码|Password/i }).fill(admin.password)

  const loginBtn = page.getByRole('button', { name: /登\s*录|Sign in/i })
  await expect(loginBtn).toBeDisabled()
  await expect(page.locator('.slider-text')).toContainText(/按住滑块拖动到最右侧|Hold and drag slider to the far right/)

  const captchaToken = await fetchCaptchaToken(page.request)

  await page.evaluate((token) => {
    const root = document.querySelector('.art-login-page')
    const inst = root && root.__vueParentComponent
    if (!inst || !inst.setupState || !inst.setupState.sliderState) {
      throw new Error('slider state not found')
    }
    const state = inst.setupState.sliderState
    state.verified = true
    state.error = ''
    state.handleLeft = inst.setupState.maxLeft
    inst.setupState.form.captcha_token = token
  }, captchaToken)

  await expect(page.locator('.slider-verify.verified')).toBeVisible({ timeout: 10000 })
  await expect(loginBtn).toBeEnabled()
})
