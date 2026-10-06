package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	RoleAdmin  = "ADMIN"
	RoleViewer = "VIEWER"
)

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string         `gorm:"not null" json:"-"`
	Role         string         `gorm:"size:16;not null;default:VIEWER" json:"role"`
	Disabled     bool           `gorm:"not null;default:false" json:"disabled"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type Registry struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	Name               string         `gorm:"size:128;not null" json:"name"`
	URL                string         `gorm:"size:512;not null" json:"url"`
	Username           string         `gorm:"size:256" json:"username,omitempty"`
	PasswordEncrypted  string         `gorm:"type:text" json:"-"`
	InsecureSkipVerify bool           `gorm:"not null;default:false" json:"insecure_skip_verify"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

// RegistryResponse is safe for API responses (no password).
type RegistryResponse struct {
	ID                 uint      `json:"id"`
	Name               string    `json:"name"`
	URL                string    `json:"url"`
	Username           string    `json:"username,omitempty"`
	HasPassword        bool      `json:"has_password"`
	InsecureSkipVerify bool      `json:"insecure_skip_verify"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (r *Registry) ToResponse() RegistryResponse {
	return RegistryResponse{
		ID:                 r.ID,
		Name:               r.Name,
		URL:                r.URL,
		Username:           r.Username,
		HasPassword:        r.PasswordEncrypted != "",
		InsecureSkipVerify: r.InsecureSkipVerify,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}

type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     *uint     `json:"user_id"`
	Username   string    `gorm:"size:64" json:"username"`
	Action     string    `gorm:"size:64;not null;index" json:"action"`
	RegistryID *uint     `gorm:"index" json:"registry_id"`
	Repository string    `gorm:"size:512" json:"repository"`
	Tag        string    `gorm:"size:256" json:"tag"`
	Digest     string    `gorm:"size:128" json:"digest"`
	Metadata   string    `gorm:"type:text" json:"metadata"`
	Result     string    `gorm:"size:32" json:"result"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}

type RetentionPolicy struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	RegistryID    uint      `gorm:"not null;index" json:"registry_id"`
	Repository    string    `gorm:"size:512;not null" json:"repository"`
	KeepLatest    int       `gorm:"not null;default:10" json:"keep_latest"`
	DeleteOlderDays int     `gorm:"not null;default:0" json:"delete_older_days"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Audit action constants
const (
	ActionLogin          = "LOGIN"
	ActionLogout         = "LOGOUT"
	ActionCreateRegistry = "CREATE_REGISTRY"
	ActionUpdateRegistry = "UPDATE_REGISTRY"
	ActionDeleteRegistry = "DELETE_REGISTRY"
	ActionTestRegistry   = "TEST_REGISTRY"
	ActionDeleteImage    = "DELETE_IMAGE"
	ActionRunGC          = "RUN_GC"
	ActionPreviewCleanup = "PREVIEW_CLEANUP"
	ActionCreateUser     = "CREATE_USER"
	ActionUpdateUser     = "UPDATE_USER"
	ActionDeleteUser     = "DELETE_USER"
	ActionResetPassword  = "RESET_PASSWORD"
)
