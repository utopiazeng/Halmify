import { expect, test } from '@playwright/test'
import { loginByAPI } from './helpers/auth.js'

const png1x1 = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAFgwJ/luz7GQAAAABJRU5ErkJggg==',
  'base64'
)

test('产品图片上传后在详情区显示', async ({ page }) => {
  test.setTimeout(120000)
  const login = await loginByAPI(page, process.env.E2E_USERNAME || 'utopia01', process.env.E2E_PASSWORD || 'E2e@20260523!', 'zh-CN')

  const name = `E2E-产品图片-${Date.now()}`
  const createResp = await page.request.post('/api/v1/prd/products/objects', {
    headers: { Authorization: `Bearer ${login.token}` },
    data: {
      object_type: 'product',
      name,
      code: `${name}-SKU`,
      status: 'on_sale',
      amount: 99,
      currency: 'CNY',
      payload: {
        sku: `${name}-SKU`,
        category: 'E2E',
        unit: '件',
        stock: 10,
        stock_threshold: 1,
        tax_rate: 0
      }
    }
  })
  expect(createResp.ok()).toBeTruthy()

  await page.goto('/products')
  await page.getByPlaceholder('产品名称、SKU、分类').fill(name)
  await page.getByRole('button', { name: '查询' }).click()
  await expect(page.getByText(name).first()).toBeVisible({ timeout: 15000 })
  await page.getByText(name).first().click()
  await expect(page.locator('.product-image-panel')).toBeVisible()

  await page.locator('.product-image-panel input[type="file"]').setInputFiles({
    name: 'product-e2e.png',
    mimeType: 'image/png',
    buffer: png1x1
  })

  await expect(page.getByText('产品图片已上传')).toBeVisible({ timeout: 30000 })
  const image = page.locator('.product-image-frame img')
  await expect(image).toBeVisible({ timeout: 30000 })
  await expect.poll(async () => image.evaluate((img) => img.complete && img.naturalWidth > 0)).toBeTruthy()
})
