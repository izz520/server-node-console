package router

import (
	"net/http"

	"server-sing-box-2/backend/internal/config"
	"server-sing-box-2/backend/internal/handler"
	"server-sing-box-2/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Dependencies struct {
	Config config.Config
	DB     *gorm.DB
}

func New(deps Dependencies) *gin.Engine {
	if deps.Config.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), cors(deps.Config.CORSAllowedOrigins))

	h := handler.New(deps.DB)

	r.GET("/healthz", h.Health)

	api := r.Group("/api/v1")
	{
		api.POST("/auth/register", h.NotImplemented("auth.register"))
		api.POST("/auth/login", h.NotImplemented("auth.login"))

		protected := api.Group("")
		protected.Use(middleware.Auth(deps.Config.JWTSecret))
		{
			protected.GET("/me", h.NotImplemented("users.me"))

			protected.GET("/servers", h.NotImplemented("servers.list"))
			protected.POST("/servers", h.NotImplemented("servers.create"))
			protected.GET("/servers/:id", h.NotImplemented("servers.get"))
			protected.PUT("/servers/:id", h.NotImplemented("servers.update"))
			protected.DELETE("/servers/:id", h.NotImplemented("servers.delete"))
			protected.POST("/servers/:id/test-ssh", h.NotImplemented("servers.testSSH"))

			protected.GET("/servers/:id/nat-mappings", h.NotImplemented("natMappings.list"))
			protected.POST("/servers/:id/nat-mappings", h.NotImplemented("natMappings.create"))
			protected.PUT("/nat-mappings/:id", h.NotImplemented("natMappings.update"))
			protected.DELETE("/nat-mappings/:id", h.NotImplemented("natMappings.delete"))

			protected.GET("/nodes", h.NotImplemented("nodes.list"))
			protected.POST("/nodes/install", h.NotImplemented("nodes.install"))
			protected.POST("/nodes/import", h.NotImplemented("nodes.import"))
			protected.POST("/nodes/:id/uninstall", h.NotImplemented("nodes.uninstall"))
			protected.PUT("/nodes/:id", h.NotImplemented("nodes.update"))
			protected.DELETE("/nodes/:id", h.NotImplemented("nodes.delete"))

			protected.GET("/subscriptions", h.NotImplemented("subscriptions.list"))
			protected.POST("/subscriptions", h.NotImplemented("subscriptions.create"))
			protected.GET("/subscriptions/:id", h.NotImplemented("subscriptions.get"))
			protected.PUT("/subscriptions/:id", h.NotImplemented("subscriptions.update"))
			protected.DELETE("/subscriptions/:id", h.NotImplemented("subscriptions.delete"))
			protected.POST("/subscriptions/:id/reset-token", h.NotImplemented("subscriptions.resetToken"))

			protected.GET("/tasks", h.NotImplemented("tasks.list"))
			protected.GET("/tasks/:id", h.NotImplemented("tasks.get"))

			admin := protected.Group("/admin")
			admin.Use(middleware.AdminOnly())
			{
				admin.GET("/users", h.NotImplemented("admin.users.list"))
				admin.GET("/servers", h.NotImplemented("admin.servers.list"))
				admin.GET("/nodes", h.NotImplemented("admin.nodes.list"))
				admin.GET("/subscriptions", h.NotImplemented("admin.subscriptions.list"))
				admin.GET("/tasks", h.NotImplemented("admin.tasks.list"))
			}
		}
	}

	r.GET("/sub/:token", h.NotImplemented("public.subscription"))

	return r
}

func cors(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowed[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
