package services

import (
	"github.com/regpilot/regpilot/internal/models"
	"github.com/regpilot/regpilot/internal/repositories"
)

type AuditService struct {
	audit *repositories.AuditRepository
}

func NewAuditService(audit *repositories.AuditRepository) *AuditService {
	return &AuditService{audit: audit}
}

func (s *AuditService) List(f repositories.AuditFilter) ([]models.AuditLog, int64, error) {
	return s.audit.List(f)
}
