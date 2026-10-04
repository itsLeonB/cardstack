import { Link } from "@tanstack/react-router"
import { RiUserLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useSession, useSignOut } from "@/lib/session"

export function UserMenu() {
  const { user } = useSession()
  const { signOut, isPending } = useSignOut()
  const label = user?.name ?? user?.email

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={<Button variant="ghost" aria-label="User menu" />}
      >
        <RiUserLine />
        <span className="hidden max-w-40 truncate sm:inline">{label}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <div className="px-2 py-1.5 text-sm">
          {user?.name && <p className="font-medium">{user.name}</p>}
          <p className="text-muted-foreground">{user?.email}</p>
        </div>
        <DropdownMenuItem render={<Link to="/account" />}>
          Account
        </DropdownMenuItem>
        <DropdownMenuItem disabled={isPending} onClick={() => void signOut()}>
          Log out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
