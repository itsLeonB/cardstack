import { createFileRoute } from "@tanstack/react-router"
import { SignInPage } from "@/components/auth/sign-in-page"
import { redirectSearchSchema } from "@/lib/route-guard"
import { pageHead } from "@/lib/site"

export const Route = createFileRoute("/auth/login/")({
  head: () => pageHead("Log in"),
  validateSearch: redirectSearchSchema,
  component: SignInPage,
})
