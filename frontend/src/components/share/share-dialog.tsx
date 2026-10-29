import { useId, useState } from "react"
import { RiShareLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"

interface ShareDialogProps {
  /** Whether it is Shared now: the trigger and the switch show it. */
  shared: boolean
  /** The read-only link, shown while Shared. */
  link: string
  onShare: () => void
  onUnshare: () => void
  /** A change is in flight: the switch is disabled. */
  pending?: boolean
  /** Why the last change failed, shown inline. */
  error?: string
}

type CopyOutcome = "copied" | "failed"

/**
 * Turns sharing on or off and copies the link. Holds no data of its own: the
 * caller passes the state, the link and the two callbacks, so the same dialog
 * serves every Shared thing (a Collection now, a Wishlist later).
 */
export function ShareDialog({
  shared,
  link,
  onShare,
  onUnshare,
  pending = false,
  error,
}: ShareDialogProps) {
  const [open, setOpen] = useState(false)
  const [confirmingOff, setConfirmingOff] = useState(false)
  // Keyed by the link it was made for, so a new link (a re-share) starts clean.
  const [copy, setCopy] = useState<{
    link: string
    outcome: CopyOutcome
  } | null>(null)
  const switchId = useId()

  function handleOpenChange(next: boolean) {
    setOpen(next)
    setConfirmingOff(false)
    setCopy(null)
  }

  // Turning sharing off ends every link at once, so it asks first.
  function handleToggle(next: boolean) {
    if (next) onShare()
    else setConfirmingOff(true)
  }

  // Clearing first makes a repeat copy announce again: the live region changes
  // from empty to "Copied" each time, not from "Copied" to "Copied".
  function copyLink() {
    setCopy(null)
    navigator.clipboard
      .writeText(link)
      .then(() => setCopy({ link, outcome: "copied" }))
      .catch(() => setCopy({ link, outcome: "failed" }))
  }

  const outcome = copy?.link === link ? copy.outcome : null
  const copyMessage =
    outcome === "copied"
      ? "Copied"
      : outcome === "failed"
        ? "Could not copy the link. Select it and copy it by hand."
        : ""

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger
        render={
          <Button variant="outline" size="sm">
            <RiShareLine data-icon="inline-start" />
            {shared ? "Shared" : "Share"}
          </Button>
        }
      />
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Share</DialogTitle>
          <DialogDescription>
            Share a read-only view. Turning sharing off ends every link.
          </DialogDescription>
        </DialogHeader>

        {!shared && (
          <p className="text-sm">
            Your full name appears in the link and on the page.
          </p>
        )}
        <div className="flex items-center justify-between gap-4">
          <Label htmlFor={switchId}>Anyone with the link can view</Label>
          <Switch
            id={switchId}
            checked={shared}
            disabled={pending}
            onCheckedChange={handleToggle}
          />
        </div>

        {shared && !confirmingOff && (
          <div className="flex flex-col gap-2">
            <div className="flex gap-2">
              <Input readOnly aria-label="Share link" value={link} />
              <Button variant="outline" onClick={copyLink}>
                Copy
              </Button>
            </div>
            <p aria-live="polite" className="text-sm">
              {copyMessage}
            </p>
          </div>
        )}

        {shared && confirmingOff && (
          <div className="flex flex-col gap-3">
            <p className="text-sm">
              Existing links will stop working. Sharing again makes a new link.
            </p>
            <div className="flex flex-wrap gap-2">
              <Button
                variant="destructive"
                disabled={pending}
                onClick={() => {
                  setConfirmingOff(false)
                  onUnshare()
                }}
              >
                Stop sharing
              </Button>
              <Button variant="outline" onClick={() => setConfirmingOff(false)}>
                Keep sharing
              </Button>
            </div>
          </div>
        )}

        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
      </DialogContent>
    </Dialog>
  )
}
