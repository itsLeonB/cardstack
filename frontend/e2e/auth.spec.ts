import AxeBuilder from "@axe-core/playwright"
import { test, expect } from "playwright/test"
import type { Page, Route } from "playwright/test"

// Stubs the API so the redirect flows run without a seeded user. The session
// stays a guest (401 at /auth/me) until the login stub is hit, as in the real
// backend, so the post-login redirect is exercised against a live session flip.
interface StubBody {
  data?: unknown
  detail?: string
  meta?: { total: number; page: number; limit: number }
}

function json(route: Route, status: number, body: StubBody) {
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
        ? json(route, 200, { data: { id: "u1", email: "ada@example.com" } })
        : json(route, 401, { detail: "Unauthorized" })
    )
  )
  await page.route("**/auth/login", (route) => {
    session = true
    return route.fulfill(json(route, 200, { data: { csrfToken: "t" } }))
  })
  await page.route(/\/(collections|inventory)/, (route) => {
    if (route.request().resourceType() !== "fetch") return route.fallback()
    return route.fulfill(json(route, 200, { data: [], meta: { total: 0, page: 1, limit: 24 } }))
  })
}

test.describe("Auth redirects", () => {
  test("returns to the protected page, query string included, after login", async ({ page }) => {
    await stubApi(page, { signedIn: false })
    await page.goto("/collections?q=binder")

    await expect(page).toHaveURL(/\/login\?redirect=%2Fcollections%3Fq%3Dbinder$/)
    await page.getByLabel("Email").fill("ada@example.com")
    await page.getByLabel("Password", { exact: true }).fill("correct horse")
    await page.getByRole("button", { name: "Log in" }).click()

    await expect(page).toHaveURL(/\/collections\?q=binder$/)
    await expect(page.getByRole("heading", { level: 1, name: "Collections" })).toBeVisible()
  })

  test("ignores an external redirect target after login", async ({ page }) => {
    await stubApi(page, { signedIn: false })
    await page.goto("/login?redirect=https://evil.example/")

    await page.getByLabel("Email").fill("ada@example.com")
    await page.getByLabel("Password", { exact: true }).fill("correct horse")
    await page.getByRole("button", { name: "Log in" }).click()

    await expect(page).toHaveURL(/localhost:\d+\/account$/)
  })

  for (const path of ["/login", "/register"]) {
    test(`sends a signed-in user from ${path} to /`, async ({ page }) => {
      await stubApi(page, { signedIn: true })
      await page.goto(path)

      await expect(page).toHaveURL(/\/$/)
      await expect(page.getByRole("heading", { level: 1, name: "Welcome back" })).toBeVisible()
    })
  }

  test("lets a guest reach login and register and switch between them", async ({ page }) => {
    await stubApi(page, { signedIn: false })
    await page.goto("/login")
    await expect(page.getByRole("heading", { level: 1, name: "Log in" })).toBeVisible()

    await page.getByRole("main").getByRole("link", { name: "Register" }).click()
    await expect(page).toHaveURL(/\/register$/)
    await expect(page.getByRole("heading", { level: 1, name: "Create an account" })).toBeVisible()
  })
})

test.describe("Auth forms", () => {
  test.beforeEach(async ({ page }) => {
    await stubApi(page, { signedIn: false })
  })

  for (const [path, heading] of [
    ["/login", "Log in"],
    ["/register", "Create an account"],
  ] as const) {
    test(`passes axe on ${path}`, async ({ page }) => {
      await page.goto(path)
      await expect(page.getByRole("heading", { level: 1, name: heading })).toBeVisible()
      const results = await new AxeBuilder({ page }).analyze()
      expect(results.violations).toEqual([])
    })
  }

  test("passes axe with field errors showing", async ({ page }) => {
    await page.goto("/register")
    await page.getByRole("button", { name: "Create account" }).click()
    await expect(page.getByText("Enter a valid email address.")).toBeVisible()
    const results = await new AxeBuilder({ page }).analyze()
    expect(results.violations).toEqual([])
  })

  test("ties field errors to their inputs", async ({ page }) => {
    await page.goto("/login")
    await page.getByRole("button", { name: "Log in" }).click()

    await expect(page.getByLabel("Email")).toHaveAccessibleDescription("Enter a valid email address.")
    await expect(page.getByLabel("Email")).toHaveAttribute("aria-invalid", "true")
    await expect(page.getByLabel("Password", { exact: true })).toHaveAccessibleDescription(
      "Enter your password."
    )
  })

  test("toggles password visibility", async ({ page }) => {
    await page.goto("/login")
    const password = page.getByLabel("Password", { exact: true })
    await expect(password).toHaveAttribute("type", "password")
    await page.getByRole("button", { name: "Show password" }).click()
    await expect(password).toHaveAttribute("type", "text")
    await expect(page.getByRole("button", { name: "Show password" })).toHaveAttribute(
      "aria-pressed",
      "true"
    )
  })

  test("shows the account-created notice after registering", async ({ page }) => {
    await page.route("**/auth/register", (route) => route.fulfill(json(route, 201, { data: {} })))
    await page.goto("/register")
    await page.getByLabel("Email").fill("ada@example.com")
    await page.getByLabel("Password", { exact: true }).fill("correct horse")
    await page.getByLabel("Confirm password", { exact: true }).fill("correct horse")
    await page.getByRole("button", { name: "Create account" }).click()

    await expect(page).toHaveURL(/\/login\?registered=true$/)
    await expect(page.getByRole("status")).toHaveText("Account created. Log in below.")
  })
})
