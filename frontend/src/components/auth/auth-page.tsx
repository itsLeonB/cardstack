import { PageContainer } from "@/components/layout/page-container"

/**
 * Body of the login and register pages: narrow and centred, holding Clerk's
 * prebuilt component, which draws its own card, header and switch-form link.
 * The surrounding shell comes from `AuthShell`. Clerk's header is not
 * guaranteed to be a page heading, so the page's `h1` is kept for screen readers.
 */
export function AuthPage({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <PageContainer variant="narrow" className="items-center">
      <h1 className="sr-only">{title}</h1>
      {children}
    </PageContainer>
  )
}
