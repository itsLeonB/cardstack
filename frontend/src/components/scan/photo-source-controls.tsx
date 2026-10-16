import { buttonVariants } from "@/components/ui/button"
import type { CameraState } from "@/lib/frame-source"

/** What to say in place of the viewfinder when there is no camera to use. */
export function cameraMessage(camera: CameraState) {
  return camera === "denied"
    ? "Camera access was denied. Allow it in your browser's site settings, or choose a photo instead."
    : "No camera is available here. Choose a photo instead."
}

/** Takes a photo file as the way in, where there is no camera or the user prefers one. */
export function ChoosePhotoButton({
  onFile,
}: {
  onFile: (file: File) => void
}) {
  return (
    <label
      className={`${buttonVariants({ variant: "outline", size: "lg" })} bg-background text-foreground has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-ring/30`}
    >
      Choose a photo
      <input
        type="file"
        accept="image/*"
        className="sr-only"
        onChange={(event) => {
          const file = event.currentTarget.files?.[0]
          event.currentTarget.value = ""
          if (file) onFile(file)
        }}
      />
    </label>
  )
}
