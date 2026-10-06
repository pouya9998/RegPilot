package repositories

import (
	"github.com/regpilot/regpilot/internal/models"
	"gorm.io/gorm"
)

type RegistryRepository struct {
	db *gorm.DB
}

func NewRegistryRepository(db *gorm.DB) *RegistryRepository {
	return &RegistryRepository{db: db}
}

func (r *RegistryRepository) Create(reg *models.Registry) error {
	return r.db.Create(reg).Error
}

func (r *RegistryRepository) FindByID(id uint) (*models.Registry, error) {
	var reg models.Registry
	if err := r.db.First(&reg, id).Error; err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *RegistryRepository) List() ([]models.Registry, error) {
	var list []models.Registry
	err := r.db.Order("id asc").Find(&list).Error
	return list, err
}

func (r *RegistryRepository) Update(reg *models.Registry) error {
	return r.db.Save(reg).Error
}

func (r *RegistryRepository) Delete(id uint) error {
	return r.db.Delete(&models.Registry{}, id).Error
}

func (r *RegistryRepository) Count() (int64, error) {
	var n int64
	err := r.db.Model(&models.Registry{}).Count(&n).Error
	return n, err
}
