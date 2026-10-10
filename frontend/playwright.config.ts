import { defineConfig, devices } from "playwright/test"

// `playwright` (not `@playwright/test`) is the devDependency this repo
// declares — see docs/agents/testing.md's "driving a headless browser"
// section for why. Its `playwright/test` subpath re-exports the same
// test runner API as `@playwright/test`, so we import from there instead
// of adding a second, overlapping dependency.

// The dev server reads frontend/.env itself, but this process does not. Load it
// so a local run finds the Clerk credentials documented in .env.example (see
// docs/agents/testing.md, "End-to-end authentication"); variables already in
// the environment win, and CI has no .env file.
try {
  process.loadEnvFile()
} catch {
  // No .env file: the environment alone decides.
}

const PORT = 3000
const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? `http://localhost:${PORT}`

export default defineConfig({
  testDir: "./e2e",
  // Only specs: e2e/support/*.test.ts are vitest unit tests.
  testMatch: "**/*.spec.ts",
  // Fetches the Clerk Testing Token once for the run (a no-op without the
  // Clerk credentials). A global setup, not a setup project: workers inherit
  // its environment, which is how the token reaches them.
  globalSetup: "./e2e/global-setup.ts",
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
      grepInvert: /@signed-in/,
      use: { ...devices["Desktop Chrome"] },
    },
    // Untraced on purpose: see SIGNED_IN_TAG in e2e/support/clerk-auth.ts.
    {
      name: "chromium-signed-in",
      grep: /@signed-in/,
      use: { ...devices["Desktop Chrome"], trace: "off" },
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
        // The dev server has no use for the e2e credentials.
        env: {
          // The scan screen only exists in a build with this flag on.
          VITE_SCAN_ENABLED: "true",
          CLERK_SECRET_KEY: "",
          E2E_CLERK_USER_EMAIL: "",
          E2E_CLERK_USER_PASSWORD: "",
        },
      },
})
