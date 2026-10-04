import AxeBuilder from "@axe-core/playwright"
import { test, expect } from "playwright/test"
import type { Page } from "playwright/test"

import { stubEmptyLists } from "./support/api-stubs"
import {
  signInAsTestUser,
  signInThroughForm,
  SIGNED_IN_TAG,
  useSignedInSuite,
} from "./support/clerk-auth"

// Sign-in and sign-up are Clerk's prebuilt components (ADR-0015). Guest checks
// here need no setup. The signed-in ones sign in through Clerk's real
// development instance (see support/clerk-auth.ts) and skip without its
// credentials.

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

  test.describe("signed in", { tag: SIGNED_IN_TAG }, () => {
    useSignedInSuite()

    test("returns to the protected page, query string included, after login", async ({
      page,
    }) => {
      await stubEmptyLists(page)
      await signInThroughForm(
        page,
        "/auth/login?redirect=%2Fcollections%3Fq%3Dbinder"
      )

      await expect(page).toHaveURL(/\/collections\?q=binder$/, {
        timeout: 15_000,
      })
      await expect(
        page.getByRole("heading", { level: 1, name: "Collections" })
      ).toBeVisible()
    })

    test("ignores an external redirect target after login", async ({
      page,
    }) => {
      await signInThroughForm(
        page,
        "/auth/login?redirect=https://evil.example/"
      )

      await expect(page).toHaveURL(/localhost:\d+\/account$/, {
        timeout: 15_000,
      })
    })

    for (const path of [
      "/auth/login",
      "/auth/register",
      "/auth/foo",
      "/auth",
    ]) {
      test(`sends a signed-in user from ${path} to /`, async ({ page }) => {
        await stubEmptyLists(page)
        await signInAsTestUser(page)
        await page.goto(path)

        await expect(page).toHaveURL(/\/$/)
        await expect(
          page.getByRole("heading", { level: 1, name: "Welcome back" })
        ).toBeVisible()
      })
    }
  })

  // The page's one h1 is Clerk's own header title; the app adds none.
  for (const path of ["/auth/login", "/auth/register"]) {
    test(`lets a guest reach ${path}, with exactly one h1`, async ({
      page,
    }) => {
      await page.goto(path)
      await expect(page.getByRole("heading", { level: 1 })).toHaveCount(1)
      await expect(page.getByRole("heading", { level: 1 })).toBeVisible()
    })
  }
})

// Clerk's development-mode strip (the orange bar, shown only on a development
// instance) is third-party branding whose text fails color-contrast, and its
// colour is Clerk's default warning orange, which also colours real warnings,
// so it is not restyled through the appearance API. Contrast failures against
// exactly that background are dropped from the scan; every other node, and
// every other rule, still counts. (Clerk's unsafe_disableDevelopmentModeWarnings
// would hide the strip itself, which is more than this needs.)
const CLERK_DEV_MODE_ORANGE = "#f36b16"

async function scanForViolations(page: Page) {
  const { violations } = await new AxeBuilder({ page }).analyze()
  return violations
    .map((violation) =>
      violation.id === "color-contrast"
        ? {
            ...violation,
            nodes: violation.nodes.filter(
              (node) =>
                !node.any.some(
                  (check) =>
                    String(check.data?.bgColor).toLowerCase() ===
                    CLERK_DEV_MODE_ORANGE
                )
            ),
          }
        : violation
    )
    .filter((violation) => violation.nodes.length > 0)
}

test.describe("Auth pages", () => {
  for (const path of ["/auth/login", "/auth/register"]) {
    test(`passes axe on ${path}`, async ({ page }) => {
      await page.goto(path)
      await expect(page.getByRole("heading", { level: 1 })).toBeVisible()
      expect(await scanForViolations(page)).toEqual([])
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
