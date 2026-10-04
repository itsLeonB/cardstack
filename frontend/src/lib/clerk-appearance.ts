import type { ClerkProviderProps } from "@clerk/react"

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
  },
}
