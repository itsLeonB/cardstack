import { useState } from "react"
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useLoginMutation } from "@/lib/session"

// `redirect` only ever needs to point back into this app (requireAuth sets
// it from the router's own same-origin location.href), so it's restricted
// to a same-origin relative path here. Left unvalidated, a crafted
// `/login?redirect=` link could send a successful login to an attacker
// controlled destination (open redirect).
const loginSearchSchema = z.object({
  redirect: z
    .string()
    .refine((path) => path.startsWith("/") && !path.startsWith("//"))
    .optional(),
  registered: z.boolean().optional(),
})

export const Route = createFileRoute("/login")({
  validateSearch: loginSearchSchema,
  component: LoginPage,
})

function LoginPage() {
  const { redirect, registered } = Route.useSearch()
  const navigate = useNavigate()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [formError, setFormError] = useState<string | null>(null)
  const loginMutation = useLoginMutation()

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setFormError(null)

    loginMutation.mutate(
      { data: { email, password } },
      {
        onSuccess: (response) => {
          if (response.status === 200) {
            // SAFETY: redundant with loginSearchSchema's refine, kept here
            // so a post-login redirect is never sent off-site even if
            // validateSearch's own enforcement ever changes.
            const isSameOriginPath =
              redirect?.startsWith("/") && !redirect.startsWith("//")
            void navigate({ to: isSameOriginPath ? redirect : "/account" })
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
    <div className="flex min-h-svh flex-col items-center justify-center gap-4 p-6">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Log in</CardTitle>
          <CardDescription>
            Log in with your email and password to access your Collections.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} noValidate>
            <FieldGroup>
              {registered && (
                <p className="text-sm text-muted-foreground">
                  Account created. Log in below.
                </p>
              )}
              <Field>
                <FieldLabel htmlFor="email">Email</FieldLabel>
                <Input
                  id="email"
                  type="email"
                  autoComplete="email"
                  required
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="password">Password</FieldLabel>
                <Input
                  id="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </Field>
              {formError && <FieldError>{formError}</FieldError>}
              <Field>
                <Button type="submit" disabled={loginMutation.isPending}>
                  {loginMutation.isPending ? "Logging in..." : "Log in"}
                </Button>
              </Field>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
      <p className="text-sm text-muted-foreground">
        Don&apos;t have an account?{" "}
        <Link to="/register" className="text-primary underline">
          Register
        </Link>
      </p>
    </div>
  )
}
