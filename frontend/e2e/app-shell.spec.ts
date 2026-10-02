import AxeBuilder from "@axe-core/playwright"
import { test, expect } from "playwright/test"
import type { Page } from "playwright/test"

// Shell-level checks. Guest tests rely on the real /auth/me answering 401 (or
// being unreachable), which both read as "guest". The signed-in session is
// stubbed at /auth/me so the header can be asserted without seeded users.

async function signIn(page: Page) {
  // Destination pages (including the dashboard at /) crash without a backend or
  // on a body missing `meta`; empty paginated lists keep the shell up.
  await page.route(/\/(collections|inventory)/, (route) => {
    if (route.request().resourceType() !== "fetch") return route.fallback()
    return route.fulfill({
      status: 200,
      contentType: "application/json",
      headers: {
        "access-control-allow-origin": route.request().headers().origin ?? "",
        "access-control-allow-credentials": "true",
      },
      body: JSON.stringify({ data: [], meta: { total: 0, page: 1, limit: 24 } }),
    })
  })
  await page.route("**/auth/me", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      headers: {
        "access-control-allow-origin": route.request().headers().origin ?? "",
        "access-control-allow-credentials": "true",
      },
      body: JSON.stringify({ data: { id: "u1", email: "ada@example.com" } }),
    })
  )
}

// The dev server can hydrate after the first click lands, which would be a
// no-op; retry until the menu is actually open.
async function openMenu(page: Page, name: string, item: string) {
  await expect(async () => {
    await page.getByRole("button", { name }).click()
    await expect(page.getByRole(item === "Account" ? "menuitem" : "menuitemradio", { name: item })).toBeVisible({ timeout: 1000 })
  }).toPass()
}

test.describe("App shell: guest", () => {
  test("shows guest navigation and footer", async ({ page }) => {
    await page.goto("/")

    const header = page.getByRole("banner")
    await expect(header.getByRole("link", { name: "Catalog" })).toBeVisible()
    await expect(header.getByRole("link", { name: "Log in" })).toBeVisible()
    await expect(header.getByRole("link", { name: "Register" })).toBeVisible()
    await expect(header.getByRole("link", { name: "Collections" })).toHaveCount(0)
    await expect(page.getByRole("main")).toHaveCount(1)
    await expect(page.getByRole("contentinfo")).toContainText("personal MVP")
    await expect(page).toHaveTitle("Track every Pokémon card you own · Cardstack")
  })

  test("skip link is the first focusable element", async ({ page }) => {
    await page.goto("/")
    await page.keyboard.press("Tab")
    await expect(page.getByRole("link", { name: "Skip to content" })).toBeFocused()
  })

  test("Log in leads to the auth page, which drops the site nav", async ({ page }) => {
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

test.describe("App shell: signed in", () => {
  test("shows signed-in navigation and the user menu", async ({ page }) => {
    await signIn(page)
    await page.goto("/")

    const header = page.getByRole("banner")
    await expect(header.getByRole("link", { name: "Collections" })).toBeVisible()
    await expect(header.getByRole("link", { name: "Master Inventory" })).toBeVisible()
    await expect(header.getByRole("link", { name: "Log in" })).toHaveCount(0)
    await expect(header.getByRole("link", { name: "Inventory", exact: true })).toHaveCount(0)

    await openMenu(page, "User menu", "Account")
    await expect(page.getByRole("menuitem", { name: "Log out" })).toBeVisible()
    await expect(page.getByRole("menu")).toContainText("ada@example.com")
  })

  test("passes axe on the signed-in shell", async ({ page }) => {
    await signIn(page)
    await page.goto("/")
    await expect(page.getByRole("link", { name: "Collections" }).first()).toBeVisible()
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
    await expect(header.getByRole("link", { name: "Collections" })).toHaveAttribute(
      "aria-current",
      "page"
    )

    await openMenu(page, "User menu", "Account")
    await page.getByRole("menuitem", { name: "Account" }).click()
    await expect(page).toHaveURL(/\/account$/)

    await header.getByRole("link", { name: "Master Inventory" }).click()
    await expect(page).toHaveURL(/\/inventory/)
  })
})

test.describe("App shell: theme", () => {
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

  test("collapses navigation into a menu that opens and closes", async ({ page }) => {
    await page.goto("/")
    const header = page.getByRole("banner")
    const toggle = header.getByRole("button", { name: "Menu" })

    await expect(header.getByRole("link", { name: "Log in" })).toBeHidden()
    await expect(async () => {
      await toggle.click()
      await expect(toggle).toHaveAttribute("aria-expanded", "true", { timeout: 1000 })
    }).toPass()
    await expect(header.getByRole("link", { name: "Log in" })).toBeVisible()

    await page.keyboard.press("Escape")
    await expect(toggle).toHaveAttribute("aria-expanded", "false")
    await expect(toggle).toBeFocused()
    await expect(header.getByRole("link", { name: "Log in" })).toBeHidden()
  })

  for (const path of ["/", "/auth/login", "/auth/register"]) {
    test(`no horizontal scroll on ${path}`, async ({ page }) => {
      await page.goto(path)
      await expect(page.getByRole("heading", { level: 1 })).toBeVisible()
      const overflow = await page.evaluate(
        () => document.documentElement.scrollWidth - window.innerWidth
      )
      expect(overflow).toBeLessThanOrEqual(0)
    })
  }
})
