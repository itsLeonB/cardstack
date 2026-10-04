import { createFileRoute } from "@tanstack/react-router"
import { SignUpPage } from "@/components/auth/sign-up-page"
import { redirectSearchSchema } from "@/lib/route-guard"
import { pageHead } from "@/lib/site"

export const Route = createFileRoute("/auth/register/")({
  head: () => pageHead("Create an account"),
  validateSearch: redirectSearchSchema,
  component: SignUpPage,
})
