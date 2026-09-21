package service

import domainservice "github.com/itsLeonB/cardstack/backend/internal/domain/service"

type healthService struct{}

func NewHealthService() domainservice.HealthService {
	return &healthService{}
}

func (h *healthService) Check() domainservice.HealthStatus {
	return domainservice.HealthStatus{Status: "ok"}
}
