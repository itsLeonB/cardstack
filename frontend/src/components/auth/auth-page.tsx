import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Card, CardContent } from "@/components/ui/card"

/** Shared login/register card: narrow, with the page `h1` and a switch-form link below. The surrounding shell comes from `AuthShell`. */
export function AuthPage({
  title,
  description,
  footer,
  children,
}: {
  title: string
  description: string
  footer: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <PageContainer variant="narrow" className="items-center gap-4">
      <Card className="w-full">
        <CardContent className="flex flex-col gap-6">
          <PageHeader title={title} description={description} />
          {children}
        </CardContent>
      </Card>
      <p className="text-sm">{footer}</p>
    </PageContainer>
  )
}
