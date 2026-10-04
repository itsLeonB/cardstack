import { PageContainer } from "@/components/layout/page-container"

/**
 * Body of the login and register pages: narrow and centred, holding Clerk's
 * prebuilt component, which draws its own card, header and switch-form link.
 * The surrounding shell comes from `AuthShell`. Clerk's header title is the
 * page's `h1` (on every step), so this adds none: two would break the one-`h1`
 * rule.
 */
export function AuthPage({ children }: { children: React.ReactNode }) {
  return (
    <PageContainer variant="narrow" className="items-center">
      {children}
    </PageContainer>
  )
}
