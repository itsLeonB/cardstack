import { renderHook } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { useObjectUrl } from "./use-page-scan"

describe("useObjectUrl", () => {
  it("never hands out the URL of a blob it has replaced", () => {
    const made = ["blob:first", "blob:second"]
    const revoked: string[] = []
    URL.createObjectURL = vi.fn(() => made.shift() ?? "")
    URL.revokeObjectURL = vi.fn((url: string) => {
      revoked.push(url)
    })
    const returned: (string | undefined)[] = []
    const { result, rerender } = renderHook(
      ({ blob }: { blob: Blob | undefined }) => {
        const url = useObjectUrl(blob)
        returned.push(url)
        return url
      },
      { initialProps: { blob: new Blob(["first"]) } }
    )
    expect(result.current).toBe("blob:first")

    const before = returned.length
    rerender({ blob: new Blob(["second"]) })

    expect(result.current).toBe("blob:second")
    // The render that switches blobs runs after the first URL is revoked.
    expect(returned.slice(before)).not.toContain("blob:first")
    expect(revoked).toEqual(["blob:first"])
  })
})
