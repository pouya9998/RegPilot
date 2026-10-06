package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/regpilot/regpilot/internal/config"
	"github.com/regpilot/regpilot/internal/database"
	"github.com/regpilot/regpilot/internal/handlers"
	"github.com/regpilot/regpilot/internal/middleware"
	"github.com/regpilot/regpilot/internal/repositories"
	"github.com/regpilot/regpilot/internal/services"
	"github.com/regpilot/regpilot/internal/utils"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}
	utils.SetupLogger(cfg.AppEnv)

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Connect(cfg.DatabasePath)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	if err := database.Migrate(db); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	userRepo := repositories.NewUserRepository(db)
	regRepo := repositories.NewRegistryRepository(db)
	auditRepo := repositories.NewAuditRepository(db)
	retentionRepo := repositories.NewRetentionRepository(db)

	authSvc := services.NewAuthService(userRepo, auditRepo, cfg)
	userSvc := services.NewUserService(userRepo, auditRepo)
	regSvc := services.NewRegistryService(regRepo, auditRepo, retentionRepo, cfg)
	auditSvc := services.NewAuditService(auditRepo)

	if err := authSvc.EnsureAdmin(); err != nil {
		slog.Error("failed to ensure admin user", "error", err)
		os.Exit(1)
	}

	authHandler := handlers.NewAuthHandler(authSvc)
	userHandler := handlers.NewUserHandler(userSvc)
	regHandler := handlers.NewRegistryHandler(regSvc)
	auditHandler := handlers.NewAuditHandler(auditSvc)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.SecurityHeaders())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	loginLimiter := middleware.NewLoginRateLimiter(cfg.LoginRateLimit, cfg.LoginRateWindow)

	api := r.Group("/api")
	{
		api.GET("/health", handlers.Health)

		auth := api.Group("/auth")
		{
			auth.POST("/login", loginLimiter.Middleware(), authHandler.Login)
			auth.POST("/logout", middleware.Auth(cfg.JWTSecret), authHandler.Logout)
			auth.GET("/me", middleware.Auth(cfg.JWTSecret), authHandler.Me)
		}

		protected := api.Group("")
		protected.Use(middleware.Auth(cfg.JWTSecret))
		{
			protected.GET("/dashboard", regHandler.Dashboard)

			protected.GET("/registries", regHandler.List)
			protected.GET("/registries/:id", regHandler.Get)
			protected.POST("/registries", middleware.RequireAdmin(), regHandler.Create)
			protected.PUT("/registries/:id", middleware.RequireAdmin(), regHandler.Update)
			protected.DELETE("/registries/:id", middleware.RequireAdmin(), regHandler.Delete)
			protected.POST("/registries/:id/test", middleware.RequireAdmin(), regHandler.Test)
			protected.GET("/registries/:id/health", regHandler.Health)
			protected.GET("/registries/:id/storage", regHandler.Storage)
			protected.GET("/registries/:id/repositories", regHandler.ListRepositories)
			protected.GET("/registries/:id/tags", regHandler.ListTags)
			protected.GET("/registries/:id/manifests", regHandler.GetManifest)
			protected.DELETE("/registries/:id/images", middleware.RequireAdmin(), regHandler.DeleteImage)
			protected.POST("/registries/:id/gc", middleware.RequireAdmin(), regHandler.GC)
			protected.POST("/registries/:id/cleanup/preview", middleware.RequireAdmin(), regHandler.PreviewCleanup)

			protected.GET("/users", middleware.RequireAdmin(), userHandler.List)
			protected.POST("/users", middleware.RequireAdmin(), userHandler.Create)
			protected.PATCH("/users/:id", middleware.RequireAdmin(), userHandler.SetDisabled)
			protected.POST("/users/:id/reset-password", middleware.RequireAdmin(), userHandler.ResetPassword)
			protected.DELETE("/users/:id", middleware.RequireAdmin(), userHandler.Delete)

			protected.GET("/audit-logs", middleware.RequireAdmin(), auditHandler.List)
		}
	}

	if cfg.EnableSwagger {
		r.GET("/api/openapi.json", func(c *gin.Context) {
			candidates := []string{
				filepath.Join(cfg.StaticDir, "openapi.json"),
				"/app/docs/openapi.json",
				"docs/openapi.json",
			}
			for _, p := range candidates {
				if _, err := os.Stat(p); err == nil {
					c.File(p)
					return
				}
			}
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "OpenAPI spec not found"}})
		})
		r.GET("/swagger", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, swaggerHTML)
		})
		r.GET("/swagger/*any", func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/swagger")
		})
	}

	staticDir := cfg.StaticDir
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		r.Static("/assets", filepath.Join(staticDir, "assets"))
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") || strings.HasPrefix(c.Request.URL.Path, "/swagger") {
				c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "Not found"}})
				return
			}
			index := filepath.Join(staticDir, "index.html")
			c.File(index)
		})
	}

	addr := ":" + cfg.AppPort
	slog.Info("RegPilot starting", "addr", addr, "env", cfg.AppEnv)
	if err := r.Run(addr); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

const swaggerHTML = `<!DOCTYPE html>
<html>
<head>
  <title>RegPilot API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({ url: '/api/openapi.json', dom_id: '#swagger-ui' });
  </script>
</body>
</html>`
