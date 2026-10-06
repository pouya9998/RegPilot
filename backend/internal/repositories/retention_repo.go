package repositories

import (
	"github.com/regpilot/regpilot/internal/models"
	"gorm.io/gorm"
)

type RetentionRepository struct {
	db *gorm.DB
}

func NewRetentionRepository(db *gorm.DB) *RetentionRepository {
	return &RetentionRepository{db: db}
}

func (r *RetentionRepository) Upsert(p *models.RetentionPolicy) error {
	var existing models.RetentionPolicy
	err := r.db.Where("registry_id = ? AND repository = ?", p.RegistryID, p.Repository).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(p).Error
	}
	if err != nil {
		return err
	}
	existing.KeepLatest = p.KeepLatest
	existing.DeleteOlderDays = p.DeleteOlderDays
	return r.db.Save(&existing).Error
}

func (r *RetentionRepository) Get(registryID uint, repository string) (*models.RetentionPolicy, error) {
	var p models.RetentionPolicy
	if err := r.db.Where("registry_id = ? AND repository = ?", registryID, repository).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *RetentionRepository) ListByRegistry(registryID uint) ([]models.RetentionPolicy, error) {
	var list []models.RetentionPolicy
	err := r.db.Where("registry_id = ?", registryID).Find(&list).Error
	return list, err
}
