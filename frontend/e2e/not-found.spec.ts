import AxeBuilder from "@axe-core/playwright"
import { test, expect } from "playwright/test"

test.describe("Not-found page", () => {
  test("an unknown URL shows the not-found page inside the shell", async ({
    page,
  }) => {
    await page.goto("/this/does/not/exist")

    await expect(
      page.getByRole("heading", { level: 1, name: "Page not found" })
    ).toBeVisible()
    await expect(page.getByRole("banner")).toBeVisible()
    await expect(page.getByRole("contentinfo")).toBeVisible()
    const main = page.getByRole("main")
    await expect(main).toHaveCount(1)
    await expect(main.getByRole("link", { name: "Home" })).toHaveAttribute(
      "href",
      "/"
    )
    await expect(main.getByRole("link", { name: "Catalog" })).toHaveAttribute(
      "href",
      "/catalog"
    )
  })

  test("passes axe", async ({ page }) => {
    await page.goto("/this/does/not/exist")
    await expect(
      page.getByRole("heading", { level: 1, name: "Page not found" })
    ).toBeVisible()

    const results = await new AxeBuilder({ page }).analyze()

    expect(results.violations).toEqual([])
  })

  const cors = (origin: string | undefined) => ({
    "access-control-allow-origin": origin ?? "",
    "access-control-allow-credentials": "true",
  })

  test("a missing Collection names what was not found, on its page and its edit page", async ({
    page,
  }) => {
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

    for (const path of [
      "/collections/missing-id",
      "/collections/missing-id/edit",
    ]) {
      await page.goto(path)

      await expect(
        page.getByRole("heading", { level: 1, name: "Collection not found" })
      ).toBeVisible()
      await expect(page.getByRole("banner")).toBeVisible()
    }
  })

  test("a missing Card names what was not found", async ({ page }) => {
    await page.route(/\/catalog\/cards\?/, (route) => {
      if (route.request().resourceType() !== "fetch") return route.fallback()
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: cors(route.request().headers().origin),
        body: JSON.stringify({
          data: [],
          meta: { total: 0, page: 1, limit: 1 },
        }),
      })
    })

    await page.goto("/catalog/cards/no-such-set/000")

    await expect(
      page.getByRole("heading", { level: 1, name: "Card not found" })
    ).toBeVisible()
    await expect(page.getByRole("banner")).toBeVisible()
  })
})
