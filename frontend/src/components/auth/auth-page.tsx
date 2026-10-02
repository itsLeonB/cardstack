import { BrandMark } from "@/components/layout/wordmark"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Card, CardContent } from "@/components/ui/card"

/** Shared login/register frame: narrow centered card with the brand mark, the page `h1`, and a switch-form link below. */
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
    <PageContainer variant="narrow" className="min-h-[70svh] items-center justify-center gap-4">
      <Card className="w-full">
        <CardContent className="flex flex-col gap-6">
          <BrandMark className="size-10" />
          <PageHeader title={title} description={description} />
          {children}
        </CardContent>
      </Card>
      <p className="text-sm">{footer}</p>
    </PageContainer>
  )
}
