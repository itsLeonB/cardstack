import { useState } from "react"
import { RiAddLine, RiSubtractLine } from "@remixicon/react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

const MAX_QUANTITY = 2147483647

/** Minus, number input, plus. Typing keeps its own text so a half-typed value isn't coerced. */
export function QuantityControl({
  cardName,
  value,
  error,
  onChange,
}: {
  cardName: string
  value: number
  error?: string
  onChange: (quantity: number) => void
}) {
  const [draft, setDraft] = useState<string | null>(null)
  // A reverted value must win over stale typed text.
  const shown = draft !== null && (draft === "" || Number(draft) === value) ? draft : value.toString()

  function handleType(text: string) {
    setDraft(text)
    const quantity = Number(text)
    if (text !== "" && Number.isInteger(quantity) && quantity >= 0 && quantity <= MAX_QUANTITY) {
      onChange(quantity)
    }
  }

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-1">
        <Button
          type="button"
          variant="outline"
          size="icon-sm"
          aria-label={`Decrease quantity of ${cardName}`}
          disabled={value <= 0}
          onClick={() => {
            setDraft(null)
            onChange(value - 1)
          }}
        >
          <RiSubtractLine />
        </Button>
        <Input
          type="number"
          min={0}
          max={MAX_QUANTITY}
          step={1}
          inputMode="numeric"
          className="text-center"
          aria-label={`Quantity of ${cardName}`}
          value={shown}
          onChange={(event) => handleType(event.target.value)}
          onBlur={() => setDraft(null)}
        />
        <Button
          type="button"
          variant="outline"
          size="icon-sm"
          aria-label={`Increase quantity of ${cardName}`}
          disabled={value >= MAX_QUANTITY}
          onClick={() => {
            setDraft(null)
            onChange(value + 1)
          }}
        >
          <RiAddLine />
        </Button>
      </div>
      {error && (
        <p role="alert" className="text-xs text-destructive">
          {error}
        </p>
      )}
    </div>
  )
}
