import AxeBuilder from "@axe-core/playwright"
import { test, expect } from "playwright/test"

test.describe("Not-found page", () => {
  test("an unknown URL shows the not-found page inside the shell", async ({ page }) => {
    await page.goto("/this/does/not/exist")

    await expect(page.getByRole("heading", { level: 1, name: "Page not found" })).toBeVisible()
    await expect(page.getByRole("banner")).toBeVisible()
    await expect(page.getByRole("contentinfo")).toBeVisible()
    const main = page.getByRole("main")
    await expect(main).toHaveCount(1)
    await expect(main.getByRole("link", { name: "Home" })).toHaveAttribute("href", "/")
    await expect(main.getByRole("link", { name: "Catalog" })).toHaveAttribute("href", "/catalog")
  })

  test("passes axe", async ({ page }) => {
    await page.goto("/this/does/not/exist")
    await expect(page.getByRole("heading", { level: 1, name: "Page not found" })).toBeVisible()

    const results = await new AxeBuilder({ page }).analyze()

    expect(results.violations).toEqual([])
  })

  test("an unknown /auth path keeps the auth shell", async ({ page }) => {
    await page.goto("/auth/nope")

    await expect(page.getByRole("heading", { level: 1, name: "Page not found" })).toBeVisible()
    await expect(page.getByRole("navigation")).toHaveCount(0)
  })

  test("a missing Collection names what was not found", async ({ page }) => {
    const cors = (origin: string | undefined) => ({
      "access-control-allow-origin": origin ?? "",
      "access-control-allow-credentials": "true",
    })
    await page.route("**/auth/me", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: cors(route.request().headers().origin),
        body: JSON.stringify({ data: { id: "u1", email: "ada@example.com" } }),
      })
    )
    await page.route(/\/collections\/missing-id$/, (route) => {
      if (route.request().resourceType() !== "fetch") return route.fallback()
      return route.fulfill({
        status: 404,
        contentType: "application/json",
        headers: cors(route.request().headers().origin),
        body: JSON.stringify({ detail: "collection not found" }),
      })
    })

    await page.goto("/collections/missing-id")

    await expect(page.getByRole("heading", { level: 1, name: "Collection not found" })).toBeVisible()
    await expect(page.getByRole("banner")).toBeVisible()
  })
})
