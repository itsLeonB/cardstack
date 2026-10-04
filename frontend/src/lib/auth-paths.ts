// The auth addresses, shared so the Clerk provider, the pages, the guards and
// sign-out cannot drift apart. `as const` keeps them usable as typed route targets.
export const LOGIN_PATH = "/auth/login" as const
export const REGISTER_PATH = "/auth/register" as const
/** Where sign-in and sign-up land when no `redirect` was asked for. */
export const AFTER_SIGN_IN_PATH = "/account" as const
