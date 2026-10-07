import { useCallback, useEffect, useRef, useState } from "react"
import { fitLongSide, guideCrop, UPLOAD_LONG_SIDE } from "./frame-crop"

export type CameraState = "starting" | "ready" | "denied" | "unavailable"

/**
 * Where scan frames come from. The scan screen only sees this boundary, so a
 * test hands it a fake that yields a JPEG without a camera or a canvas. Both
 * methods resolve to the JPEG to upload: cropped to the guide and downsized.
 */
export interface FrameSource {
  camera: CameraState
  /** Ref for the viewfinder's `<video>`; the source attaches its stream. */
  videoRef: (el: HTMLVideoElement | null) => void
  captureCamera: () => Promise<Blob>
  fromFile: (file: File) => Promise<Blob>
}

async function toGuideJpeg(
  image: CanvasImageSource,
  width: number,
  height: number
): Promise<Blob> {
  const crop = guideCrop(width, height)
  const out = fitLongSide(crop.w, crop.h, UPLOAD_LONG_SIDE)
  const canvas = document.createElement("canvas")
  canvas.width = out.w
  canvas.height = out.h
  const context = canvas.getContext("2d")
  if (!context) throw new Error("Canvas is not available")
  context.drawImage(image, crop.x, crop.y, crop.w, crop.h, 0, 0, out.w, out.h)
  return new Promise((resolve, reject) =>
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error("Encoding failed"))),
      "image/jpeg",
      0.9
    )
  )
}

/** The real source: the rear camera when there is one, and any picked photo file. */
export function useCameraFrameSource(): FrameSource {
  const [camera, setCamera] = useState<CameraState>("starting")
  const video = useRef<HTMLVideoElement | null>(null)
  const stream = useRef<MediaStream | null>(null)

  const attach = useCallback(() => {
    if (video.current && stream.current) {
      video.current.srcObject = stream.current
      void video.current.play().catch(() => {})
    }
  }, [])

  useEffect(() => {
    if (!navigator.mediaDevices?.getUserMedia) {
      setCamera("unavailable")
      return
    }
    let stopped = false
    navigator.mediaDevices
      .getUserMedia({ video: { facingMode: "environment" }, audio: false })
      .then((media) => {
        if (stopped) return media.getTracks().forEach((t) => t.stop())
        stream.current = media
        setCamera("ready")
        attach()
      })
      .catch((error: Error) => {
        if (stopped) return
        const denied =
          error.name === "NotAllowedError" || error.name === "SecurityError"
        setCamera(denied ? "denied" : "unavailable")
      })
    return () => {
      stopped = true
      stream.current?.getTracks().forEach((t) => t.stop())
      stream.current = null
    }
  }, [attach])

  return {
    camera,
    videoRef: useCallback(
      (el) => {
        video.current = el
        attach()
      },
      [attach]
    ),
    captureCamera: async () => {
      const el = video.current
      if (!el || !el.videoWidth) throw new Error("The camera is not ready")
      return toGuideJpeg(el, el.videoWidth, el.videoHeight)
    },
    fromFile: async (file) => {
      const bitmap = await createImageBitmap(file)
      try {
        return await toGuideJpeg(bitmap, bitmap.width, bitmap.height)
      } finally {
        bitmap.close()
      }
    },
  }
}
