package registry

import "errors"

var (
	ErrUnauthorized = errors.New("registry unauthorized")
	ErrUnreachable  = errors.New("registry unreachable")
	ErrNotFound     = errors.New("not found")
	ErrUnhealthy    = errors.New("registry unhealthy")
)

type HealthStatus string

const (
	HealthHealthy      HealthStatus = "healthy"
	HealthUnhealthy    HealthStatus = "unhealthy"
	HealthUnauthorized HealthStatus = "unauthorized"
	HealthUnreachable  HealthStatus = "unreachable"
)

type HealthResult struct {
	Status    HealthStatus `json:"status"`
	LatencyMs int64        `json:"latency_ms"`
	Message   string       `json:"message,omitempty"`
}
