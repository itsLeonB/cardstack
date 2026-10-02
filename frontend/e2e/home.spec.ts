import AxeBuilder from "@axe-core/playwright"
import { test, expect } from "playwright/test"
import type { Page, Route } from "playwright/test"

// Credentialed cross-origin reads need the caller's exact origin echoed back,
// which keeps these stubs independent of the port the app is served on.
type StubBody = {
  data: unknown
  meta?: { total: number; page: number; limit: number }
}

function json(route: Route, body: StubBody) {
  return {
    status: 200,
    contentType: "application/json",
    headers: {
      "access-control-allow-origin": route.request().headers()["origin"] ?? "",
      "access-control-allow-credentials": "true",
    },
    body: JSON.stringify(body),
  }
}

// Stubs the session and the two reads the dashboard makes, so it can be
// asserted without a seeded user. Guests rely on the real /auth/me answering
// 401 (or being unreachable), as in app-shell.spec.ts.
async function signIn(page: Page, collections: { id: string; title: string }[], total: number) {
  await page.route("**/auth/me", (route) =>
    route.fulfill(json(route, { data: { id: "u1", email: "ada@example.com" } }))
  )
  await page.route("**/collections", (route) => {
    if (route.request().resourceType() !== "fetch") return route.fallback()
    return route.fulfill(json(route, { data: collections }))
  })
  await page.route("**/inventory/cards?*", (route) =>
    route.fulfill(json(route, { data: [], meta: { total, page: 1, limit: 1 } }))
  )
}

test.describe("Home: guest", () => {
  test("shows the landing with both actions and the three steps", async ({ page }) => {
    await page.goto("/")

    await expect(page.getByRole("heading", { level: 1 })).toBeVisible()
    await expect(
      page.getByRole("main").getByText("Track every card you own, across every binder.")
    ).toBeVisible()
    await expect(page.getByRole("heading", { name: "How it works" })).toBeVisible()
    for (const step of [
      "Browse the Catalog",
      "Add Cards to Collections",
      "See your Master Inventory",
    ]) {
      await expect(page.getByRole("heading", { level: 3, name: step })).toBeVisible()
    }
    await expect(page.getByRole("heading", { name: "Welcome back" })).toHaveCount(0)

    const main = page.getByRole("main")
    await main.getByRole("link", { name: "Browse the catalog" }).click()
    await expect(page).toHaveURL(/\/catalog$/)
  })

  test("Create account leads to register", async ({ page }) => {
    await page.goto("/")
    await page.getByRole("main").getByRole("link", { name: "Create account" }).first().click()
    await expect(page).toHaveURL(/\/register$/)
  })

  test("passes axe", async ({ page }) => {
    await page.goto("/")
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible()
    const results = await new AxeBuilder({ page }).analyze()
    expect(results.violations).toEqual([])
  })
})

test.describe("Home: signed in", () => {
  test("shows the dashboard, not the landing", async ({ page }) => {
    await signIn(page, [{ id: "c1", title: "Trade binder" }], 42)
    await page.goto("/")

    await expect(page.getByRole("heading", { level: 1, name: "Welcome back" })).toBeVisible()
    await expect(page.getByRole("link", { name: "Trade binder" })).toBeVisible()
    await expect(page.getByRole("link", { name: "New collection" })).toBeVisible()
    await expect(page.getByText("distinct cards")).toBeVisible()
    await expect(page.getByText("42", { exact: true })).toBeVisible()
    await expect(page.getByRole("heading", { name: "How it works" })).toHaveCount(0)
  })

  test("guides a user with no Collections", async ({ page }) => {
    await signIn(page, [], 0)
    await page.goto("/")

    await expect(page.getByText(/Start by creating a Collection/)).toBeVisible()
  })

  test("passes axe", async ({ page }) => {
    await signIn(page, [{ id: "c1", title: "Trade binder" }], 42)
    await page.goto("/")
    await expect(page.getByRole("heading", { name: "Welcome back" })).toBeVisible()
    await expect(page.getByText("distinct cards")).toBeVisible()
    const results = await new AxeBuilder({ page }).analyze()
    expect(results.violations).toEqual([])
  })

  test("account page is settings-style without navigation links and offers Log out", async ({ page }) => {
    await signIn(page, [], 0)
    await page.goto("/")
    // The dev server can hydrate after the first click lands; retry until the menu opens.
    await expect(async () => {
      await page.getByRole("button", { name: "User menu" }).click()
      await expect(page.getByRole("menuitem", { name: "Account" })).toBeVisible({ timeout: 1000 })
    }).toPass()
    await page.getByRole("menuitem", { name: "Account" }).click()

    const main = page.getByRole("main")
    await expect(main.getByRole("heading", { level: 1, name: "Account" })).toBeVisible()
    await expect(main).toContainText("ada@example.com")
    await expect(main.getByRole("link")).toHaveCount(0)
    await expect(main.getByRole("button", { name: "Log out" })).toBeVisible()
  })
})

test.describe("Home: mobile", () => {
  test.use({ viewport: { width: 375, height: 700 } })

  test("landing has no horizontal scroll at phone width", async ({ page }) => {
    await page.goto("/")
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible()
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - window.innerWidth
    )
    expect(overflow).toBeLessThanOrEqual(0)
  })

  test("dashboard has no horizontal scroll at phone width", async ({ page }) => {
    await signIn(page, [{ id: "c1", title: "Trade binder" }], 42)
    await page.goto("/")
    await expect(page.getByRole("heading", { name: "Welcome back" })).toBeVisible()
    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth - window.innerWidth
    )
    expect(overflow).toBeLessThanOrEqual(0)
  })
})
