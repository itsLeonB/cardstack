import type { ClerkProviderProps } from "@clerk/react"

// Clerk draws every link-like control in `colorPrimary`, which is our amber:
// fine as a button fill, unreadable as text on a light surface (frontend.md).
// The app's own inline links are `text-foreground` with an underline, so these
// controls get the same look, in light and dark.
const linkStyle = {
  color: "var(--foreground)",
  textDecoration: "underline",
  textUnderlineOffset: "4px",
  "&:hover": { color: "var(--foreground)", textDecoration: "underline" },
}

// The app's own CSS variables (styles.css), so Clerk's prebuilt components
// follow `ThemeProvider` for free: the `.dark` class swaps the variables and
// Clerk, which reads them live, repaints. No theme state is needed here.
export const clerkAppearance: ClerkProviderProps["appearance"] = {
  variables: {
    colorPrimary: "var(--primary)",
    colorPrimaryForeground: "var(--primary-foreground)",
    colorBackground: "var(--card)",
    colorForeground: "var(--card-foreground)",
    colorMutedForeground: "var(--muted-foreground)",
    colorMuted: "var(--muted)",
    colorInput: "var(--background)",
    colorInputForeground: "var(--foreground)",
    colorBorder: "var(--border)",
    colorDanger: "var(--destructive)",
    colorRing: "var(--ring)",
    fontFamily: "inherit",
    borderRadius: "var(--radius)",
  },
  elements: {
    // Fill the narrow auth column instead of Clerk's fixed card width.
    rootBox: "w-full",
    cardBox: "w-full max-w-none",
    footerActionLink: linkStyle,
    formFieldAction: linkStyle,
    formResendCodeLink: linkStyle,
    backLink: linkStyle,
    identityPreviewEditButton: linkStyle,
  },
}
