import { useState } from "react"
import { RiEyeLine, RiEyeOffLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import { Field, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

/**
 * Labelled input with its field-level error tied to it (`aria-describedby`,
 * `aria-invalid`); the error is a live `role="alert"`, so it is announced when
 * it appears. `type="password"` adds a show/hide toggle.
 */
export function AuthField({
  id,
  label,
  error,
  type = "text",
  ...props
}: Omit<React.ComponentProps<"input">, "id"> & {
  id: string
  label: string
  error?: string
}) {
  const [revealed, setRevealed] = useState(false)
  const isPassword = type === "password"
  const errorId = `${id}-error`

  const input = (
    <Input
      id={id}
      type={isPassword && revealed ? "text" : type}
      aria-invalid={error ? true : undefined}
      aria-describedby={error ? errorId : undefined}
      className={isPassword ? "pr-11" : undefined}
      {...props}
    />
  )

  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      {isPassword ? (
        <div className="relative">
          {input}
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            className="absolute top-1/2 right-1 -translate-y-1/2"
            aria-label={`Show ${label.toLowerCase()}`}
            aria-pressed={revealed}
            onClick={() => setRevealed((value) => !value)}
          >
            {revealed ? <RiEyeOffLine aria-hidden="true" /> : <RiEyeLine aria-hidden="true" />}
          </Button>
        </div>
      ) : (
        input
      )}
      {error && <FieldError id={errorId}>{error}</FieldError>}
    </Field>
  )
}
