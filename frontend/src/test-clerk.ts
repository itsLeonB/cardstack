import type { useUser } from "@clerk/react"

// What `useUser()` from `@clerk/react` returns in each auth state, for tests
// that fake Clerk's hooks (the real ones need a ClerkProvider talking to Clerk).
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
