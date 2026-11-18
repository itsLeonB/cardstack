import { chmod, mkdir, rm } from "node:fs/promises"
import { dirname, resolve } from "node:path"
import {
  clerk,
  clerkSetup,
  setupClerkTestingToken,
} from "@clerk/testing/playwright"
import { chromium } from "playwright/test"
import type { FullConfig } from "playwright/test"
import { readClerkE2eCredentials } from "./support/clerk-credentials"

const authStatePath = resolve("playwright/.clerk/user.json")

export default async function globalSetup(config: FullConfig) {
  const credentials = readClerkE2eCredentials(process.env)
  if (!credentials) return
  await clerkSetup({
    publishableKey: process.env.VITE_CLERK_PUBLISHABLE_KEY,
  })

  const baseURL = config.projects.find(
    (project) => project.name === "chromium-signed-in"
  )?.use.baseURL
  if (!baseURL) throw new Error("Playwright baseURL is missing")

  const directory = dirname(authStatePath)
  await mkdir(directory, { recursive: true, mode: 0o700 })
  await chmod(directory, 0o700)
  const browser = await chromium.launch()
  try {
    const context = await browser.newContext({ baseURL })
    try {
      const page = await context.newPage()
      await setupClerkTestingToken({ page })
      await page.goto("/")
      await clerk.signIn({ page, emailAddress: credentials.email })
      await context.storageState({ path: authStatePath })
      await chmod(authStatePath, 0o600)
    } finally {
      await context.close()
    }
  } catch (error) {
    await rm(authStatePath, { force: true })
    throw error
  } finally {
    await browser.close()
  }
  return async () => rm(authStatePath, { force: true })
}
