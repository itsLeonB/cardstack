import { Link } from "@tanstack/react-router"
import { PageContainer } from "@/components/layout/page-container"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

const steps = [
  {
    title: "Browse the Catalog",
    body: "Search Indonesian print Pokémon cards by set, rarity and more. No account needed.",
  },
  {
    title: "Add Cards to Collections",
    body: "Group the cards you own into Collections, one for each binder, box or deck.",
  },
  {
    title: "See your Master Inventory",
    body: "Every card you own across all your Collections, in one place.",
  },
]

/** Marketing page shown at `/` to anyone without a session; no screenshots or testimonials on purpose. */
export function GuestLanding() {
  return (
    <PageContainer className="gap-12 sm:gap-16">
      <div className="flex flex-col gap-6">
        <PageHeader
          title="Keep your Pokémon card collection in one place"
          description={
            <p className="text-base text-foreground">
              Track every card you own, across every binder.
            </p>
          }
        />
        <div className="flex flex-col gap-3 sm:flex-row">
          <Button size="lg" render={<Link to="/auth/register" />}>
            Create account
          </Button>
          <Button size="lg" variant="outline" render={<Link to="/catalog" />}>
            Browse the catalog
          </Button>
        </div>
      </div>

      <section className="flex flex-col gap-6" aria-labelledby="how-it-works">
        <h2 id="how-it-works" className="font-heading text-xl font-medium">
          How it works
        </h2>
        <ol className="grid gap-6 sm:grid-cols-3">
          {steps.map((step, index) => (
            <li key={step.title} className="flex flex-col gap-2">
              <span
                aria-hidden="true"
                className="flex size-8 items-center justify-center rounded-full bg-primary font-heading text-sm font-semibold text-primary-foreground"
              >
                {index + 1}
              </span>
              <h3 className="font-heading text-base font-medium">{step.title}</h3>
              <p className="text-sm">{step.body}</p>
            </li>
          ))}
        </ol>
      </section>

      <Card>
        <CardHeader>
          <CardTitle>
            <h2 className="text-xl">Ready to count your cards?</h2>
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-start gap-4">
          <p className="text-sm">
            Create a free account and start your first Collection.
          </p>
          <Button size="lg" render={<Link to="/auth/register" />}>
            Create account
          </Button>
        </CardContent>
      </Card>
    </PageContainer>
  )
}
