import process from 'node:process'
import { chromium } from 'playwright'

const baseURL = process.env.BASE_URL ?? 'http://127.0.0.1:5173'
const browser = await chromium.launch({ headless: true })

try {
  const hostContext = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
  const page = await hostContext.newPage()

  await page.goto(`${baseURL}/`, { waitUntil: 'networkidle' })
  await page.getByRole('heading', { name: '小游戏总览' }).waitFor()
  await page.getByRole('link', { name: '进入游戏' }).waitFor()
  await page.screenshot({ fullPage: true, path: '/private/tmp/games-platform-catalog.png' })

  await page.goto(`${baseURL}/games/uno`, { waitUntil: 'networkidle' })
  await page.getByRole('heading', { name: '登录后进入房间' }).waitFor()
  await page.getByRole('button', { name: '游客登录' }).click()
  await page.getByRole('heading', { name: /^UNO$/ }).waitFor()
  await page.getByRole('button', { name: '创建并进入' }).waitFor()
  await page.screenshot({ fullPage: true, path: '/private/tmp/games-platform-room-gate.png' })

  await page.getByRole('button', { name: '创建并进入' }).click()
  await page.getByRole('heading', { name: /房间\s*/ }).waitFor()
  const roomURL = page.url()
  const roomID = new URL(roomURL).searchParams.get('room')

  if (!roomID) {
    throw new Error('room id was not created')
  }

  const guestContext = await browser.newContext({ viewport: { width: 390, height: 900 } })
  const guestPage = await guestContext.newPage()
  await guestPage.goto(roomURL, { waitUntil: 'networkidle' })
  await guestPage.getByRole('heading', { name: '登录后进入房间' }).waitFor()
  await guestPage.getByRole('button', { name: '游客登录' }).click()
  await guestPage.getByRole('heading', { name: new RegExp(`房间\\s*${roomID}`) }).waitFor()

  await page.getByRole('button', { name: '添加 AI' }).click()
  await page.getByText(/AI 已加入房间|加入了房间/).waitFor()
  await page.getByRole('button', { name: '开始游戏' }).click()
  await page.getByRole('heading', { name: /^UNO$/ }).waitFor()
  await page.getByText(new RegExp(roomID)).waitFor()
  await page.getByText(/可出牌：/).waitFor()
  await expectNoPageScroll(page, 'desktop 1440x1000')
  await page.screenshot({ fullPage: true, path: '/private/tmp/games-platform-uno-desktop.png' })

  await guestPage.getByText(/可出牌：/).waitFor()
  await expectNoPageScroll(guestPage, 'mobile shared room 390x900')
  await guestPage.screenshot({ fullPage: true, path: '/private/tmp/games-platform-uno-mobile.png' })
  await guestContext.close()

  await page.setViewportSize({ width: 1366, height: 768 })
  await page.goto(roomURL, { waitUntil: 'networkidle' })
  await page.getByRole('heading', { name: /^UNO$/ }).waitFor()
  await expectNoPageScroll(page, 'desktop 1366x768')

  console.log('UI smoke verification passed')
}
finally {
  await browser.close()
}

async function expectNoPageScroll(page, label) {
  const metrics = await page.evaluate(() => ({
    height: window.innerHeight,
    scrollHeight: document.documentElement.scrollHeight,
  }))

  if (metrics.scrollHeight > metrics.height + 4) {
    throw new Error(`${label} scrolls: ${metrics.scrollHeight} > ${metrics.height}`)
  }
}
