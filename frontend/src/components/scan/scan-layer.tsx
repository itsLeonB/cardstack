import type { ReactNode } from "react"
import { cn } from "cn"
import { Button } from "@/components/ui/button"
import { CARD_ASPECT, GUIDE_HEIGHT, VIEW_ASPECT } from "@/lib/frame-crop"
import type { FrameSource } from "@/lib/frame-source"
import { useScanLayer } from "@/lib/scan-layer"
import { cameraMessage, ChoosePhotoButton } from "./photo-source-controls"

/**
 * The full-screen layer both scan pages sit in: it replaces the page below the
 * site header, clears the device safe area, and hides the mobile bar.
 */
export function ScanLayer({
  children,
  scroll = false,
}: {
  children: ReactNode
  /** Scroll the layer instead of fitting its content to the screen. */
  scroll?: boolean
}) {
  useScanLayer()
  return (
    <div
      className={cn(
        "fixed inset-x-0 top-[calc(3.5rem+1px)] bottom-0 z-30 bg-background pb-[env(safe-area-inset-bottom,0px)]",
        scroll && "overflow-y-auto"
      )}
    >
      {children}
    </div>
  )
}

/** The title and the way back, at the top of every scan page. */
export function ScanHeader({
  title,
  back,
  className,
}: {
  title: ReactNode
  back?: ReactNode
  className?: string
}) {
  return (
    <div className={cn("flex items-center justify-between gap-2", className)}>
      <h2 className="text-lg font-medium">{title}</h2>
      {back}
    </div>
  )
}

/**
 * The camera step both scan pages share: the viewfinder, the message when there
 * is no camera, and the Capture and Choose a photo controls.
 */
export function ScanCamera({
  source,
  onCapture,
  onFile,
  error,
  guide = false,
  contain = false,
  overlay = false,
}: {
  source: FrameSource
  onCapture: () => void
  onFile: (file: File) => void
  /** A capture failure, shown over the viewfinder or above it. */
  error?: string | null
  /** Draw the card-shaped guide over the viewfinder. */
  guide?: boolean
  /** Show the whole frame instead of cropping it to fill. */
  contain?: boolean
  /** Put the controls over the viewfinder instead of below it. */
  overlay?: boolean
}) {
  const hasCamera = source.camera === "starting" || source.camera === "ready"
  return (
    <div
      className={
        overlay
          ? "relative min-h-0 flex-1 overflow-hidden bg-black text-white"
          : "flex flex-col gap-3"
      }
    >
      {error && !overlay && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
      {hasCamera ? (
        <div
          className={cn(
            "relative w-full overflow-hidden",
            !overlay && "bg-black"
          )}
          style={{ aspectRatio: VIEW_ASPECT }}
        >
          <video
            ref={source.videoRef}
            playsInline
            muted
            aria-label="Camera viewfinder"
            className={cn(
              "size-full",
              contain ? "object-contain" : "object-cover"
            )}
          />
          {guide && (
            <div
              aria-hidden
              className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 rounded-xl border-2 border-white/90 shadow-[0_0_0_9999px_rgb(0_0_0/0.45)]"
              style={{
                height: `${GUIDE_HEIGHT * 100}%`,
                aspectRatio: CARD_ASPECT,
              }}
            />
          )}
        </div>
      ) : (
        <p
          role="alert"
          className={overlay ? "px-4 pt-20 text-center text-sm" : "text-sm"}
        >
          {cameraMessage(source.camera)}
        </p>
      )}
      {error && overlay && (
        <p
          role="alert"
          className="absolute inset-x-3 top-3 rounded-lg bg-background p-2 text-sm text-destructive"
        >
          {error}
        </p>
      )}
      <div
        className={
          overlay
            ? "absolute inset-x-0 bottom-3 flex flex-wrap justify-center gap-2 px-4"
            : "flex flex-wrap gap-2"
        }
      >
        {hasCamera && (
          <Button
            size="lg"
            disabled={source.camera !== "ready"}
            onClick={onCapture}
          >
            Capture
          </Button>
        )}
        <ChoosePhotoButton onFile={onFile} />
      </div>
    </div>
  )
}
