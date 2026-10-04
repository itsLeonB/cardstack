import { test, expect } from "playwright/test"
import type { Page } from "playwright/test"

// Metadata is set from the browser (SPA mode), so these read the live document head.

const cors = (origin: string | undefined) => ({
  "access-control-allow-origin": origin ?? "",
  "access-control-allow-credentials": "true",
})

async function signIn(page: Page) {
  await page.route("**/auth/me", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      headers: cors(route.request().headers().origin),
      body: JSON.stringify({ data: { id: "u1", email: "ada@example.com" } }),
    })
  )
}

const robotsMeta = (page: Page) => page.locator('head meta[name="robots"]')

test.describe("Titles and indexing", () => {
  test("the landing is indexable and has a descriptive title and static Open Graph tags", async ({
    page,
  }) => {
    await page.goto("/")

    await expect(page).toHaveTitle(
      "Track every Pokémon card you own · Cardstack"
    )
    await expect(robotsMeta(page)).toHaveCount(0)
    await expect(page.locator('head meta[property="og:title"]')).toHaveCount(1)
    await expect(
      page.locator('head meta[property="og:description"]')
    ).toHaveCount(1)
    await expect(page.locator('head meta[name="description"]')).toHaveCount(1)
  })

  test("the public Catalog is indexable and has its own title and description", async ({
    page,
  }) => {
    await page.route(/\/catalog\/series/, (route) => {
      if (route.request().resourceType() !== "fetch") return route.fallback()
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: cors(route.request().headers().origin),
        body: JSON.stringify({ data: [] }),
      })
    })
    await page.goto("/catalog")

    await expect(page).toHaveTitle("Catalog · Cardstack")
    await expect(robotsMeta(page)).toHaveCount(0)
    await expect(page.locator('head meta[name="description"]')).toHaveCount(1)
    await expect(page.locator('head meta[name="description"]')).toHaveAttribute(
      "content",
      /Browse every Pokémon TCG series/
    )
  })

  // The next two reach their page by client-side navigation: the dev server
  // server-renders a direct load, where browser-level API stubs don't apply and
  // a guard can drop the child route's head. Production is a static SPA shell,
  // so there the client router always runs the heads, as it does here.
  test("login and register are noindex", async ({ page }) => {
    await page.goto("/")
    await expect(async () => {
      await page
        .getByRole("banner")
        .getByRole("link", { name: "Log in" })
        .click()
      // The route guard waits for Clerk to load before it lets the page in.
      await expect(page).toHaveURL(/\/auth\/login/, { timeout: 5000 })
    }).toPass()
    await expect(page).toHaveTitle("Log in · Cardstack")
    await expect(robotsMeta(page)).toHaveAttribute("content", "noindex")
    await expect(robotsMeta(page)).toHaveCount(1)

    await page.goBack()
    await page
      .getByRole("banner")
      .getByRole("link", { name: "Register" })
      .click()
    await expect(page).toHaveTitle("Create an account · Cardstack")
    await expect(robotsMeta(page)).toHaveAttribute("content", "noindex")
  })

  // FIXME(ticket 09): needs a real Clerk session; signIn() stubbed the removed /auth/me.
  test.fixme("an authenticated page is noindex", async ({ page }) => {
    await signIn(page)
    await page.goto("/")
    await expect(async () => {
      await page.getByRole("button", { name: "User menu" }).click()
      await page
        .getByRole("menuitem", { name: "Account" })
        .click({ timeout: 1000 })
    }).toPass()

    await expect(page).toHaveTitle("Account · Cardstack")
    await expect(robotsMeta(page)).toHaveAttribute("content", "noindex")
    await expect(robotsMeta(page)).toHaveCount(1)
  })

  test("robots.txt disallows the authenticated paths", async ({ request }) => {
    const body = await (await request.get("/robots.txt")).text()

    for (const path of ["/collections", "/inventory", "/account"]) {
      expect(body).toContain(`Disallow: ${path}`)
    }
    expect(body).not.toContain("Disallow: /auth")
  })
})
