import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

// Walks every player-facing route at 360px and fails if a page
//   - scrolls horizontally,
//   - shows text smaller than MIN_FONT_PX, or
//   - has a tappable control shorter than MIN_TARGET_PX
// (the three checks proposed in the mobile UX epic, #268). The admin panel is out of scope.
//
// Needs the backend running on a seeded world with DEV_LOGIN_ENABLED=true - see playwright.config.ts.

const API = process.env.SMOKE_API_URL ?? 'http://localhost:1984'
const MIN_FONT_PX = 11
const MIN_TARGET_PX = 44

interface Me {
  teamId: string
  teamLeague: string
  pendingCountrySelection?: boolean
}

// Signs in as a dev user (creating its team in "es" on first use) and resolves the ids that
// parameterised routes need. A fresh world has no played matches, so /match/:id shows the
// scheduled view.
async function setup(request: APIRequestContext) {
  const login = await request.post(`${API}/v1/login`, { data: { devEmail: 'user1@dev.local' } })
  expect(login.ok(), 'dev login (is DEV_LOGIN_ENABLED=true?)').toBeTruthy()
  let me = (await (await request.get(`${API}/v1/auth/me`)).json()) as Me
  if (me.pendingCountrySelection) {
    const picked = await request.post(`${API}/v1/auth/onboarding/country`, { data: { country: 'es' } })
    expect(picked.ok(), 'onboarding country').toBeTruthy()
    me = (await (await request.get(`${API}/v1/auth/me`)).json()) as Me
  }
  const team = (await (await request.get(`${API}/v1/user/team`)).json()) as { players: { id: number }[] }
  const matches = (await (await request.get(`${API}/v1/teams/team/${me.teamId}/matches`)).json()) as { kind: string; matchId: number }[]
  const league = matches.find((m) => m.kind === 'league')
  return { teamId: me.teamId, divisionId: me.teamLeague, playerId: team.players[0].id, matchId: league?.matchId }
}

interface Finding {
  kind: 'overflow' | 'font' | 'target'
  detail: string
}

// Runs in the page. Returns what breaks the three rules.
async function audit(page: Page, minFont: number, minTarget: number): Promise<Finding[]> {
  return page.evaluate(
    ({ minFont, minTarget }) => {
      const findings: { kind: 'overflow' | 'font' | 'target'; detail: string }[] = []
      const describe = (el: Element) => {
        const text = (el.textContent ?? '').trim().replace(/\s+/g, ' ').slice(0, 40)
        const tag = el.tagName.toLowerCase()
        const label = el.getAttribute('aria-label')
        const type = el.getAttribute('type')
        return `<${tag}${type ? ` type=${type}` : ''}${label ? ` aria-label="${label}"` : ''}> "${text}"`
      }
      const visible = (el: Element) => {
        const r = el.getBoundingClientRect()
        if (r.width === 0 || r.height === 0) return false
        const style = getComputedStyle(el)
        return style.visibility !== 'hidden' && style.display !== 'none' && Number(style.opacity) > 0
      }

      const root = document.documentElement
      if (root.scrollWidth > window.innerWidth + 1) {
        findings.push({ kind: 'overflow', detail: `scrollWidth ${root.scrollWidth} > ${window.innerWidth}` })
      }

      // Text smaller than the floor: every element that directly owns a non-empty text node.
      const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT)
      const seen = new Set<Element>()
      for (let node = walker.nextNode(); node; node = walker.nextNode()) {
        const el = node.parentElement
        if (!el || seen.has(el) || !(node.textContent ?? '').trim() || !visible(el)) continue
        seen.add(el)
        const size = parseFloat(getComputedStyle(el).fontSize)
        if (size < minFont) findings.push({ kind: 'font', detail: `${size}px ${describe(el)}` })
      }

      // Tappable controls whose tap area is shorter than the target size. Inline links inside
      // running text are exempt (their hit area is the line), as are controls MUI hides from view.
      const controls = document.querySelectorAll('button, [role="button"], [role="tab"], [role="menuitem"], a[href], input:not([type="hidden"]), select')
      for (const el of controls) {
        if (!visible(el)) continue
        const style = getComputedStyle(el)
        if (el.tagName === 'A' && style.display === 'inline') continue
        if (el.hasAttribute('data-smoke-ignore') || el.closest('[data-smoke-ignore]')) continue
        // A text field's or slider's tap area is its whole box, not the <input> inside it.
        const hit = el.closest('.MuiInputBase-root, .MuiSlider-root') ?? el
        let height = hit.getBoundingClientRect().height
        // Compact controls grow an invisible ::after hit area on touch screens (see theme.ts).
        const after = getComputedStyle(hit, '::after')
        if (after.content !== 'none' && after.position === 'absolute') height = Math.max(height, parseFloat(after.height) || 0)
        if (height < minTarget - 0.5) findings.push({ kind: 'target', detail: `${Math.round(height)}px ${describe(el)}` })
      }
      return findings
    },
    { minFont, minTarget },
  )
}

const STATIC_ROUTES = [
  '/',
  '/dashboard',
  '/dashboard/lineup',
  '/dashboard/finances',
  '/transfers',
  '/cup',
  '/international-cup',
  '/friendlies',
  '/national-teams',
  '/groups',
  '/help',
  '/leagues',
  '/players/evolutions',
  '/search',
]

test.describe('mobile smoke (360px)', () => {
  let routes: string[] = []
  let divisionId = ''

  test.beforeAll(async ({ playwright }) => {
    const request = await playwright.request.newContext()
    const ids = await setup(request)
    divisionId = ids.divisionId
    routes = [
      ...STATIC_ROUTES,
      `/division/${ids.divisionId}`,
      `/team/${ids.teamId}`,
      `/team/${ids.teamId}/matches`,
      `/player/${ids.playerId}`,
      ...(ids.matchId ? [`/match/${ids.matchId}`] : []),
    ]
    await request.dispose()
  })

  test('every route fits the phone screen', async ({ page, context }) => {
    // The browser needs its own session: sign in through the same API from the page's context.
    const login = await context.request.post(`${API}/v1/login`, { data: { devEmail: 'user1@dev.local' } })
    expect(login.ok()).toBeTruthy()

    // Every route below renders a public fallback when signed out, so a broken session would let
    // the audit pass on pages nobody actually uses: make sure the signed-in shell (bottom bar) is up.
    await page.goto('/dashboard', { waitUntil: 'networkidle' })
    await expect(page.locator('.MuiBottomNavigation-root'), 'signed in (check CORS origin and cookie host)').toBeVisible()

    const report: string[] = []
    for (const route of routes) {
      await page.goto(route, { waitUntil: 'networkidle' })
      await page.waitForTimeout(500)
      const findings = await audit(page, MIN_FONT_PX, MIN_TARGET_PX)
      for (const f of findings) report.push(`${route}  [${f.kind}] ${f.detail}`)
    }
    expect(report, `\n${report.join('\n')}\n`).toEqual([])
  })

  // The league page's four tabs and its matchday navigator: each state the audit above never
  // reaches by just loading the route.
  test('league page tabs and matchday navigator fit the phone screen', async ({ page, context }) => {
    const login = await context.request.post(`${API}/v1/login`, { data: { devEmail: 'user1@dev.local' } })
    expect(login.ok()).toBeTruthy()
    await page.goto(`/division/${divisionId}`, { waitUntil: 'networkidle' })

    const report: string[] = []
    const check = async (step: string) => {
      await page.waitForTimeout(300)
      for (const f of await audit(page, MIN_FONT_PX, MIN_TARGET_PX)) report.push(`${step}  [${f.kind}] ${f.detail}`)
    }

    // All four labels must be fully visible, not clipped by the tab strip.
    const tabs = page.getByRole('tab')
    await expect(tabs).toHaveCount(4)
    const strip = await page.locator('.MuiTabs-root').boundingBox()
    for (const [index, tab] of (await tabs.all()).entries()) {
      const box = await tab.boundingBox()
      expect(box && strip && box.x >= strip.x - 1 && box.x + box.width <= strip.x + strip.width + 1, `tab ${index} inside the strip`).toBeTruthy()
      const clipped = await tab.evaluate((el) => el.scrollWidth > el.clientWidth + 1)
      expect(clipped, `tab ${index} label clipped`).toBeFalsy()
    }

    for (const [index, tab] of (await tabs.all()).entries()) {
      await tab.click()
      await check(`tab ${index}`)
    }

    await tabs.nth(1).click()
    for (const name of [/siguiente|next/i, /anterior|previous/i]) {
      const button = page.getByRole('button', { name })
      if (await button.isEnabled()) await button.click()
      await check(`navigator ${name}`)
    }
    expect(report, `\n${report.join('\n')}\n`).toEqual([])
  })
})
