package tests

import (
	"testing"

	"github.com/regpilot/regpilot/internal/models"
	"github.com/regpilot/regpilot/internal/utils"
)

func TestLoginSuccess(t *testing.T) {
	app := setupTestApp(t)
	code, resp := app.login(t, "admin", "change-me-admin")
	if code != 200 {
		t.Fatalf("expected 200, got %d: %v", code, resp)
	}
	data := resp["data"].(map[string]interface{})
	if data["token"] == nil || data["token"] == "" {
		t.Fatal("expected token")
	}
}

func TestLoginInvalid(t *testing.T) {
	app := setupTestApp(t)
	code, resp := app.login(t, "admin", "wrong-password")
	if code != 401 {
		t.Fatalf("expected 401, got %d: %v", code, resp)
	}
}

func TestPasswordNotStoredPlaintext(t *testing.T) {
	app := setupTestApp(t)
	u, err := app.userRepo.FindByUsername("admin")
	if err != nil {
		t.Fatal(err)
	}
	if u.PasswordHash == "change-me-admin" {
		t.Fatal("password stored in plaintext")
	}
	if !utils.CheckPassword(u.PasswordHash, "change-me-admin") {
		t.Fatal("bcrypt hash invalid")
	}
}

func TestAdminAuthorization(t *testing.T) {
	app := setupTestApp(t)
	tok := app.token(t, "admin", "change-me-admin")
	w := app.do("GET", "/api/users", tok, nil)
	if w.Code != 200 {
		t.Fatalf("admin should list users: %d %s", w.Code, w.Body.String())
	}
}

func TestViewerAuthorization(t *testing.T) {
	app := setupTestApp(t)
	adminTok := app.token(t, "admin", "change-me-admin")
	w := app.do("POST", "/api/users", adminTok, map[string]string{
		"username": "viewer1",
		"password": "password123",
		"role":     models.RoleViewer,
	})
	if w.Code != 201 {
		t.Fatalf("create viewer: %d %s", w.Code, w.Body.String())
	}
	viewerTok := app.token(t, "viewer1", "password123")
	w = app.do("GET", "/api/users", viewerTok, nil)
	if w.Code != 403 {
		t.Fatalf("viewer should be forbidden: %d", w.Code)
	}
	w = app.do("POST", "/api/registries", viewerTok, map[string]interface{}{
		"name": "r", "url": "http://127.0.0.1:5000",
	})
	if w.Code != 403 {
		t.Fatalf("viewer cannot create registry: %d", w.Code)
	}
}

func TestAuditLogOnLogin(t *testing.T) {
	app := setupTestApp(t)
	_, _ = app.login(t, "admin", "change-me-admin")
	tok := app.token(t, "admin", "change-me-admin")
	w := app.do("GET", "/api/audit-logs", tok, nil)
	if w.Code != 200 {
		t.Fatalf("audit logs: %d %s", w.Code, w.Body.String())
	}
}
