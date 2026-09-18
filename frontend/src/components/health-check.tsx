import { useGetHealth } from "@/generated/endpoints/health/health"

export function HealthCheck() {
  const { data, isPending, isError, error } = useGetHealth()

  return (
    <div className="flex min-h-svh p-6">
      <div className="flex max-w-md min-w-0 flex-col gap-4 text-sm leading-loose">
        <h1 className="font-medium">Backend health check</h1>
        {isPending && <p>Checking backend status...</p>}
        {isError && <p>Failed to reach backend: {error.detail}</p>}
        {data && data.status === 200 && (
          // data.data.data: outer .data is the fetch wrapper ({status, data}),
          // inner .data is the backend's Envelope[HealthStatus] response body.
          <p>
            Backend status: <strong>{data.data.data.status}</strong>
          </p>
        )}
        {data && data.status !== 200 && (
          <p>Backend returned an error: {data.data.detail}</p>
        )}
      </div>
    </div>
  )
}
