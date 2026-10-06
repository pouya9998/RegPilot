package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/regpilot/regpilot/internal/config"
	"github.com/regpilot/regpilot/internal/database"
	"github.com/regpilot/regpilot/internal/handlers"
	"github.com/regpilot/regpilot/internal/middleware"
	"github.com/regpilot/regpilot/internal/repositories"
	"github.com/regpilot/regpilot/internal/services"
	"gorm.io/gorm"
)

type testApp struct {
	router   *gin.Engine
	db       *gorm.DB
	cfg      *config.Config
	auth     *services.AuthService
	users    *services.UserService
	regs     *services.RegistryService
	userRepo *repositories.UserRepository
}

func setupTestApp(t *testing.T) *testApp {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	cfg := &config.Config{
		AppEnv:             "test",
		AppPort:            "8080",
		DatabasePath:       dbPath,
		JWTSecret:          "test-secret-key-for-jwt",
		JWTExpiration:      24 * time.Hour,
		AdminUsername:      "admin",
		AdminPassword:      "change-me-admin",
		EncryptionKey:      "test-encryption-key-32b!!!!",
		CORSAllowedOrigins: []string{"*"},
		EnableSwagger:      false,
		LoginRateLimit:     100,
		LoginRateWindow:    time.Minute,
		StaticDir:          "",
	}

	db, err := database.Connect(dbPath)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	userRepo := repositories.NewUserRepository(db)
	regRepo := repositories.NewRegistryRepository(db)
	auditRepo := repositories.NewAuditRepository(db)
	retentionRepo := repositories.NewRetentionRepository(db)

	authSvc := services.NewAuthService(userRepo, auditRepo, cfg)
	userSvc := services.NewUserService(userRepo, auditRepo)
	regSvc := services.NewRegistryService(regRepo, auditRepo, retentionRepo, cfg)

	if err := authSvc.EnsureAdmin(); err != nil {
		t.Fatalf("admin: %v", err)
	}

	authHandler := handlers.NewAuthHandler(authSvc)
	userHandler := handlers.NewUserHandler(userSvc)
	regHandler := handlers.NewRegistryHandler(regSvc)
	auditHandler := handlers.NewAuditHandler(services.NewAuditService(auditRepo))

	r := gin.New()
	api := r.Group("/api")
	api.GET("/health", handlers.Health)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/logout", middleware.Auth(cfg.JWTSecret), authHandler.Logout)
	api.GET("/auth/me", middleware.Auth(cfg.JWTSecret), authHandler.Me)

	protected := api.Group("")
	protected.Use(middleware.Auth(cfg.JWTSecret))
	{
		protected.GET("/dashboard", regHandler.Dashboard)
		protected.GET("/registries", regHandler.List)
		protected.POST("/registries", middleware.RequireAdmin(), regHandler.Create)
		protected.GET("/registries/:id", regHandler.Get)
		protected.POST("/registries/:id/test", middleware.RequireAdmin(), regHandler.Test)
		protected.GET("/registries/:id/repositories", regHandler.ListRepositories)
		protected.GET("/registries/:id/tags", regHandler.ListTags)
		protected.GET("/registries/:id/manifests", regHandler.GetManifest)
		protected.DELETE("/registries/:id/images", middleware.RequireAdmin(), regHandler.DeleteImage)
		protected.GET("/users", middleware.RequireAdmin(), userHandler.List)
		protected.POST("/users", middleware.RequireAdmin(), userHandler.Create)
		protected.GET("/audit-logs", middleware.RequireAdmin(), auditHandler.List)
	}

	_ = os.Setenv("TEST", "1")

	return &testApp{
		router: r, db: db, cfg: cfg, auth: authSvc, users: userSvc, regs: regSvc, userRepo: userRepo,
	}
}

func (a *testApp) login(t *testing.T, username, password string) (int, map[string]interface{}) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	a.router.ServeHTTP(w, req)
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w.Code, resp
}

func (a *testApp) token(t *testing.T, username, password string) string {
	t.Helper()
	code, resp := a.login(t, username, password)
	if code != 200 {
		t.Fatalf("login failed: %d %v", code, resp)
	}
	data := resp["data"].(map[string]interface{})
	return data["token"].(string)
}

func (a *testApp) do(method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	a.router.ServeHTTP(w, req)
	return w
}
