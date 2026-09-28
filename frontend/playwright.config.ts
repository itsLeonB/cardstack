import { defineConfig, devices } from "playwright/test"

// `playwright` (not `@playwright/test`) is the devDependency this repo
// declares — see docs/agents/testing.md's "driving a headless browser"
// section for why. Its `playwright/test` subpath re-exports the same
// test runner API as `@playwright/test`, so we import from there instead
// of adding a second, overlapping dependency.

const PORT = 3000
const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? `http://localhost:${PORT}`

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  // Keep this at its default location: CI uploads playwright-report/ as a
  // build artifact on failure by that exact path.
  reporter: "html",
  use: {
    baseURL,
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  // Only start a local dev server when there's no already-deployed target
  // to test against (CI sets PLAYWRIGHT_BASE_URL to the PR's Vercel
  // preview and runs the suite against real deployed frontend + backend).
  // Locally, `bunx playwright test` then just works without a manually
  // started dev server — but the dev server still needs a real backend
  // and seeded Postgres running separately (see docs/agents/testing.md
  // and this repo's README); Playwright's config has no way to start
  // those for you.
  webServer: process.env.PLAYWRIGHT_BASE_URL
    ? undefined
    : {
        command: "bun run dev",
        url: baseURL,
        reuseExistingServer: !process.env.CI,
      },
})
