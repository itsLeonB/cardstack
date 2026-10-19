// Fake for `@clerk/react`: its provider and hooks need a real Clerk instance
// talking to Clerk's servers, so tests replace the module at its edge. Use it as
//   vi.mock("@clerk/react", () => import("@/test-clerk").then((m) => m.clerkModule()))
// The provider passes its children through and the hooks report that Clerk has
// not loaded, so the app's Clerk bridge stays idle.
export function clerkModule() {
  return {
    ClerkProvider: ({ children }: { children: React.ReactNode }) => children,
    useAuth: () => ({ isLoaded: false }),
    useClerk: () => undefined,
  }
}
