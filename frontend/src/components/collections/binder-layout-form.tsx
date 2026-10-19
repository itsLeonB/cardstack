import { useState } from "react"
import { SELECT_CLASS } from "@/components/admin/form-shared"
import { Button } from "@/components/ui/button"
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { BINDER_PRESETS, MAX_GRID, MAX_LEAVES, presetIdFor } from "@/lib/binder"

export interface BinderLayoutValues {
  rows: number
  columns: number
  /** Only sent when the form asks for it, on a new binder. */
  leafCount?: number
}

interface BinderLayoutFormProps {
  initial?: { rows: number; columns: number }
  /** Only a new binder chooses its starting Leaf count. */
  withLeafCount: boolean
  submitLabel: string
  pendingLabel: string
  isPending: boolean
  errorMessage?: string | null
  onCancel?: () => void
  onSubmit: (values: BinderLayoutValues) => void
}

function parseWhole(text: string, min: number, max: number) {
  const trimmed = text.trim()
  const parsed = Number(trimmed)
  return trimmed !== "" &&
    Number.isInteger(parsed) &&
    parsed >= min &&
    parsed <= max
    ? parsed
    : null
}

export function BinderLayoutForm({
  initial,
  withLeafCount,
  submitLabel,
  pendingLabel,
  isPending,
  errorMessage,
  onCancel,
  onSubmit,
}: BinderLayoutFormProps) {
  const [rows, setRows] = useState(String(initial?.rows ?? 3))
  const [columns, setColumns] = useState(String(initial?.columns ?? 3))
  const [leafCount, setLeafCount] = useState("10")
  const [validationError, setValidationError] = useState<string | null>(null)

  function choosePreset(id: string) {
    const preset = BINDER_PRESETS.find((p) => p.id === id)
    if (!preset) return
    setRows(String(preset.rows))
    setColumns(String(preset.columns))
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setValidationError(null)

    const parsedRows = parseWhole(rows, 1, MAX_GRID)
    const parsedColumns = parseWhole(columns, 1, MAX_GRID)
    if (parsedRows === null || parsedColumns === null) {
      setValidationError("Rows and columns must each be from 1 to 6.")
      return
    }
    if (!withLeafCount) {
      onSubmit({ rows: parsedRows, columns: parsedColumns })
      return
    }
    const parsedLeaves = parseWhole(leafCount, 1, MAX_LEAVES)
    if (parsedLeaves === null) {
      setValidationError("A binder holds from 1 to 100 Leaves.")
      return
    }
    onSubmit({
      rows: parsedRows,
      columns: parsedColumns,
      leafCount: parsedLeaves,
    })
  }

  return (
    <form onSubmit={handleSubmit} noValidate>
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="binder-preset">Layout</FieldLabel>
          <select
            id="binder-preset"
            className={SELECT_CLASS}
            value={presetIdFor(Number(rows), Number(columns))}
            onChange={(event) => choosePreset(event.target.value)}
          >
            {BINDER_PRESETS.map((preset) => (
              <option key={preset.id} value={preset.id}>
                {preset.label} (rows × columns)
              </option>
            ))}
            <option value="custom">Custom</option>
          </select>
        </Field>
        <Field>
          <FieldLabel htmlFor="binder-rows">Rows per Page</FieldLabel>
          <Input
            id="binder-rows"
            type="number"
            min={1}
            max={MAX_GRID}
            step={1}
            inputMode="numeric"
            value={rows}
            onChange={(event) => setRows(event.target.value)}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="binder-columns">Columns per Page</FieldLabel>
          <Input
            id="binder-columns"
            type="number"
            min={1}
            max={MAX_GRID}
            step={1}
            inputMode="numeric"
            value={columns}
            onChange={(event) => setColumns(event.target.value)}
          />
        </Field>
        {withLeafCount && (
          <Field>
            <FieldLabel htmlFor="binder-leaves">Leaves</FieldLabel>
            <Input
              id="binder-leaves"
              type="number"
              min={1}
              max={MAX_LEAVES}
              step={1}
              inputMode="numeric"
              value={leafCount}
              onChange={(event) => setLeafCount(event.target.value)}
            />
          </Field>
        )}
        {(validationError ?? errorMessage) && (
          <FieldError>{validationError ?? errorMessage}</FieldError>
        )}
        <Field orientation="horizontal">
          <Button type="submit" disabled={isPending}>
            {isPending ? pendingLabel : submitLabel}
          </Button>
          {onCancel && (
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
          )}
        </Field>
      </FieldGroup>
    </form>
  )
}
