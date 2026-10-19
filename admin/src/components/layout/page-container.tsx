import { cn } from "cn"

const widths = {
  narrow: "max-w-sm",
  default: "max-w-4xl",
}

/**
 * Page body wrapper. A `div`, never `main`: the root layout owns the single
 * `main` landmark. `min-w-0` lets long titles truncate/wrap instead of scrolling.
 */
export function PageContainer({
  variant = "default",
  className,
  ...props
}: React.ComponentProps<"div"> & { variant?: keyof typeof widths }) {
  return (
    <div
      className={cn(
        "mx-auto flex w-full min-w-0 flex-col gap-6 px-4 py-6 sm:px-6 sm:py-8",
        widths[variant],
        className
      )}
      {...props}
    />
  )
}
