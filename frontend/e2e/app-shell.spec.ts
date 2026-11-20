import AxeBuilder from "@axe-core/playwright"
import { test, expect } from "playwright/test"
import type { Page } from "playwright/test"

import { stubEmptyLists } from "./support/api-stubs"
import {
  expectShowsTestUserEmail,
  signInAsTestUser,
  SIGNED_IN_TAG,
  useSignedInSuite,
  waitForClerkReady,
} from "./support/clerk-auth"

// Shell-level checks. Guest tests need no setup: Clerk loads, finds no session
// and the shell reads as "guest". Signed-in tests sign in through Clerk's real
// development instance (see support/clerk-auth.ts) and skip without its
// credentials; the Collection and Inventory lists stay stubbed so the
// destination pages render whatever the backend holds.

async function signIn(page: Page) {
  await stubEmptyLists(page)
  await signInAsTestUser(page)
}

// The dev server can hydrate after the first click lands, which would be a
// no-op; retry until the menu is actually open.
async function openMenu(page: Page, name: string, item: string) {
  await expect(async () => {
    await page.getByRole("button", { name }).click()
    await expect(
      page.getByRole(item === "Account" ? "menuitem" : "menuitemradio", {
        name: item,
      })
    ).toBeVisible({ timeout: 1000 })
  }).toPass()
}

test.describe("App shell: guest", () => {
  test("shows guest navigation and footer", async ({ page }) => {
    await page.goto("/")
    await waitForClerkReady(page)

    const header = page.getByRole("banner")
    await expect(header.getByRole("link", { name: "Catalog" })).toBeVisible()
    await expect(header.getByRole("link", { name: "Log in" })).toBeVisible()
    await expect(header.getByRole("link", { name: "Register" })).toBeVisible()
    await expect(header.getByRole("link", { name: "Collections" })).toHaveCount(
      0
    )
    await expect(page.getByRole("main")).toHaveCount(1)
    await expect(page.getByRole("contentinfo")).toContainText("personal MVP")
    await expect(page).toHaveTitle(
      "Track every Pokémon card you own · Cardstack"
    )
  })

  test("skip link is the first focusable element", async ({ page }) => {
    await page.goto("/")
    await page.keyboard.press("Tab")
    await expect(
      page.getByRole("link", { name: "Skip to content" })
    ).toBeFocused()
  })

  // Startup contract (follow-up to the PR #95 "unstyled app on first load"
  // report): the document head itself carries the stylesheet link and the
  // theme bootstrap script, so the stylesheet is render-blocking and the
  // shell cannot paint before it applies, whatever its delivery speed. This
  // test pins that delivery contract, not paint timing itself; the paint
  // measurement lives in the ticket 04 diagnosis. Gating only the stylesheet
  // request simulates a slow first visit; no production code delays anything.
  test("critical CSS and theme bootstrap ship in the document", async ({
    page,
  }) => {
    await page.route(/\.css($|\?)/, (route) => {
      setTimeout(() => route.continue(), 1500)
    })
    await page.goto("/", { waitUntil: "domcontentloaded" })

    const startup = await page.evaluate(() => {
      const link = document.querySelector("head link[rel=stylesheet]")
      return {
        stylesheetInHead: link !== null,
        stylesheetApplied:
          link instanceof HTMLLinkElement && link.sheet !== null,
        themeBootstrapped: ["light", "dark"].some((theme) =>
          document.documentElement.classList.contains(theme)
        ),
        bodyStyled:
          getComputedStyle(document.body).backgroundColor !==
          "rgba(0, 0, 0, 0)",
      }
    })

    expect(startup).toEqual({
      stylesheetInHead: true,
      stylesheetApplied: true,
      themeBootstrapped: true,
      bodyStyled: true,
    })
  })

  test("Log in leads to the auth page, which drops the site nav", async ({
    page,
  }) => {
    await page.goto("/")
    const header = page.getByRole("banner")

    // Auth pages swap in their own shell, so the site nav is gone there.
    await header.getByRole("link", { name: "Log in" }).click()
    await expect(page).toHaveURL(/\/auth\/login$/)
    await expect(page.getByRole("navigation")).toHaveCount(0)
  })

  test("passes axe on the shell at /", async ({ page }) => {
    await page.goto("/")
    await expect(page.getByRole("banner")).toBeVisible()
    // Headings belong to page content, not the shell, so the h1 rule is out of
    // scope here. The auth shell has its own axe checks in auth.spec.ts.
    const results = await new AxeBuilder({ page })
      .disableRules(["page-has-heading-one"])
      .analyze()
    expect(results.violations).toEqual([])
  })
})

test.describe("App shell: signed in", { tag: SIGNED_IN_TAG }, () => {
  useSignedInSuite()

  test("shows signed-in navigation and the user menu", async ({ page }) => {
    await signIn(page)
    await page.goto("/")

    const header = page.getByRole("banner")
    await expect(
      header.getByRole("link", { name: "Collections" })
    ).toBeVisible()
    await expect(
      header.getByRole("link", { name: "Master Inventory" })
    ).toBeVisible()
    await expect(header.getByRole("link", { name: "Log in" })).toHaveCount(0)
    await expect(
      header.getByRole("link", { name: "Inventory", exact: true })
    ).toHaveCount(0)

    await openMenu(page, "User menu", "Account")
    await expect(page.getByRole("menuitem", { name: "Log out" })).toBeVisible()
    await expectShowsTestUserEmail(page.getByRole("menu"))
    await page.keyboard.press("Escape")

    await page.setViewportSize({ width: 768, height: 900 })
    await page
      .locator('header button[aria-label="User menu"] span')
      .evaluate((name) => {
        name.textContent =
          "A very long display name that must not hide navigation"
      })
    const nav = header.getByRole("navigation", { name: "Main" })
    for (const destination of [
      "Catalog",
      "Collections",
      "Wishlist",
      "Trades",
      "Master Inventory",
    ]) {
      await expect(nav.getByRole("link", { name: destination })).toBeVisible()
    }
    const headerOverflow = await header
      .locator(":scope > div")
      .evaluate((inner) => inner.scrollWidth - inner.clientWidth)
    expect(headerOverflow).toBeLessThanOrEqual(0)
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth - innerWidth
      )
    ).toBeLessThanOrEqual(0)

    await page.addStyleTag({ content: "html { font-size: 200% !important; }" })
    await expect(header.locator("a").first()).toBeVisible()
    for (const destination of [
      "Catalog",
      "Collections",
      "Wishlist",
      "Trades",
      "Master Inventory",
    ]) {
      await expect(nav.getByRole("link", { name: destination })).toBeVisible()
    }
    await expect(
      header.getByRole("button", { name: "User menu" })
    ).toBeVisible()
    const enlargedHeaderOverflow = await header
      .locator(":scope > div")
      .evaluate((inner) => inner.scrollWidth - inner.clientWidth)
    expect(enlargedHeaderOverflow).toBeLessThanOrEqual(0)
  })

  test("passes axe on the signed-in shell", async ({ page }) => {
    await signIn(page)
    await page.goto("/")
    await expect(
      page.getByRole("link", { name: "Collections" }).first()
    ).toBeVisible()
    const results = await new AxeBuilder({ page })
      .disableRules(["page-has-heading-one"])
      .analyze()
    expect(results.violations).toEqual([])
  })

  test("keeps Catalog current on a nested catalog route", async ({ page }) => {
    await signIn(page)
    await page.goto("/catalog/search")
    await expect(
      page.getByRole("banner").getByRole("link", { name: "Catalog" })
    ).toHaveAttribute("aria-current", "page")
  })

  test("navigates between main areas", async ({ page }) => {
    await signIn(page)
    await page.goto("/")
    const header = page.getByRole("banner")

    await header.getByRole("link", { name: "Collections" }).click()
    await expect(page).toHaveURL(/\/collections$/)
    await expect(
      header.getByRole("link", { name: "Collections" })
    ).toHaveAttribute("aria-current", "page")

    await openMenu(page, "User menu", "Account")
    await page.getByRole("menuitem", { name: "Account" }).click()
    await expect(page).toHaveURL(/\/account$/)

    await header.getByRole("link", { name: "Master Inventory" }).click()
    await expect(page).toHaveURL(/\/inventory/)

    await page.setViewportSize({ width: 375, height: 700 })
    const nav = page.getByRole("navigation", { name: "Bottom navigation" })
    const more = nav.getByRole("button", { name: "More" })
    await more.click()
    await expect(page.getByRole("menuitem", { name: "Account" })).toBeVisible()
    await page.keyboard.press("Escape")
    await expect(page.getByRole("menuitem", { name: "Account" })).toHaveCount(0)
    await expect(more).toBeFocused()

    await more.click()
    await page.getByRole("menuitem", { name: "Account" }).click()
    await expect(page).toHaveURL(/\/account$/)
    await expect(more).toHaveAttribute("aria-current", "page")

    await more.click()
    await page.getByRole("menuitem", { name: "Master Inventory" }).click()
    await expect(page).toHaveURL(/\/inventory/)
    await more.click()
    await expect(page.getByRole("menuitem", { name: "Account" })).toBeVisible()
    await page.setViewportSize({ width: 1024, height: 768 })
    await expect(page.getByRole("menuitem", { name: "Account" })).toHaveCount(0)

    await page.setViewportSize({ width: 375, height: 700 })
    await nav.getByRole("link", { name: "Trades" }).click()
    await expect(page).toHaveURL(/\/trades$/)
    await expect(
      page.getByRole("heading", { name: "Your Trades" })
    ).toBeVisible()
  })
})

test.describe("App shell: theme", () => {
  test("keeps content clear of simulated horizontal safe areas", async ({
    page,
  }) => {
    await page.goto("/catalog")
    const pageContainer = page
      .locator('#main-content div[class*="py-6"]')
      .first()
    await expect(pageContainer).toBeVisible()

    for (const [width, contentPadding] of [
      [375, "16px"],
      [768, "24px"],
    ] as const) {
      await page.setViewportSize({ width, height: 800 })
      const zeroInsetPadding = await page.evaluate(() => {
        const wrapper = document.querySelector<HTMLElement>(
          '#main-content div[class*="py-6"]'
        )
        if (!wrapper) throw new Error("PageContainer is missing")
        return getComputedStyle(wrapper).paddingLeft
      })
      expect(zeroInsetPadding).toBe(contentPadding)
    }

    for (const width of [375, 768]) {
      await page.setViewportSize({ width, height: 800 })
      const result = await page.evaluate(() => {
        const changed: {
          rule: CSSStyleRule
          property: string
          value: string
          priority: string
        }[] = []
        function replaceInsets(rules: CSSRuleList) {
          for (const rule of Array.from(rules)) {
            if (rule instanceof CSSStyleRule) {
              for (const property of Array.from(rule.style)) {
                const value = rule.style.getPropertyValue(property)
                const updated = value
                  .replace(/env\(safe-area-inset-left,\s*0px\)/g, "32px")
                  .replace(/env\(safe-area-inset-right,\s*0px\)/g, "28px")
                if (updated !== value) {
                  changed.push({
                    rule,
                    property,
                    value,
                    priority: rule.style.getPropertyPriority(property),
                  })
                  rule.style.setProperty(property, updated)
                }
              }
            } else if (rule instanceof CSSGroupingRule) {
              replaceInsets(rule.cssRules)
            }
          }
        }
        try {
          for (const sheet of Array.from(document.styleSheets)) {
            replaceInsets(sheet.cssRules)
          }
          const wrapper = document.querySelector<HTMLElement>(
            '#main-content div[class*="py-6"]'
          )
          const header = document.querySelector<HTMLElement>("header > div")
          const footer = document.querySelector<HTMLElement>("footer > div")
          if (!wrapper || !header || !footer)
            throw new Error("A page shell wrapper is missing")
          return {
            changedRules: changed.length,
            padding: [wrapper, header, footer].map((element) => {
              const style = getComputedStyle(element)
              return [style.paddingLeft, style.paddingRight]
            }),
            overflow: document.documentElement.scrollWidth - innerWidth,
          }
        } finally {
          for (const { rule, property, value, priority } of changed) {
            rule.style.setProperty(property, value, priority)
          }
        }
      })
      expect(result.changedRules).toBeGreaterThanOrEqual(6)
      expect(result.padding).toEqual([
        ["32px", "28px"],
        ["32px", "28px"],
        ["32px", "28px"],
      ])
      expect(result.overflow).toBeLessThanOrEqual(0)
    }
  })

  test("persists the chosen theme across reloads", async ({ page }) => {
    await page.goto("/")
    await openMenu(page, "Theme", "Dark")
    await page.getByRole("menuitemradio", { name: "Dark" }).click()
    await expect(page.locator("html")).toHaveClass(/dark/)

    await page.reload()
    await expect(page.locator("html")).toHaveClass(/dark/)

    await openMenu(page, "Theme", "Light")
    await page.getByRole("menuitemradio", { name: "Light" }).click()
    await page.reload()
    await expect(page.locator("html")).not.toHaveClass(/dark/)
  })

  test("applies the stored theme before first paint", async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem("theme", "dark"))
    await page.goto("/", { waitUntil: "commit" })
    await expect(page.locator("html")).toHaveClass(/dark/)
  })
})

test.describe("App shell: mobile", () => {
  test.use({ viewport: { width: 375, height: 700 } })

  test("keeps guest destinations visible and content clear at mobile sizes", async ({
    page,
  }) => {
    await page.goto("/")
    await waitForClerkReady(page)
    const nav = page.getByRole("navigation", { name: "Bottom navigation" })

    await expect(nav.getByRole("link", { name: "Catalog" })).toBeVisible()
    await expect(nav.getByRole("link", { name: "Log in" })).toBeVisible()
    await expect(nav.getByRole("link", { name: "Register" })).toBeVisible()
    await expect(page.getByRole("button", { name: "Menu" })).toHaveCount(0)

    await openMenu(page, "Theme", "Light")
    await page.getByRole("menuitemradio", { name: "Light" }).click()
    await page.reload()
    await waitForClerkReady(page)
    await expect(page.locator("html")).not.toHaveClass(/dark/)
    expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([])

    await openMenu(page, "Theme", "Dark")
    await page.getByRole("menuitemradio", { name: "Dark" }).click()
    await page.reload()
    await waitForClerkReady(page)
    await expect(page.locator("html")).toHaveClass(/dark/)
    expect((await new AxeBuilder({ page }).analyze()).violations).toEqual([])

    for (const viewport of [
      { width: 320, height: 640 },
      { width: 375, height: 700 },
      { width: 700, height: 375 },
    ]) {
      await page.setViewportSize(viewport)
      await expect(nav).toBeVisible()
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - window.innerWidth
      )
      expect(overflow).toBeLessThanOrEqual(0)
    }

    await page.evaluate(() => {
      document.documentElement.style.fontSize = "200%"
    })
    const zoomOverflow = await page.evaluate(
      () => document.documentElement.scrollWidth - window.innerWidth
    )
    expect(zoomOverflow).toBeLessThanOrEqual(0)

    const lastAction = page.getByRole("link", { name: "Create account" }).last()
    await lastAction.scrollIntoViewIfNeeded()
    const actionBox = await lastAction.boundingBox()
    const navBox = await nav.boundingBox()
    expect(actionBox).not.toBeNull()
    expect(navBox).not.toBeNull()
    expect(actionBox!.y + actionBox!.height).toBeLessThanOrEqual(navBox!.y)
  })

  for (const path of ["/", "/auth/login", "/auth/register"]) {
    test(`no horizontal scroll on ${path}`, async ({ page }) => {
      await page.goto(path)
      await waitForClerkReady(page)
      await expect(page.getByRole("heading", { level: 1 })).toBeVisible({
        timeout: 15_000,
      })
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - window.innerWidth
      )
      expect(overflow).toBeLessThanOrEqual(0)
    })
  }
})
