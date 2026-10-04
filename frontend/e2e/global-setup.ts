import { clerkSetup } from "@clerk/testing/playwright"
import { readClerkE2eCredentials } from "./support/clerk-credentials"

// Fetches one Clerk Testing Token for the whole run and exposes it (with the
// instance's Frontend API) to the workers through the environment, which they
// inherit because they start after this. Skipped when the credentials are
// absent, so guest-only runs (fork pull requests, local) never call Clerk.
export default async function globalSetup() {
  if (!readClerkE2eCredentials(process.env)) return
  await clerkSetup({
    publishableKey: process.env.VITE_CLERK_PUBLISHABLE_KEY,
  })
}
