import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { HealthCheck } from "./health-check"
import { useGetHealth } from "@/generated/endpoints/health/health"

// useGetHealth is a generated orval/TanStack Query hook with no service layer to
// inject; mocking the generated module is the standard way to isolate components
// from it in tests.
// oxlint-disable-next-line anti-slop/no-module-mocking
vi.mock("@/generated/endpoints/health/health", () => ({
  useGetHealth: vi.fn(),
}))

const mockUseGetHealth = vi.mocked(useGetHealth)

describe("HealthCheck", () => {
  it("renders the backend status once the health check succeeds", () => {
    // SAFETY: partial mock covering only the fields HealthCheck reads
    // (isPending, isError, error, data); the real hook return has more.
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
    // SAFETY: partial mock covering only the fields HealthCheck reads
    // (isPending, isError, error, data); the real hook return has more.
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
    // SAFETY: partial mock covering only the fields HealthCheck reads
    // (isPending, isError, error, data); the real hook return has more.
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
