/** The page's single `h1`, with an optional description and an actions slot that stacks below on phones. */
export function PageHeader({
  title,
  description,
  actions,
}: {
  title: React.ReactNode
  description?: React.ReactNode
  actions?: React.ReactNode
}) {
  return (
    <header className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
      <div className="flex min-w-0 flex-col gap-1">
        <h1 className="font-heading text-2xl font-medium break-words">
          {title}
        </h1>
        {description && (
          <div className="text-sm text-muted-foreground">{description}</div>
        )}
      </div>
      {actions && <div className="shrink-0">{actions}</div>}
    </header>
  )
}
