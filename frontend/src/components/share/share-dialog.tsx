import { useId, useState } from "react"
import { RiShareLine } from "@remixicon/react"
import { LinkDelivery } from "@/components/share/link-delivery"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
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
  const switchId = useId()

  function handleOpenChange(next: boolean) {
    setOpen(next)
    setConfirmingOff(false)
  }

  // Turning sharing off ends every link at once, so it asks first.
  function handleToggle(next: boolean) {
    if (next) onShare()
    else setConfirmingOff(true)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger
        render={
          <Button
            variant="outline"
            className="h-auto min-h-11 whitespace-normal"
          >
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
          <LinkDelivery key={`${link}:${open}`} link={link} />
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
