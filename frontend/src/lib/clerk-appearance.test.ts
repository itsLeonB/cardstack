import { describe, expect, it } from "vitest"
import { clerkAppearance } from "./clerk-appearance"

describe("clerkAppearance", () => {
  const elements = new Map(Object.entries(clerkAppearance?.elements ?? {}))

  it.each([
    "footerActionLink",
    "formFieldAction",
    "formResendCodeLink",
    "backLink",
    "identityPreviewEditButton",
  ] as const)(
    "draws %s in the foreground colour with an underline, not in amber",
    (name) => {
      expect(elements.get(name)).toMatchObject({
        color: "var(--foreground)",
        textDecoration: "underline",
      })
    }
  )

  it("keeps amber for fills only, as variables that follow the theme", () => {
    expect(clerkAppearance?.variables?.colorPrimary).toBe("var(--primary)")
    expect(clerkAppearance?.variables?.colorBackground).toBe("var(--card)")
  })
})
