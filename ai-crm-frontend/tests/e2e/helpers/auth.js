import { expect } from '@playwright/test'

const sliderTrack = [0, 7, 15, 26, 39, 51, 64, 78, 89, 96, 100]

export async function fetchCaptchaToken(request) {
  const challengeResp = await request.get('/api/v1/auth/captcha/challenge')
  expect(challengeResp.ok()).toBeTruthy()
  const challengeBody = await challengeResp.json()
  const challengeID = challengeBody?.data?.challenge_id
  expect(challengeID).toBeTruthy()

  const verifyResp = await request.post('/api/v1/auth/captcha/verify', {
    data: {
      challenge_id: challengeID,
      elapsed_ms: 900,
      track: sliderTrack,
      final_percent: 100
    }
  })
  expect(verifyResp.ok()).toBeTruthy()
  const verifyBody = await verifyResp.json()
  const captchaToken = verifyBody?.data?.captcha_token
  expect(captchaToken).toBeTruthy()
  return captchaToken
}

export async function loginByAPI(page, username, password, locale = 'zh-CN') {
  const captchaToken = await fetchCaptchaToken(page.request)
  const loginResp = await page.request.post('/api/v1/auth/login', {
    data: { username, password, captcha_token: captchaToken }
  })
  if (!loginResp.ok()) {
    const body = await loginResp.json().catch(() => ({}))
    throw new Error(`login failed for ${username}: status=${loginResp.status()} message=${body?.message || 'unknown'}`)
  }
  const loginBody = await loginResp.json()
  const token = loginBody?.data?.token
  const user = loginBody?.data?.user
  expect(token).toBeTruthy()
  expect(user?.id).toBeTruthy()
  user.language_pref = locale
  await page.goto('/')
  await page.evaluate(({ tokenValue, userValue, localeValue }) => {
    localStorage.setItem('token', tokenValue)
    localStorage.setItem('current_user', JSON.stringify(userValue || {}))
    localStorage.setItem('locale', localeValue)
  }, { tokenValue: token, userValue: user, localeValue: locale })
  return { token, user }
}
