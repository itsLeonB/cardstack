import AxeBuilder from "@axe-core/playwright"
import { test, expect } from "playwright/test"
import type { Page, Route } from "playwright/test"

import {
  SIGNED_IN_TAG,
  signInAsTestUser,
  useSignedInSuite,
} from "./support/clerk-auth"

const API = "http://api:8080"
const COLLECTION = "col-share"
const SLUG = "holiday-binder-3f9a2"

function collection() {
  return {
    id: COLLECTION,
    title: "Holiday Binder",
    description: "",
    maxCardCount: 0,
    cardCount: 3,
    binder: null,
    slug: SLUG,
  }
}

interface ApiResponse {
  data: object | object[]
}

function reply(route: Route, status: number, response: ApiResponse) {
  return route.fulfill({
    status,
    contentType: "application/json",
    headers: {
      "access-control-allow-origin": route.request().headers()["origin"] ?? "",
      "access-control-allow-credentials": "true",
      "access-control-allow-headers": "authorization, content-type",
      "access-control-allow-methods": "GET, OPTIONS",
    },
    body: JSON.stringify(response),
  })
}

async function stubApi(page: Page) {
  await page.route(`${API}/**`, (route) => {
    if (route.request().resourceType() !== "fetch") return route.fallback()
    const path = new URL(route.request().url()).pathname
    if (path === "/collections") {
      return reply(route, 200, { data: [collection()] })
    }
    if (path === `/collections/${COLLECTION}`) {
      return reply(route, 200, { data: collection() })
    }
    if (path === `/collections/${COLLECTION}/entries`) {
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        headers: {
          "access-control-allow-origin":
            route.request().headers()["origin"] ?? "",
          "access-control-allow-credentials": "true",
        },
        body: JSON.stringify({
          data: [],
          meta: { total: 0, page: 1, limit: 24 },
        }),
      })
    }
    if (path === `/collections/${COLLECTION}/facets`) {
      return reply(route, 200, {
        data: { categories: [], expansionSets: [], rarities: [], tags: [] },
      })
    }
    if (path === "/profile") {
      return reply(route, 200, {
        data: { name: "Ash Ketchum", slug: "profile-ash" },
      })
    }
    if (path === "/catalog/series") {
      return reply(route, 200, {
        data: { series: [], ungroupedExpansionSets: [] },
      })
    }
    return route.fallback()
  })
}

test.describe("Collection sharing", { tag: SIGNED_IN_TAG }, () => {
  useSignedInSuite()

  test("keeps publication controls and passes axe in both themes", async ({
    page,
  }) => {
    await stubApi(page)
    await signInAsTestUser(page)
    await page.goto("/")
    await page
      .getByRole("banner")
      .getByRole("link", { name: "Collections" })
      .click()
    await page.getByRole("link", { name: "Holiday Binder" }).click()
    await expect(
      page.getByRole("heading", { name: "Holiday Binder", level: 1 })
    ).toBeVisible()

    const trigger = page.getByRole("button", { name: "Shared" })
    for (const theme of ["Light", "Dark"]) {
      await page
        .getByRole("banner")
        .getByRole("button", { name: "Theme" })
        .click()
      await page.getByRole("menuitemradio", { name: theme }).click()
      await page.keyboard.press("Escape")
      await trigger.click()
      const dialog = page.getByRole("dialog", { name: "Share" })
      await expect(
        dialog.getByRole("switch", { name: "Anyone with the link can view" })
      ).toBeVisible()
      await expect(
        dialog.getByRole("textbox", { name: "Share link" })
      ).toHaveValue(`${new URL(page.url()).origin}/s/profile-ash/${SLUG}`)
      const { violations } = await new AxeBuilder({ page }).analyze()
      expect(violations).toEqual([])
      await page.keyboard.press("Escape")
      await expect(dialog).toBeHidden()
      await expect(trigger).toBeFocused()
    }
  })
})
