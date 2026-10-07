import { useRef, useState } from "react"
import { toast } from "sonner"
import { Button, buttonVariants } from "@/components/ui/button"
import { DraftTray } from "./draft-tray"
import { matchScannedCard } from "@/generated/endpoints/scan/scan"
import type { CardSummary, MatchCandidate } from "@/generated/models"
import type { FrameSource } from "@/lib/frame-source"
import { imageSources } from "@/lib/image"
import { useDraftAddition } from "@/lib/use-draft-addition"

// A match that takes longer than this is reported as failed.
const MATCH_TIMEOUT_MS = 20_000

type Outcome =
  | { kind: "added"; card: CardSummary }
  | { kind: "candidates"; candidates: MatchCandidate[] }
  | { kind: "none" }

const DRAFT_KEPT = "Your draft is unchanged."

function CardFace({ card }: { card: CardSummary }) {
  return (
    <>
      {card.imageUrl ? (
        <img
          src={imageSources(card.imageUrl, "cardTile").src}
          alt=""
          width={48}
          height={67}
          className="aspect-[5/7] w-12 rounded object-cover"
        />
      ) : (
        <div className="aspect-[5/7] w-12 rounded bg-muted" />
      )}
      <span className="flex min-w-0 flex-col text-left">
        <span className="truncate text-sm font-medium">{card.name}</span>
        <span className="text-xs">
          {card.expansionSet.name} · No. {card.localId}
        </span>
      </span>
    </>
  )
}

/**
 * The scan screen of one Collection: capture (camera or photo file), match,
 * then the Draft Addition. The frame source is injected so tests can fake the
 * camera. Remount per Collection (the route keys it) so drafts never mix.
 */
export function ScanScreen({
  collectionId,
  source,
}: {
  collectionId: string
  source: FrameSource
}) {
  const draft = useDraftAddition(collectionId)
  const [outcome, setOutcome] = useState<Outcome | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  // Cards matched this visit, so the tray can show them before its lookup answers.
  const known = useRef(new Map<string, CardSummary>())

  function addCard(card: CardSummary) {
    known.current.set(card.id, card)
    draft.add(card.id)
    setOutcome({ kind: "added", card })
  }

  async function scan(takeFrame: () => Promise<Blob>) {
    setBusy(true)
    setError(null)
    setOutcome(null)
    try {
      let frame: Blob
      try {
        frame = await takeFrame()
      } catch {
        setError(`Could not capture that photo. ${DRAFT_KEPT}`)
        return
      }
      const response = await matchScannedCard(frame, {
        signal: AbortSignal.timeout(MATCH_TIMEOUT_MS),
      })
      if (response.status !== 200) throw new Error("match failed")
      const { confident, candidates } = response.data.data
      if (confident && candidates?.[0]) addCard(candidates[0].card)
      else if (candidates?.length)
        setOutcome({ kind: "candidates", candidates })
      else setOutcome({ kind: "none" })
    } catch {
      setError(`Could not match this photo. Try again. ${DRAFT_KEPT}`)
    } finally {
      setBusy(false)
    }
  }

  const hasCamera = source.camera === "starting" || source.camera === "ready"

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-3">
        {hasCamera ? (
          // The 3:4 box and the 85%-high guide are what `lib/frame-crop.ts` crops to.
          <div className="relative mx-auto aspect-[3/4] w-full max-w-sm overflow-hidden rounded-2xl bg-black">
            <video
              ref={source.videoRef}
              playsInline
              muted
              aria-label="Camera viewfinder"
              className="size-full object-cover"
            />
            <div
              aria-hidden
              className="absolute top-1/2 left-1/2 h-[85%] -translate-x-1/2 -translate-y-1/2 rounded-xl border-2 border-white/90 shadow-[0_0_0_9999px_rgb(0_0_0/0.45)]"
              style={{ aspectRatio: "63 / 88" }}
            />
          </div>
        ) : (
          <p role="alert" className="text-sm text-destructive">
            {source.camera === "denied"
              ? "Camera access was denied. Allow it in your browser's site settings, or choose a photo instead."
              : "No camera is available here. Choose a photo instead."}
          </p>
        )}
        <div className="flex flex-wrap justify-center gap-2">
          {hasCamera && (
            <Button
              disabled={busy || source.camera !== "ready"}
              onClick={() => void scan(source.captureCamera)}
            >
              Capture
            </Button>
          )}
          <label
            className={`${buttonVariants({ variant: "outline" })} has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-ring/30`}
          >
            Choose a photo
            <input
              type="file"
              accept="image/*"
              disabled={busy}
              className="sr-only"
              onChange={(event) => {
                const file = event.currentTarget.files?.[0]
                event.currentTarget.value = ""
                if (file) void scan(() => source.fromFile(file))
              }}
            />
          </label>
        </div>
      </div>

      {busy && <p className="text-sm">Matching…</p>}
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
      <div role="status">
        {outcome?.kind === "added" && (
          <div className="flex items-center gap-3 rounded-xl border p-2">
            <CardFace card={outcome.card} />
            <p className="sr-only">Added {outcome.card.name}</p>
            <Button
              variant="outline"
              size="sm"
              className="ml-auto"
              onClick={() => {
                draft.undo(outcome.card.id)
                setOutcome(null)
              }}
            >
              Undo
            </Button>
          </div>
        )}
        {outcome?.kind === "none" && (
          <p className="text-sm">No card matched. Try another photo.</p>
        )}
      </div>
      {outcome?.kind === "candidates" && (
        <section
          aria-labelledby="candidates-heading"
          className="flex flex-col gap-2"
        >
          <h2
            id="candidates-heading"
            className="font-heading text-lg font-medium"
          >
            Which card is it?
          </h2>
          {outcome.candidates.map(({ card }) => (
            <button
              key={card.id}
              type="button"
              aria-label={`Add ${card.name} (${card.expansionSet.code}, No. ${card.localId})`}
              className="flex items-center gap-3 rounded-xl border p-2 hover:bg-muted"
              onClick={() => addCard(card)}
            >
              <CardFace card={card} />
            </button>
          ))}
          <Button variant="outline" onClick={() => setOutcome(null)}>
            Skip
          </Button>
        </section>
      )}

      <DraftTray
        rows={draft.rows}
        known={known.current}
        onRaise={draft.add}
        onLower={draft.lower}
        onRemove={draft.remove}
        onDiscard={draft.discard}
        // Placeholder: ticket 04 replaces it with the review step.
        onReview={() => toast.info("Reviewing a draft is coming soon.")}
      />
    </div>
  )
}
