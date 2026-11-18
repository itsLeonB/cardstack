import { useState } from "react"
import { RiShareLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

export function LinkDelivery({
  link,
  label = "Share link",
}: {
  link: string
  label?: string
}) {
  const [message, setMessage] = useState("")
  const [copyFailureCount, setCopyFailureCount] = useState(0)
  const canShare = Boolean(globalThis.navigator?.share)

  async function copyLink() {
    setMessage("")
    try {
      if (!navigator.clipboard?.writeText)
        throw new Error("Clipboard unavailable")
      await navigator.clipboard.writeText(link)
      setCopyFailureCount(0)
      setMessage("Copied")
    } catch {
      setCopyFailureCount((count) => count + 1)
      setMessage("Could not copy the link. Select it and copy it by hand.")
    }
  }

  async function shareLink() {
    setMessage("")
    try {
      await navigator.share({ url: link })
    } catch (error) {
      if (error instanceof DOMException && error.name === "AbortError") return
      setMessage(
        "Could not share the link. You can copy it or select it and copy it by hand."
      )
    }
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap gap-2">
        <Input readOnly aria-label={label} value={link} />
        <Button
          variant="outline"
          className="min-h-11"
          onClick={() => void copyLink()}
        >
          Copy
        </Button>
        {canShare && (
          <Button
            variant="outline"
            className="min-h-11"
            onClick={() => void shareLink()}
          >
            <RiShareLine data-icon="inline-start" />
            Share…
          </Button>
        )}
      </div>
      <p role="status" aria-live="polite" className="text-sm">
        {message.startsWith("Could not copy")
          ? `${message} Attempt ${copyFailureCount}.`
          : message}
      </p>
    </div>
  )
}
