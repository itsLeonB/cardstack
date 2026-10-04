import type { ReactNode } from "react"
import { Link } from "@tanstack/react-router"
import { cn } from "cn"
import { LOGIN_PATH } from "@/lib/auth-paths"
import { isSameOriginPath } from "@/lib/route-guard"

/**
 * Link to sign-in from a prompt on a public page. `redirect` is the path (with
 * search) to come back to; one that is not a same-origin path is dropped, as
 * the login page would drop it anyway.
 */
export function SignInLink({
  redirect,
  className,
  children,
}: {
  redirect?: string
  className?: string
  children: ReactNode
}) {
  return (
    <Link
      to={LOGIN_PATH}
      search={{ redirect: isSameOriginPath(redirect) ? redirect : undefined }}
      className={cn(
        "font-medium text-foreground underline underline-offset-4",
        className
      )}
    >
      {children}
    </Link>
  )
}
