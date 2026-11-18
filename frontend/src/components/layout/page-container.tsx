import { cn } from "cn"

const widths = {
  narrow: "max-w-sm",
  default: "max-w-4xl",
  wide: "max-w-6xl",
}

/**
 * Page body wrapper. A `div`, never `main`: each shell (`AppShell`, `AuthShell`)
 * owns its single `main` landmark. `min-w-0` lets long titles truncate/wrap instead of scrolling.
 */
export function PageContainer({
  variant = "default",
  className,
  ...props
}: React.ComponentProps<"div"> & { variant?: keyof typeof widths }) {
  return (
    <div
      className={cn(
        "mx-auto flex w-full min-w-0 flex-col gap-6 py-6 pr-[max(1rem,env(safe-area-inset-right,0px))] pl-[max(1rem,env(safe-area-inset-left,0px))] sm:py-8 sm:pr-[max(1.5rem,env(safe-area-inset-right,0px))] sm:pl-[max(1.5rem,env(safe-area-inset-left,0px))]",
        widths[variant],
        className
      )}
      {...props}
    />
  )
}
