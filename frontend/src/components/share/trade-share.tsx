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

export function TradeShare({ link }: { link: string }) {
  return (
    <Dialog>
      <DialogTrigger
        render={
          <Button className="min-h-11">
            <RiShareLine data-icon="inline-start" />
            Invite
          </Button>
        }
      />
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Share Trade link</DialogTitle>
          <DialogDescription>
            Send this link to someone to open this Trade.
          </DialogDescription>
        </DialogHeader>
        <LinkDelivery key={link} link={link} label="Trade link" />
      </DialogContent>
    </Dialog>
  )
}
