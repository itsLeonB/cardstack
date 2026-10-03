import { useState } from "react"
import { RiArrowDownSLine } from "@remixicon/react"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import { cn } from "cn"

export interface MultiSelectOption {
  value: string
  label: string
  available?: boolean
}

export interface MultiSelectGroup {
  /** Omit for a flat, unlabelled list. */
  label?: string
  options: MultiSelectOption[]
}

interface MultiSelectProps {
  label: string
  groups: MultiSelectGroup[]
  selected: string[]
  onChange: (values: string[]) => void
  searchable?: boolean
}

/**
 * Dropdown of checkboxes that stays open while toggling. Native checkboxes
 * inside a popover keep Tab/Space/Escape keyboard handling for free. The
 * popover only mounts when open, so tests and e2e must click the trigger first.
 */
export function MultiSelect({
  label,
  groups,
  selected,
  onChange,
  searchable,
}: MultiSelectProps) {
  const [query, setQuery] = useState("")
  const needle = query.trim().toLowerCase()
  // Selected options always stay listed so they can be unchecked.
  const visible = (option: MultiSelectOption) =>
    selected.includes(option.value) ||
    option.label.toLowerCase().includes(needle)
  const shown = groups.map((group) => ({
    ...group,
    options: group.options.filter(visible),
  }))
  const noMatches = shown.every((group) => group.options.length === 0)

  return (
    <Popover onOpenChange={() => setQuery("")}>
      <PopoverTrigger className="inline-flex h-9 w-full items-center justify-between gap-1.5 rounded-3xl border border-transparent bg-input/50 px-3 py-2 text-sm outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30 sm:w-fit">
        {label}
        {selected.length > 0 && (
          <Badge>
            <span className="sr-only">selected </span>
            {selected.length}
          </Badge>
        )}
        <RiArrowDownSLine
          aria-hidden
          className="size-4 text-muted-foreground"
        />
      </PopoverTrigger>
      <PopoverContent>
        {searchable && (
          <Input
            type="search"
            aria-label={`Search ${label}`}
            placeholder={`Search ${label}`}
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            className="mb-2"
          />
        )}
        <div className="flex flex-col gap-3">
          {noMatches && (
            <p className="text-sm text-muted-foreground">No matches</p>
          )}
          {shown.map((group) => {
            const options = group.options
            if (options.length === 0) return null
            return (
              <fieldset
                key={group.label ?? ""}
                className="flex min-w-0 flex-col gap-1.5"
              >
                <legend
                  className={
                    group.label
                      ? "mb-1 text-xs text-muted-foreground"
                      : "sr-only"
                  }
                >
                  {group.label ?? label}
                </legend>
                {options.map((option) => {
                  const checked = selected.includes(option.value)
                  return (
                    <label
                      key={option.value}
                      className={cn(
                        "flex items-center gap-2 text-sm",
                        option.available === false && "text-muted-foreground"
                      )}
                    >
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() =>
                          onChange(
                            checked
                              ? selected.filter(
                                  (value) => value !== option.value
                                )
                              : [...selected, option.value]
                          )
                        }
                      />
                      {option.label}
                      {option.available === false && (
                        <span className="sr-only"> (unavailable)</span>
                      )}
                    </label>
                  )
                })}
              </fieldset>
            )
          })}
        </div>
      </PopoverContent>
    </Popover>
  )
}
