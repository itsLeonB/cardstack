package service

// HealthStatus is the response body of the health check endpoint.
type HealthStatus struct {
	Status string `json:"status"`
}

// HealthService reports whether the API is up. It has no dependencies today
// (no DB ping, no downstream checks) — that can grow once there's something
// worth checking.
type HealthService interface {
	Check() HealthStatus
}
