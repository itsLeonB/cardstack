import AxeBuilder from "@axe-core/playwright"
import { test, expect } from "playwright/test"
import type { Page, Route } from "playwright/test"

// Sign-in and sign-up are Clerk's prebuilt components (ADR-0015). Guest checks
// here need no setup. Flows that sign in need a real Clerk session, which only
// ticket 09's Clerk Testing Tokens layer can create, so they are `fixme` until
// then. `stubApi` below simulated the removed /auth/me and /auth/login endpoints.
interface StubBody {
  data?: unknown
  detail?: string
  meta?: { total: number; page: number; limit: number }
}

function fulfillJson(route: Route, status: number, body: StubBody) {
  return {
    status,
    contentType: "application/json",
    headers: {
      "access-control-allow-origin": route.request().headers()["origin"] ?? "",
      "access-control-allow-credentials": "true",
    },
    body: JSON.stringify(body),
  }
}

async function stubApi(page: Page, { signedIn }: { signedIn: boolean }) {
  let session = signedIn
  await page.route("**/auth/me", (route) =>
    route.fulfill(
      session
        ? fulfillJson(route, 200, {
            data: { id: "u1", email: "ada@example.com" },
          })
        : fulfillJson(route, 401, { detail: "Unauthorized" })
    )
  )
  await page.route("**/auth/login", (route) => {
    // The page itself lives at /auth/login; only the API call is stubbed.
    if (route.request().resourceType() !== "fetch") return route.fallback()
    session = true
    return route.fulfill(fulfillJson(route, 200, { data: { csrfToken: "t" } }))
  })
  await page.route(/\/(collections|inventory)/, (route) => {
    if (route.request().resourceType() !== "fetch") return route.fallback()
    return route.fulfill(
      fulfillJson(route, 200, {
        data: [],
        meta: { total: 0, page: 1, limit: 24 },
      })
    )
  })
}

test.describe("Auth redirects", () => {
  test("sends a guest from a protected page to login, remembering the page and its query", async ({
    page,
  }) => {
    await page.goto("/collections?q=binder")

    // The route guard waits for Clerk to load before it decides.
    await expect(page).toHaveURL(
      /\/auth\/login\?redirect=%2Fcollections%3Fq%3Dbinder$/,
      { timeout: 15000 }
    )
  })

  test.fixme("returns to the protected page, query string included, after login", async ({
    page,
  }) => {
    await stubApi(page, { signedIn: false })
    await page.goto("/collections?q=binder")

    await expect(page).toHaveURL(
      /\/auth\/login\?redirect=%2Fcollections%3Fq%3Dbinder$/
    )
    await page.getByLabel("Email").fill("ada@example.com")
    await page.getByLabel("Password", { exact: true }).fill("correct horse")
    await page.getByRole("button", { name: "Log in" }).click()

    await expect(page).toHaveURL(/\/collections\?q=binder$/)
    await expect(
      page.getByRole("heading", { level: 1, name: "Collections" })
    ).toBeVisible()
  })

  test.fixme("ignores an external redirect target after login", async ({
    page,
  }) => {
    await stubApi(page, { signedIn: false })
    await page.goto("/auth/login?redirect=https://evil.example/")

    await page.getByLabel("Email").fill("ada@example.com")
    await page.getByLabel("Password", { exact: true }).fill("correct horse")
    await page.getByRole("button", { name: "Log in" }).click()

    await expect(page).toHaveURL(/localhost:\d+\/account$/)
  })

  for (const path of ["/auth/login", "/auth/register", "/auth/foo", "/auth"]) {
    test.fixme(`sends a signed-in user from ${path} to /`, async ({ page }) => {
      await stubApi(page, { signedIn: true })
      await page.goto(path)

      await expect(page).toHaveURL(/\/$/)
      await expect(
        page.getByRole("heading", { level: 1, name: "Welcome back" })
      ).toBeVisible()
    })
  }

  test("lets a guest reach login and register", async ({ page }) => {
    await page.goto("/auth/login")
    await expect(
      page.getByRole("heading", { level: 1, name: "Log in" })
    ).toBeVisible()

    await page.goto("/auth/register")
    await expect(
      page.getByRole("heading", { level: 1, name: "Create an account" })
    ).toBeVisible()
  })
})

test.describe("Auth pages", () => {
  for (const [path, heading] of [
    ["/auth/login", "Log in"],
    ["/auth/register", "Create an account"],
  ] as const) {
    test(`passes axe on ${path}`, async ({ page }) => {
      await page.goto(path)
      await expect(
        page.getByRole("heading", { level: 1, name: heading })
      ).toBeVisible()
      const results = await new AxeBuilder({ page }).analyze()
      expect(results.violations).toEqual([])
    })
  }

  for (const path of ["/auth/login", "/auth/register"]) {
    test(`${path} uses the minimal auth shell, not the site shell`, async ({
      page,
    }) => {
      await page.goto(path)
      await expect(page.getByRole("heading", { level: 1 })).toBeVisible()
      await expect(page.getByRole("main")).toHaveCount(1)
      await expect(page.getByRole("navigation")).toHaveCount(0)
      await expect(page.getByRole("contentinfo")).toHaveCount(0)
      await expect(
        page.getByRole("link", { name: "Cardstack" })
      ).toHaveAttribute("href", "/")
      await expect(page.getByRole("button", { name: "Theme" })).toBeVisible()
    })
  }

  for (const path of ["/auth/foo", "/auth"]) {
    test(`${path} shows not-found inside the auth shell`, async ({ page }) => {
      await page.goto(path)
      await expect(
        page.getByRole("heading", { level: 1, name: "Page not found" })
      ).toBeVisible()
      await expect(page.getByRole("main")).toHaveCount(1)
      await expect(page.getByRole("navigation")).toHaveCount(0)
      await expect(page.getByRole("contentinfo")).toHaveCount(0)
      await expect(
        page.getByRole("link", { name: "Cardstack" })
      ).toHaveAttribute("href", "/")
    })
  }

  test("a normal page keeps the site header and footer", async ({ page }) => {
    await page.goto("/catalog")
    await expect(page.getByRole("banner")).toBeVisible()
    await expect(page.getByRole("navigation", { name: "Main" })).toBeVisible()
    await expect(page.getByRole("contentinfo")).toBeVisible()
  })
})
