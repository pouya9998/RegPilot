package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/regpilot/regpilot/internal/config"
	"github.com/regpilot/regpilot/internal/models"
	"github.com/regpilot/regpilot/internal/repositories"
	"github.com/regpilot/regpilot/internal/utils"
	"gorm.io/gorm"
)

type AuthService struct {
	users  *repositories.UserRepository
	audit  *repositories.AuditRepository
	cfg    *config.Config
}

func NewAuthService(users *repositories.UserRepository, audit *repositories.AuditRepository, cfg *config.Config) *AuthService {
	return &AuthService{users: users, audit: audit, cfg: cfg}
}

func (s *AuthService) EnsureAdmin() error {
	_, err := s.users.FindByUsername(s.cfg.AdminUsername)
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	hash, err := utils.HashPassword(s.cfg.AdminPassword)
	if err != nil {
		return err
	}
	admin := &models.User{
		Username:     s.cfg.AdminUsername,
		PasswordHash: hash,
		Role:         models.RoleAdmin,
		Disabled:     false,
	}
	return s.users.Create(admin)
}

type LoginResult struct {
	Token     string      `json:"token"`
	ExpiresIn int64       `json:"expires_in"`
	User      UserDTO     `json:"user"`
}

type UserDTO struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Disabled bool   `json:"disabled"`
}

func ToUserDTO(u *models.User) UserDTO {
	return UserDTO{
		ID:       u.ID,
		Username: u.Username,
		Role:     u.Role,
		Disabled: u.Disabled,
	}
}

func (s *AuthService) Login(username, password string) (*LoginResult, error) {
	user, err := s.users.FindByUsername(username)
	if err != nil {
		s.auditLogin(nil, username, "failure")
		return nil, errors.New("invalid credentials")
	}
	if user.Disabled {
		s.auditLogin(&user.ID, username, "failure")
		return nil, errors.New("account disabled")
	}
	if !utils.CheckPassword(user.PasswordHash, password) {
		s.auditLogin(&user.ID, username, "failure")
		return nil, errors.New("invalid credentials")
	}
	token, err := utils.GenerateToken(user.ID, user.Username, user.Role, s.cfg.JWTSecret, s.cfg.JWTExpiration)
	if err != nil {
		return nil, fmt.Errorf("token generation failed")
	}
	s.auditLogin(&user.ID, username, "success")
	utils.LogAction(username, models.ActionLogin, "", "", "", "success", nil)
	return &LoginResult{
		Token:     token,
		ExpiresIn: int64(s.cfg.JWTExpiration / time.Second),
		User:      ToUserDTO(user),
	}, nil
}

func (s *AuthService) Logout(userID uint, username string) {
	uid := userID
	_ = s.audit.Create(&models.AuditLog{
		UserID:   &uid,
		Username: username,
		Action:   models.ActionLogout,
		Result:   "success",
	})
	utils.LogAction(username, models.ActionLogout, "", "", "", "success", nil)
}

func (s *AuthService) auditLogin(userID *uint, username, result string) {
	_ = s.audit.Create(&models.AuditLog{
		UserID:   userID,
		Username: username,
		Action:   models.ActionLogin,
		Result:   result,
	})
}
