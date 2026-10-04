import type { Page } from "playwright/test"

// Empty paginated Collection and Inventory lists, so a signed-in page (the
// dashboard at / included) renders without depending on what the backend
// holds for the test user. Register before signing in: the shell reads these
// as soon as Clerk reports a session. The real session token still
// authenticates every other request.
export async function stubEmptyLists(page: Page) {
  await page.route(/\/(collections|inventory)/, (route) => {
    if (route.request().resourceType() !== "fetch") return route.fallback()
    return route.fulfill({
      status: 200,
      contentType: "application/json",
      headers: {
        // Echo the caller's origin instead of hard-coding the port.
        "access-control-allow-origin": route.request().headers().origin ?? "",
      },
      body: JSON.stringify({
        data: [],
        meta: { total: 0, page: 1, limit: 24 },
      }),
    })
  })
}
