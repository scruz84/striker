import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

// Walks every player-facing route at 360px and fails if a page
//   - scrolls horizontally,
//   - shows text smaller than MIN_FONT_PX, or
//   - has a tappable control shorter than MIN_TARGET_PX
// (the three checks proposed in the mobile UX epic, #268). The admin panel is out of scope.
//
// Needs the backend running on a seeded world with DEV_LOGIN_ENABLED=true - see playwright.config.ts.

const API = process.env.SMOKE_API_URL ?? 'http://127.0.0.1:1984'
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

      // Tappable controls shorter than the target size. Inline links inside running text are exempt
      // (their hit area is the line), as are controls MUI renders but hides from view.
      const controls = document.querySelectorAll('button, [role="button"], [role="tab"], [role="menuitem"], a[href], input:not([type="hidden"]), select')
      for (const el of controls) {
        if (!visible(el)) continue
        const style = getComputedStyle(el)
        if (el.tagName === 'A' && style.display === 'inline') continue
        if (el.hasAttribute('data-smoke-ignore') || el.closest('[data-smoke-ignore]')) continue
        // A text field's tap area is its whole box, not the <input> inside the padding.
        const hit = el.closest('.MuiInputBase-root') ?? el
        const height = hit.getBoundingClientRect().height
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

  test.beforeAll(async ({ playwright }) => {
    const request = await playwright.request.newContext()
    const ids = await setup(request)
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

    const report: string[] = []
    for (const route of routes) {
      await page.goto(route, { waitUntil: 'networkidle' })
      await page.waitForTimeout(500)
      const findings = await audit(page, MIN_FONT_PX, MIN_TARGET_PX)
      for (const f of findings) report.push(`${route}  [${f.kind}] ${f.detail}`)
    }
    expect(report, `\n${report.join('\n')}\n`).toEqual([])
  })
})
