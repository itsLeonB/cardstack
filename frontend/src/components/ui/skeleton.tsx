import { cn } from "cn"

function Skeleton({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="skeleton"
      className={cn(
        "rounded-2xl bg-muted motion-safe:animate-pulse",
        className
      )}
      {...props}
    />
  )
}

export { Skeleton }
