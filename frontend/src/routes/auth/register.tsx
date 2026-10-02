import { useState } from "react"
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
import { pageHead } from "@/lib/site"
import { z } from "zod"
import { AuthField } from "@/components/auth/auth-field"
import { AuthPage } from "@/components/auth/auth-page"
import { Button } from "@/components/ui/button"
import { Field, FieldError, FieldGroup } from "@/components/ui/field"
import { redirectSchema } from "@/lib/route-guard"
import { useRegisterMutation } from "@/lib/session"

// Carried through to login so a user sent here from a protected page still
// lands back on it after registering and signing in.
const registerSearchSchema = z.object({ redirect: redirectSchema })

export const Route = createFileRoute("/auth/register")({
  head: () => pageHead("Create an account"),
  validateSearch: registerSearchSchema,
  component: RegisterPage,
})

interface FieldErrors {
  email?: string
  password?: string
  passwordConfirmation?: string
}

function RegisterPage() {
  const { redirect } = Route.useSearch()
  const navigate = useNavigate()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [passwordConfirmation, setPasswordConfirmation] = useState("")
  const [errors, setErrors] = useState<FieldErrors>({})
  const [formError, setFormError] = useState<string | null>(null)
  const registerMutation = useRegisterMutation()

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setFormError(null)

    const nextErrors: FieldErrors = {}
    if (!z.email().safeParse(email).success) {
      nextErrors.email = "Enter a valid email address."
    }
    if (password.length < 8) {
      nextErrors.password = "Use at least 8 characters."
    }
    if (password !== passwordConfirmation) {
      nextErrors.passwordConfirmation = "Passwords do not match."
    }
    setErrors(nextErrors)
    if (Object.keys(nextErrors).length > 0) return

    registerMutation.mutate(
      { data: { email, password, passwordConfirmation } },
      {
        onSuccess: (response) => {
          if (response.status === 201) {
            void navigate({
              to: "/auth/login",
              search: { registered: true, redirect },
            })
            return
          }
          setFormError(response.data.detail ?? "Could not create account.")
        },
        onError: () => {
          setFormError("Could not reach the server. Please try again.")
        },
      }
    )
  }

  return (
    <AuthPage
      title="Create an account"
      description="Register with an email and password to start tracking your Collections."
      footer={
        <>
          Already have an account?{" "}
          <Link
            to="/auth/login"
            search={{ redirect }}
            className="font-medium text-foreground underline underline-offset-4"
          >
            Log in
          </Link>
        </>
      }
    >
      <form
        onSubmit={handleSubmit}
        noValidate
        aria-busy={registerMutation.isPending}
      >
        <FieldGroup>
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
            autoComplete="new-password"
            minLength={8}
            required
            value={password}
            error={errors.password}
            onChange={(event) => setPassword(event.target.value)}
          />
          <AuthField
            id="passwordConfirmation"
            label="Confirm password"
            type="password"
            autoComplete="new-password"
            minLength={8}
            required
            value={passwordConfirmation}
            error={errors.passwordConfirmation}
            onChange={(event) => setPasswordConfirmation(event.target.value)}
          />
          {formError && <FieldError>{formError}</FieldError>}
          <Field>
            <Button type="submit" disabled={registerMutation.isPending}>
              {registerMutation.isPending
                ? "Creating account..."
                : "Create account"}
            </Button>
          </Field>
        </FieldGroup>
      </form>
    </AuthPage>
  )
}
