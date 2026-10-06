package services

import (
	"errors"
	"fmt"

	"github.com/regpilot/regpilot/internal/models"
	"github.com/regpilot/regpilot/internal/repositories"
	"github.com/regpilot/regpilot/internal/utils"
	"gorm.io/gorm"
)

type UserService struct {
	users *repositories.UserRepository
	audit *repositories.AuditRepository
}

func NewUserService(users *repositories.UserRepository, audit *repositories.AuditRepository) *UserService {
	return &UserService{users: users, audit: audit}
}

type CreateUserInput struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=8,max=128"`
	Role     string `json:"role" binding:"required"`
}

func (s *UserService) List() ([]UserDTO, error) {
	users, err := s.users.List()
	if err != nil {
		return nil, err
	}
	out := make([]UserDTO, 0, len(users))
	for i := range users {
		out = append(out, ToUserDTO(&users[i]))
	}
	return out, nil
}

func (s *UserService) Create(actorID uint, actorName string, in CreateUserInput) (*UserDTO, error) {
	if in.Role != models.RoleAdmin && in.Role != models.RoleViewer {
		return nil, errors.New("role must be ADMIN or VIEWER")
	}
	_, err := s.users.FindByUsername(in.Username)
	if err == nil {
		return nil, errors.New("username already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	hash, err := utils.HashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	u := &models.User{
		Username:     in.Username,
		PasswordHash: hash,
		Role:         in.Role,
	}
	if err := s.users.Create(u); err != nil {
		return nil, err
	}
	aid := actorID
	_ = s.audit.Create(&models.AuditLog{
		UserID:   &aid,
		Username: actorName,
		Action:   models.ActionCreateUser,
		Metadata: fmt.Sprintf(`{"created_user":"%s","role":"%s"}`, in.Username, in.Role),
		Result:   "success",
	})
	dto := ToUserDTO(u)
	return &dto, nil
}

func (s *UserService) SetDisabled(actorID uint, actorName string, id uint, disabled bool) (*UserDTO, error) {
	u, err := s.users.FindByID(id)
	if err != nil {
		return nil, err
	}
	if u.ID == actorID && disabled {
		return nil, errors.New("cannot disable your own account")
	}
	u.Disabled = disabled
	if err := s.users.Update(u); err != nil {
		return nil, err
	}
	aid := actorID
	_ = s.audit.Create(&models.AuditLog{
		UserID:   &aid,
		Username: actorName,
		Action:   models.ActionUpdateUser,
		Metadata: fmt.Sprintf(`{"target_user":"%s","disabled":%v}`, u.Username, disabled),
		Result:   "success",
	})
	dto := ToUserDTO(u)
	return &dto, nil
}

func (s *UserService) ResetPassword(actorID uint, actorName string, id uint, password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	u, err := s.users.FindByID(id)
	if err != nil {
		return err
	}
	hash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	if err := s.users.Update(u); err != nil {
		return err
	}
	aid := actorID
	_ = s.audit.Create(&models.AuditLog{
		UserID:   &aid,
		Username: actorName,
		Action:   models.ActionResetPassword,
		Metadata: fmt.Sprintf(`{"target_user":"%s"}`, u.Username),
		Result:   "success",
	})
	return nil
}

func (s *UserService) Delete(actorID uint, actorName string, id uint) error {
	if actorID == id {
		return errors.New("cannot delete your own account")
	}
	u, err := s.users.FindByID(id)
	if err != nil {
		return err
	}
	if err := s.users.Delete(id); err != nil {
		return err
	}
	aid := actorID
	_ = s.audit.Create(&models.AuditLog{
		UserID:   &aid,
		Username: actorName,
		Action:   models.ActionDeleteUser,
		Metadata: fmt.Sprintf(`{"deleted_user":"%s"}`, u.Username),
		Result:   "success",
	})
	return nil
}
