import { randomUUID } from "node:crypto"
import { expect, test } from "playwright/test"
import type { Page } from "playwright/test"
import { z } from "zod"

import {
  SIGNED_IN_TAG,
  signInAsUser,
  useSignedInSuite,
} from "./support/clerk-auth"

const API = "http://api:8080"
const cardsSchema = z.object({
  data: z.array(z.object({ id: z.string(), name: z.string() })),
})
const collectionSchema = z.object({
  data: z.object({ id: z.string() }),
})
type ApiInput =
  | { title: string }
  | { cardId: string; quantity: number }
  | { items: { cardId: string; quantity: number }[] }

async function api(
  page: Page,
  path: string,
  method: string,
  body?: ApiInput
): Promise<{ status: number; body: unknown }> {
  return page.evaluate(
    async ({ apiRoot, requestPath, requestMethod, requestBody }) => {
      const token = await window.Clerk.session?.getToken()
      if (!token) throw new Error("Signed-in session is not ready")
      const headers = new Headers({ Authorization: `Bearer ${token}` })
      if (requestBody !== undefined)
        headers.set("Content-Type", "application/json")
      const response = await fetch(`${apiRoot}${requestPath}`, {
        method: requestMethod,
        headers,
        body:
          requestBody === undefined ? undefined : JSON.stringify(requestBody),
      })
      return { status: response.status, body: await response.json() }
    },
    {
      apiRoot: API,
      requestPath: path,
      requestMethod: method,
      requestBody: body,
    }
  )
}

async function collectionQuantity(
  page: Page,
  collectionId: string,
  cardId: string
) {
  const response = await api(
    page,
    `/collections/${collectionId}/entries`,
    "GET"
  )
  expect(response.status).toBe(200)
  const entries = z
    .object({
      data: z.array(
        z.object({
          card: z.object({ id: z.string() }),
          quantity: z.number(),
        })
      ),
    })
    .parse(response.body).data
  return entries.find((entry) => entry.card.id === cardId)?.quantity ?? 0
}

async function createCollection(page: Page, title: string) {
  const response = await api(page, "/collections", "POST", { title })
  expect(response.status).toBe(201)
  return collectionSchema.parse(response.body).data.id
}

async function createDisposableUser(firstName: string) {
  const secret = process.env.CLERK_SECRET_KEY?.trim()
  if (!secret?.startsWith("sk_test_")) {
    throw new Error("Trade E2E requires a Clerk development secret key")
  }
  const headers = {
    Authorization: `Bearer ${secret}`,
    "Content-Type": "application/json",
  }
  const email = `trade-e2e-${randomUUID()}+clerk_test@example.com`
  const response = await fetch("https://api.clerk.com/v1/users", {
    method: "POST",
    headers,
    signal: AbortSignal.timeout(10_000),
    body: JSON.stringify({
      email_address: [email],
      first_name: firstName,
      last_name: "Trade",
      password: `E2e-${randomUUID()}-aA1!`,
    }),
  })
  if (!response.ok)
    throw new Error(
      `Clerk disposable user creation failed: HTTP ${response.status}`
    )
  const user = z.object({ id: z.string().min(1) }).parse(await response.json())
  return { id: user.id, email, headers }
}

async function deleteDisposableUser(
  user: Awaited<ReturnType<typeof createDisposableUser>>
) {
  const response = await fetch(
    `https://api.clerk.com/v1/users/${encodeURIComponent(user.id)}`,
    {
      method: "DELETE",
      headers: user.headers,
      signal: AbortSignal.timeout(10_000),
    }
  )
  if (!response.ok)
    throw new Error(
      `Clerk disposable user cleanup failed: HTTP ${response.status}`
    )
}

async function addOffer(
  page: Page,
  collectionId: string,
  cardId: string,
  cardName: string
) {
  await page.getByLabel("Source Collection").selectOption(collectionId)
  await page.getByLabel("Card", { exact: true }).selectOption(cardId)
  await page.getByLabel("Quantity", { exact: true }).fill("1")
  const itemResponse = page.waitForResponse(
    (response) =>
      response.url().includes("/trades/") &&
      response.url().endsWith("/items") &&
      response.request().method() === "POST"
  )
  await page.getByRole("button", { name: "Add Card", exact: true }).click()
  const response = await itemResponse
  if (!response.ok()) {
    const problem = z
      .object({ detail: z.string().optional() })
      .safeParse(await response.json())
    throw new Error(
      `Trade offer failed (${response.status()}): ${problem.success ? problem.data.detail : "API error"}`
    )
  }
  await expect(
    page.getByRole("button", { name: `Remove ${cardName}`, exact: true })
  ).toBeVisible()
}

async function startTrade(page: Page): Promise<string> {
  await page.goto("/trades")
  await page.getByRole("button", { name: "Start a Trade" }).click()
  await expect(
    page.getByRole("heading", { name: "Trade", level: 1, exact: true })
  ).toBeVisible()
  await expect(page).toHaveURL(/\/trades\/[^/]+$/)
  const slug = new URL(page.url()).pathname.split("/").at(-1)
  if (!slug) throw new Error("Trade URL did not include a slug")
  return slug
}

async function acceptTrade(page: Page) {
  await page.getByRole("button", { name: "Accept the Trade" }).click()
  await expect(
    page.getByText("Accepted the deal.", { exact: true }).first()
  ).toBeVisible()
}

async function lockTrade(owner: Page, partner: Page) {
  await acceptTrade(partner)
  await owner.reload()
  await expect(
    owner.getByRole("button", { name: "Accept the Trade" })
  ).toBeVisible()
  await acceptTrade(owner)
  await expect(owner.getByText("Locked", { exact: true })).toBeVisible()
}

test.describe("Trade exchange", { tag: SIGNED_IN_TAG }, () => {
  useSignedInSuite()

  test("two users exchange cards, and a short offer is refused", async ({
    page,
    browser,
  }) => {
    test.setTimeout(90_000)
    let ownerUser: Awaited<ReturnType<typeof createDisposableUser>> | undefined
    let partnerUser:
      Awaited<ReturnType<typeof createDisposableUser>> | undefined
    const partnerContext = await browser.newContext()
    const partner = await partnerContext.newPage()
    page.setDefaultTimeout(10_000)
    partner.setDefaultTimeout(10_000)
    try {
      // Both seats are new Clerk users, so old inventory cannot change the
      // shortage check. Cleanup deletes only the IDs created by this test.
      ownerUser = await createDisposableUser("Owner")
      partnerUser = await createDisposableUser("Partner")
      await signInAsUser(page, ownerUser.email)
      await signInAsUser(partner, partnerUser.email)

      const catalog = await api(page, "/catalog/cards?limit=2", "GET")
      const cards = cardsSchema.parse(catalog.body).data
      expect(catalog.status).toBe(200)
      expect(cards.length).toBeGreaterThanOrEqual(2)
      const [ownerCard, partnerCard] = cards
      const ownerCollection = await createCollection(
        page,
        `Trade ${randomUUID()}`
      )
      const partnerCollection = await createCollection(
        partner,
        `Trade ${randomUUID()}`
      )
      for (const [userPage, collectionId, cardId] of [
        [page, ownerCollection, ownerCard.id],
        [partner, partnerCollection, partnerCard.id],
      ] as const) {
        const added = await api(
          userPage,
          `/collections/${collectionId}/entries`,
          "POST",
          {
            cardId,
            quantity: 2,
          }
        )
        expect(added.status).toBe(201)
      }

      const slug = await startTrade(page)
      await expect(
        page.getByRole("button", { name: "Add Card", exact: true })
      ).toHaveCount(0)
      await partner.goto(`/trades/${slug}`)
      await partner.getByRole("button", { name: "Join this Trade" }).click()
      await expect(
        partner.getByText("Negotiating", { exact: true })
      ).toBeVisible()
      await page.reload()
      await addOffer(page, ownerCollection, ownerCard.id, ownerCard.name)
      await addOffer(
        partner,
        partnerCollection,
        partnerCard.id,
        partnerCard.name
      )
      const inboxTitle = (
        await partner.getByRole("heading", { level: 2 }).first().textContent()
      )?.trim()
      if (!inboxTitle)
        throw new Error("Trade page did not show the owner's name")
      await lockTrade(page, partner)

      await page.getByRole("button", { name: "Confirm handover" }).click()
      await expect(
        page
          .getByRole("status")
          .filter({ hasText: "Waiting on the other side" })
      ).toBeVisible()
      await partner.reload()
      await partner.getByRole("button", { name: "Confirm handover" }).click()
      await expect(partner.getByText("The exchange is complete.")).toBeVisible()
      await partner.getByRole("link", { name: "View your Trade Inbox" }).click()
      await expect(
        partner.getByRole("heading", { name: inboxTitle! })
      ).toBeVisible()
      await expect(
        partner.getByRole("link", { name: ownerCard.name })
      ).toBeVisible()
      const ownerTrade = await api(page, `/trades/${slug}`, "GET")
      const partnerTrade = await api(partner, `/trades/${slug}`, "GET")
      expect(ownerTrade.status).toBe(200)
      expect(partnerTrade.status).toBe(200)
      const inboxSchema = z.object({ data: z.object({ inboxId: z.string() }) })
      const ownerInbox = inboxSchema.parse(ownerTrade.body).data.inboxId
      const partnerInbox = inboxSchema.parse(partnerTrade.body).data.inboxId
      expect(
        await collectionQuantity(page, ownerCollection, ownerCard.id)
      ).toBe(1)
      expect(
        await collectionQuantity(partner, partnerCollection, partnerCard.id)
      ).toBe(1)
      expect(await collectionQuantity(page, ownerInbox, partnerCard.id)).toBe(1)
      expect(
        await collectionQuantity(partner, partnerInbox, ownerCard.id)
      ).toBe(1)
      await expect(partner.getByLabel("Quantity")).toHaveCount(0)
      await expect(
        partner.getByRole("button", { name: /add|delete|rename/i })
      ).toHaveCount(0)

      // A second exchange uses the same two users. After both accept, the
      // owner removes the offered copy; the other side's confirmation must
      // refuse the whole Trade and name the short Card.
      const shortSlug = await startTrade(page)
      await partner.goto(`/trades/${shortSlug}`)
      await partner.getByRole("button", { name: "Join this Trade" }).click()
      await expect(
        partner.getByText("Negotiating", { exact: true })
      ).toBeVisible()
      await page.reload()
      await addOffer(page, ownerCollection, ownerCard.id, ownerCard.name)
      await addOffer(
        partner,
        partnerCollection,
        partnerCard.id,
        partnerCard.name
      )
      await lockTrade(page, partner)
      await partner.reload()
      await expect(
        partner.getByRole("button", { name: "Confirm handover" })
      ).toBeVisible({ timeout: 20_000 })
      const shortage = await api(
        page,
        `/collections/${ownerCollection}/entries`,
        "PATCH",
        {
          items: [{ cardId: ownerCard.id, quantity: 0 }],
        }
      )
      expect(shortage.status).toBe(200)
      expect(shortage.body).toMatchObject({
        data: [{ cardId: ownerCard.id, quantity: 0, status: "removed" }],
      })
      const holdings = await api(
        page,
        `/inventory/cards/${ownerCard.id}/holdings`,
        "GET"
      )
      expect(holdings.status).toBe(200)
      const remainingHoldings = z
        .object({ data: z.array(z.object({ quantity: z.number() })) })
        .parse(holdings.body)
      expect(
        remainingHoldings.data.reduce(
          (total, holding) => total + holding.quantity,
          0
        )
      ).toBe(0)
      const unchanged = [
        [partner, partnerCollection, partnerCard.id],
        [page, ownerInbox, partnerCard.id],
        [partner, partnerInbox, ownerCard.id],
      ] as const
      const beforeRefusal = await Promise.all(
        unchanged.map(([userPage, collectionId, cardId]) =>
          collectionQuantity(userPage, collectionId, cardId)
        )
      )
      expect(beforeRefusal).toEqual([1, 1, 1])
      await test.step("partner confirms the short Trade", async () => {
        const confirmationResponse = partner.waitForResponse(
          (response) =>
            new URL(response.url()).pathname ===
              `/trades/${shortSlug}/confirm` &&
            response.request().method() === "POST",
          { timeout: 20_000 }
        )
        await partner
          .getByRole("button", { name: "Confirm handover" })
          .click({ timeout: 8_000 })
        expect((await confirmationResponse).ok()).toBe(true)
      })
      const refusalResponse = page.waitForResponse(
        (response) =>
          response.url().includes(`/trades/${shortSlug}/confirm`) &&
          response.request().method() === "POST",
        { timeout: 20_000 }
      )
      await test.step("owner submits the short confirmation", async () => {
        await page
          .getByRole("button", { name: "Confirm handover" })
          .click({ timeout: 8_000 })
      })
      const refusal = await refusalResponse
      expect(refusal.status()).not.toBe(200)
      const refusalBody = z
        .object({ detail: z.string() })
        .parse(await refusal.json())
      expect(refusalBody.detail).toContain(ownerCard.name)
      await expect(page.getByRole("alert")).toContainText(ownerCard.name)
      await expect(page.getByText("Locked", { exact: true })).toBeVisible()
      const afterRefusal = await Promise.all(
        unchanged.map(([userPage, collectionId, cardId]) =>
          collectionQuantity(userPage, collectionId, cardId)
        )
      )
      expect(afterRefusal).toEqual(beforeRefusal)
    } finally {
      await partnerContext.close().catch(() => undefined)
      await Promise.all([
        ...(ownerUser ? [deleteDisposableUser(ownerUser)] : []),
        ...(partnerUser ? [deleteDisposableUser(partnerUser)] : []),
      ])
    }
  })
})
