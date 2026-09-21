import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { HealthCheck } from "./health-check"
import { useGetHealth } from "@/generated/endpoints/health/health"

vi.mock("@/generated/endpoints/health/health", () => ({
  useGetHealth: vi.fn(),
}))

const mockUseGetHealth = vi.mocked(useGetHealth)

describe("HealthCheck", () => {
  it("renders the backend status once the health check succeeds", () => {
    mockUseGetHealth.mockReturnValue({
      isPending: false,
      isError: false,
      error: null,
      data: { status: 200, data: { data: { status: "ok" } } },
    } as any)

    render(<HealthCheck />)

    expect(screen.getByText("ok")).toBeTruthy()
  })

  it("renders a pending state while the request is in flight", () => {
    mockUseGetHealth.mockReturnValue({
      isPending: true,
      isError: false,
      error: null,
      data: undefined,
    } as any)

    render(<HealthCheck />)

    expect(screen.getByText("Checking backend status...")).toBeTruthy()
  })

  it("renders the native error message when fetch itself fails", () => {
    mockUseGetHealth.mockReturnValue({
      isPending: false,
      isError: true,
      error: new TypeError("Failed to fetch"),
      data: undefined,
    } as any)

    render(<HealthCheck />)

    expect(
      screen.getByText("Failed to reach backend: Failed to fetch"),
    ).toBeTruthy()
  })
})
