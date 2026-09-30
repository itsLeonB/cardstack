import type { CardSummary } from "@/generated/models"
import { Input } from "@/components/ui/input"

export function CardLine({ card }: { card: CardSummary }) {
  return (
    <div className="min-w-0 flex-1">
      <p className="truncate text-sm font-medium">{card.name}</p>
      <p className="text-xs text-muted-foreground">
        {card.expansionSet.code} · No. {card.localId} · {card.rarity.name}
      </p>
    </div>
  )
}

export function QuantityInput({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange: (value: string) => void
}) {
  return (
    <Input
      type="number"
      min={1}
      step={1}
      inputMode="numeric"
      className="w-20"
      aria-label={label}
      value={value}
      onChange={(event) => onChange(event.target.value)}
    />
  )
}
