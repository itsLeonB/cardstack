import { useState } from "react"
import { createFileRoute, Link, useRouter } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { z } from "zod"
import { AuthField } from "@/components/auth/auth-field"
import { AuthPage } from "@/components/auth/auth-page"
import { Button } from "@/components/ui/button"
import { Field, FieldError, FieldGroup } from "@/components/ui/field"
import { isSameOriginPath, redirectSchema } from "@/lib/route-guard"
import { useLoginMutation } from "@/lib/session"

// `redirect` only ever needs to point back into this app (requireAuth sets
// it from the router's own location path and query), so it's restricted to
// a same-origin relative path here. Left unvalidated, a crafted
// `/auth/login?redirect=` link could send a successful login to an attacker
// controlled destination (open redirect).
const loginSearchSchema = z.object({
  redirect: redirectSchema,
  registered: z.boolean().optional(),
})

export const Route = createFileRoute("/auth/login")({
  head: () => pageHead("Log in"),
  validateSearch: loginSearchSchema,
  component: LoginPage,
})

interface FieldErrors {
  email?: string
  password?: string
}

function LoginPage() {
  const { redirect, registered } = Route.useSearch()
  const router = useRouter()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [errors, setErrors] = useState<FieldErrors>({})
  const [formError, setFormError] = useState<string | null>(null)
  const loginMutation = useLoginMutation()

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setFormError(null)

    const nextErrors: FieldErrors = {}
    if (!z.email().safeParse(email).success) {
      nextErrors.email = "Enter a valid email address."
    }
    if (!password) nextErrors.password = "Enter your password."
    setErrors(nextErrors)
    if (nextErrors.email || nextErrors.password) return

    loginMutation.mutate(
      { data: { email, password } },
      {
        onSuccess: (response) => {
          if (response.status === 200) {
            // SAFETY: redundant with loginSearchSchema's refine, kept here
            // so a post-login redirect is never sent off-site even if
            // validateSearch's own enforcement ever changes. `history.push`
            // (not `navigate({ to })`) because the target carries a query string.
            router.history.push(isSameOriginPath(redirect) ? redirect : "/account")
            return
          }
          setFormError(response.data.detail ?? "Invalid email or password.")
        },
        onError: () => {
          setFormError("Could not reach the server. Please try again.")
        },
      }
    )
  }

  return (
    <AuthPage
      title="Log in"
      description="Log in with your email and password to access your Collections."
      footer={
        <>
          Don&apos;t have an account?{" "}
          <Link to="/auth/register" search={{ redirect }} className="font-medium text-foreground underline underline-offset-4">
            Register
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit} noValidate aria-busy={loginMutation.isPending}>
        <FieldGroup>
          {registered && (
            <p role="status" className="rounded-2xl bg-muted px-4 py-3 text-sm text-foreground">
              Account created. Log in below.
            </p>
          )}
          <AuthField
            id="email"
            label="Email"
            type="email"
            autoComplete="email"
            required
            value={email}
            error={errors.email}
            onChange={(event) => setEmail(event.target.value)}
          />
          <AuthField
            id="password"
            label="Password"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            error={errors.password}
            onChange={(event) => setPassword(event.target.value)}
          />
          {formError && <FieldError>{formError}</FieldError>}
          <Field>
            <Button type="submit" disabled={loginMutation.isPending}>
              {loginMutation.isPending ? "Logging in..." : "Log in"}
            </Button>
          </Field>
        </FieldGroup>
      </form>
    </AuthPage>
  )
}
