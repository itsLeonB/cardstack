import { renderHook, waitFor } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"
import { useCameraFrameSource } from "./frame-source"

// SAFETY: partial fixture; the code under test reads only `deviceId` and `label`.
const device = (deviceId: string, label: string) =>
  ({ deviceId, label, kind: "videoinput" }) as MediaDeviceInfo

function fakeStream(deviceId: string) {
  const track = {
    stop: vi.fn(),
    getSettings: () => ({ deviceId }),
  }
  return {
    track,
    stream: { getTracks: () => [track], getVideoTracks: () => [track] },
  }
}

function fakeCamera(
  devices: MediaDeviceInfo[],
  ...opens: Array<ReturnType<typeof fakeStream> | Error>
) {
  const getUserMedia = vi.fn(async () => {
    const next = opens.shift()
    if (!next) throw new Error("unexpected getUserMedia call")
    if (next instanceof Error) throw next
    return next.stream
  })
  Object.defineProperty(navigator, "mediaDevices", {
    configurable: true,
    value: { getUserMedia, enumerateDevices: async () => devices },
  })
  return getUserMedia
}

// Samsung lists the ultrawide as camera2 2 and the 1x main as camera2 0.
const samsung = [
  device("main", "camera2 0, facing back"),
  device("front", "camera2 1, facing front"),
  device("ultra", "camera2 2, facing back"),
]

afterEach(() => {
  Reflect.deleteProperty(navigator, "mediaDevices")
})

describe("useCameraFrameSource", () => {
  it("reopens the main back camera when the browser handed over the ultrawide", async () => {
    const ultra = fakeStream("ultra")
    const main = fakeStream("main")
    const getUserMedia = fakeCamera(samsung, ultra, main)

    const { result } = renderHook(() => useCameraFrameSource())

    await waitFor(() => expect(getUserMedia).toHaveBeenCalledTimes(2))
    expect(getUserMedia).toHaveBeenLastCalledWith({
      video: { deviceId: { exact: "main" } },
      audio: false,
    })
    expect(ultra.track.stop).toHaveBeenCalled()
    await waitFor(() => expect(result.current.camera).toBe("ready"))
  })

  it("keeps the first stream when it is already the main camera", async () => {
    const main = fakeStream("main")
    const getUserMedia = fakeCamera(samsung, main)

    const { result } = renderHook(() => useCameraFrameSource())

    await waitFor(() => expect(result.current.camera).toBe("ready"))
    expect(getUserMedia).toHaveBeenCalledTimes(1)
    expect(main.track.stop).not.toHaveBeenCalled()
  })

  it("keeps the facing-mode stream when no device label says facing back", async () => {
    const webcam = fakeStream("cam")
    const getUserMedia = fakeCamera(
      [device("cam", "Integrated Camera")],
      webcam
    )

    const { result } = renderHook(() => useCameraFrameSource())

    await waitFor(() => expect(result.current.camera).toBe("ready"))
    expect(getUserMedia).toHaveBeenCalledTimes(1)
  })

  it("falls back to the facing-mode camera when the main camera will not open", async () => {
    const getUserMedia = fakeCamera(
      samsung,
      fakeStream("ultra"),
      new Error("busy"),
      fakeStream("ultra")
    )

    const { result } = renderHook(() => useCameraFrameSource())

    await waitFor(() => expect(result.current.camera).toBe("ready"))
    expect(getUserMedia).toHaveBeenLastCalledWith({
      video: { facingMode: "environment" },
      audio: false,
    })
  })

  it("reports denied when permission is refused", async () => {
    const refused = new Error("no")
    refused.name = "NotAllowedError"
    fakeCamera(samsung, refused)

    const { result } = renderHook(() => useCameraFrameSource())

    await waitFor(() => expect(result.current.camera).toBe("denied"))
  })
})
