import { useCallback, useEffect, useRef, useState } from "react"
import { fitLongSide, guideCrop, UPLOAD_LONG_SIDE } from "./frame-crop"
import type { Rect } from "./frame-crop"

export type CameraState = "starting" | "ready" | "denied" | "unavailable"

/** The longest side of a whole photo kept for a Page scan to cut; cells are downsized after. */
const FRAME_LONG_SIDE = 2048

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
  /** The whole frame, uncropped, for a Page scan to cut into cells. */
  captureFrame: () => Promise<Blob>
  frameFromFile: (file: File) => Promise<Blob>
}

/** Crops a frame to `crop` and downsizes it so its long side is at most `limit`, as a JPEG. */
async function toJpeg(
  image: CanvasImageSource,
  crop: Rect,
  limit: number
): Promise<Blob> {
  const out = fitLongSide(crop.w, crop.h, limit)
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

const wholeFrame = (width: number, height: number): Rect => ({
  x: 0,
  y: 0,
  w: width,
  h: height,
})

async function decodeFile(
  file: File,
  crop: (width: number, height: number) => Rect,
  limit: number
): Promise<Blob> {
  const bitmap = await createImageBitmap(file)
  try {
    return await toJpeg(bitmap, crop(bitmap.width, bitmap.height), limit)
  } finally {
    bitmap.close()
  }
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

  const videoFrame = (el: HTMLVideoElement | null) => {
    if (!el || !el.videoWidth) throw new Error("The camera is not ready")
    return el
  }

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
      const el = videoFrame(video.current)
      return toJpeg(
        el,
        guideCrop(el.videoWidth, el.videoHeight),
        UPLOAD_LONG_SIDE
      )
    },
    captureFrame: async () => {
      const el = videoFrame(video.current)
      return toJpeg(
        el,
        wholeFrame(el.videoWidth, el.videoHeight),
        FRAME_LONG_SIDE
      )
    },
    fromFile: (file) => decodeFile(file, guideCrop, UPLOAD_LONG_SIDE),
    frameFromFile: (file) => decodeFile(file, wholeFrame, FRAME_LONG_SIDE),
  }
}
