import { vi } from "vitest"
import type { useUser } from "@clerk/react"

// Fakes for `@clerk/react`: its hooks and provider need a real Clerk instance
// talking to Clerk's servers, so tests replace the module at its edge. Use it as
//   vi.mock("@clerk/react", () => import("@/test-clerk").then((m) => m.clerkModule()))
// then set each hook's result with `vi.mocked(useUser).mockReturnValue(...)`.
export type ClerkState = "loading" | "guest" | "signed-in"

export interface ClerkUserData {
  id: string
  fullName: string | null
  primaryEmailAddress: { emailAddress: string } | null
}

export const ADA: ClerkUserData = {
  id: "user_ada",
  fullName: "Ada Lovelace",
  primaryEmailAddress: { emailAddress: "ada@example.com" },
}

export interface ClerkComponentProps {
  routing?: string
  path?: string
  forceRedirectUrl?: string
  signUpUrl?: string
  signInUrl?: string
}

/**
 * The module factory. `SignIn` and `SignUp` render their props as JSON, so a
 * test can read what the app asked Clerk for.
 */
export function clerkModule() {
  return {
    ClerkProvider: vi.fn(({ children }: { children: React.ReactNode }) => (
      <div data-testid="clerk">{children}</div>
    )),
    useAuth: vi.fn(() => ({ isLoaded: false })),
    useClerk: vi.fn(),
    useUser: vi.fn(),
    SignIn: (props: ClerkComponentProps) => (
      <div data-testid="sign-in" data-props={JSON.stringify(props)} />
    ),
    SignUp: (props: ClerkComponentProps) => (
      <div data-testid="sign-up" data-props={JSON.stringify(props)} />
    ),
  }
}

export function clerkUserResult(
  state: ClerkState,
  user: ClerkUserData = ADA
): ReturnType<typeof useUser> {
  const result =
    state === "loading"
      ? { isLoaded: false, isSignedIn: undefined, user: undefined }
      : state === "guest"
        ? { isLoaded: true, isSignedIn: false, user: null }
        : { isLoaded: true, isSignedIn: true, user }
  // SAFETY: a partial result covering only the fields `useSession` reads.
  return result as any
}
