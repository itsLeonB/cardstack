import { useState } from "react"
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router"
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
import { useRegisterMutation } from "@/lib/session"

export const Route = createFileRoute("/register")({
  component: RegisterPage,
})

function RegisterPage() {
  const navigate = useNavigate()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [passwordConfirmation, setPasswordConfirmation] = useState("")
  const [formError, setFormError] = useState<string | null>(null)
  const registerMutation = useRegisterMutation()

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setFormError(null)

    if (password !== passwordConfirmation) {
      setFormError("Passwords do not match.")
      return
    }

    registerMutation.mutate(
      { data: { email, password, passwordConfirmation } },
      {
        onSuccess: (response) => {
          if (response.status === 201) {
            void navigate({ to: "/login", search: { registered: true } })
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
    <div className="flex min-h-svh flex-col items-center justify-center gap-4 p-6">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Create an account</CardTitle>
          <CardDescription>
            Register with an email and password to start tracking your
            Collections.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} noValidate>
            <FieldGroup>
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
                  autoComplete="new-password"
                  minLength={8}
                  required
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="passwordConfirmation">
                  Confirm password
                </FieldLabel>
                <Input
                  id="passwordConfirmation"
                  type="password"
                  autoComplete="new-password"
                  minLength={8}
                  required
                  value={passwordConfirmation}
                  onChange={(event) =>
                    setPasswordConfirmation(event.target.value)
                  }
                />
              </Field>
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
        </CardContent>
      </Card>
      <p className="text-sm text-muted-foreground">
        Already have an account?{" "}
        <Link to="/login" className="text-primary underline">
          Log in
        </Link>
      </p>
    </div>
  )
}
