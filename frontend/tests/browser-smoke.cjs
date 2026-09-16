// Optional real-browser checks. Requires Playwright on NODE_PATH and a local
// Vite server. The Wails bridge is mocked; backend integration has separate Go tests.
const { chromium } = require('playwright')
const assert = require('node:assert/strict')

;(async () => {
  const browser = await chromium.launch({ headless: true })
  try {
    const page = await browser.newPage({ viewport: { width: 1180, height: 760 }, timezoneId: 'Europe/Warsaw' })
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    page.on('console', message => { if (message.type() === 'error') errors.push(message.text()) })
    await page.addInitScript(() => {
      localStorage.setItem('kursomat-auto-convert', 'false')
      window.calls = []
      window.convertDelay = 0
      window.failConversion = false
      window.runtime = { ClipboardSetText: async () => true }
      const config = { cache_path: 'test.db', timeout_seconds: 10, retry_count: 0, max_lookback_days: 92, verbose: false, last_from_date: '', last_converter_date: '2026-04-14' }
      window.go = { main: { DesktopApp: {
        GetDashboard: async () => ({ config_path: 'test.json', config, cache: {entries: 1, currency_count: 1, query_mappings: 0, size_bytes: 0, last_saved_at: ''}, currencies: [] }),
        GetCurrencies: async () => [{code: 'USD', name: 'dolar amerykański'}, {code: 'EUR', name: 'euro'}],
        Convert: async request => {
          window.calls.push(request)
          await new Promise(resolve => setTimeout(resolve, window.convertDelay))
          if (window.failConversion) throw new Error('Testowy błąd API')
          const multiply = request.direction === 'foreign_to_pln'
          const value = multiply ? request.amount * 3.6015 : request.amount / 3.6015
          return { source_amount: request.amount, source_currency: multiply ? request.currency : 'PLN', target_amount: Math.round(value * 10000) / 10000, target_currency: multiply ? 'PLN' : request.currency, rate: { currency: request.currency, requested_date: request.date, effective_rate_date: '2026-04-14', mid: 3.6015, table_no: '071/A/NBP/2026', source: 'cache' } }
        },
      } } }
    })
    await page.goto(process.env.KURSOMAT_TEST_URL || 'http://127.0.0.1:34115')
    const amount = page.getByRole('textbox', { name: /Kwota źródłowa/ })
    const convert = page.getByRole('button', { name: 'Przelicz kwotę' })
    const result = page.locator('.conversion-result')
    await amount.fill('100,00')
    await convert.click()
    await result.waitFor()
    assert.match(await result.innerText(), /27,7662/)
    await page.getByRole('button', { name: 'Waluta na PLN', exact: true }).click()
    assert.equal(await result.count(), 0)
    await convert.click()
    await result.waitFor()
    assert.match(await result.innerText(), /360,15/)
    const beforeInvalid = await page.evaluate(() => window.calls.length)
    for (const invalid of ['', '0x10']) {
      await amount.fill(invalid)
      await convert.click()
      assert.equal(await result.count(), 0)
      assert.equal(await page.evaluate(() => window.calls.length), beforeInvalid)
    }
    // A pending old response must not repopulate a result after an input edit.
    await page.evaluate(() => { window.convertDelay = 600 })
    await amount.fill('10')
    await convert.click()
    await page.waitForFunction(n => window.calls.length > n, beforeInvalid)
    await amount.fill('20')
    await page.waitForTimeout(750)
    assert.equal(await result.count(), 0)
    await page.evaluate(() => { window.convertDelay = 0 })
    await convert.click()
    await result.waitFor()
    assert.match(await result.innerText(), /72,03/)
    await page.evaluate(() => { window.failConversion = true })
    await convert.click()
    await page.getByText('Testowy błąd API', { exact: true }).waitFor()
    assert.equal(await result.count(), 0)
    await page.evaluate(() => { window.failConversion = false })
    await page.getByRole('checkbox', { name: 'Przeliczaj automatycznie' }).check()
    await amount.fill('30,50')
    await result.waitFor()
    assert.match(await result.innerText(), /109,8458/)
    assert.equal(await page.getByText('Testowy błąd API', { exact: true }).count(), 0)
    await page.getByRole('button', { name: 'Waluta na PLN', exact: true }).click()
    assert.equal(await result.count(), 1)
    assert.deepEqual(errors, [])
    if (process.env.KURSOMAT_SCREENSHOT) await page.screenshot({ path: process.env.KURSOMAT_SCREENSHOT })
    console.log('PASS: both directions, invalid inputs, stale response, API failure, automatic conversion, clean console')
  } finally { await browser.close() }
})().catch(error => { console.error(error); process.exitCode = 1 })
