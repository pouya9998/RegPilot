package repositories

import (
	"time"

	"github.com/regpilot/regpilot/internal/models"
	"gorm.io/gorm"
)

type AuditFilter struct {
	Username   string
	Action     string
	RegistryID *uint
	From       *time.Time
	To         *time.Time
	Limit      int
	Offset     int
}

type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(log *models.AuditLog) error {
	return r.db.Create(log).Error
}

func (r *AuditRepository) List(f AuditFilter) ([]models.AuditLog, int64, error) {
	q := r.db.Model(&models.AuditLog{})
	if f.Username != "" {
		q = q.Where("username = ?", f.Username)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.RegistryID != nil {
		q = q.Where("registry_id = ?", *f.RegistryID)
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at <= ?", *f.To)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var logs []models.AuditLog
	err := q.Order("created_at desc").Limit(limit).Offset(f.Offset).Find(&logs).Error
	return logs, total, err
}

func (r *AuditRepository) Recent(limit int) ([]models.AuditLog, error) {
	if limit <= 0 {
		limit = 10
	}
	var logs []models.AuditLog
	err := r.db.Order("created_at desc").Limit(limit).Find(&logs).Error
	return logs, err
}
