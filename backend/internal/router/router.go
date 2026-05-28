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

	h := handler.New(deps.DB, deps.Config.JWTSecret, deps.Config.EncryptionKey)

	r.GET("/healthz", h.Health)

	api := r.Group("/api/v1")
	{
		api.POST("/auth/register", h.Register)
		api.POST("/auth/login", h.Login)

		protected := api.Group("")
		protected.Use(middleware.Auth(deps.Config.JWTSecret))
		{
			protected.GET("/me", h.Me)
			protected.POST("/auth/refresh", h.RefreshToken)

			protected.GET("/servers", h.ListServers)
			protected.POST("/servers", h.CreateServer)
			protected.GET("/servers/:id", h.GetServer)
			protected.PUT("/servers/:id", h.UpdateServer)
			protected.DELETE("/servers/:id", h.DeleteServer)
			protected.POST("/servers/:id/test-ssh", h.TestServerSSH)

			protected.GET("/servers/:id/nat-mappings", h.ListNATMappings)
			protected.POST("/servers/:id/nat-mappings", h.CreateNATMapping)
			protected.PUT("/nat-mappings/:id", h.UpdateNATMapping)
			protected.DELETE("/nat-mappings/:id", h.DeleteNATMapping)

			protected.GET("/nodes", h.ListNodes)
			protected.POST("/nodes/install", h.InstallNode)
			protected.POST("/nodes/import", h.ImportNode)
			protected.POST("/nodes/test-proxy", h.TestAllNodeProxies)
			protected.POST("/nodes/:id/uninstall", h.UninstallNode)
			protected.POST("/nodes/:id/test-proxy", h.TestNodeProxy)
			protected.GET("/nodes/:id/share-link", h.GetNodeShareLink)
			protected.PUT("/nodes/:id", h.UpdateNode)
			protected.DELETE("/nodes/:id", h.DeleteNode)

			protected.GET("/subscriptions", h.ListSubscriptions)
			protected.POST("/subscriptions", h.CreateSubscription)
			protected.GET("/subscriptions/:id", h.GetSubscription)
			protected.PUT("/subscriptions/:id", h.UpdateSubscription)
			protected.DELETE("/subscriptions/:id", h.DeleteSubscription)
			protected.POST("/subscriptions/:id/reset-token", h.ResetSubscriptionToken)
			protected.GET("/clash-templates", h.ListClashTemplates)
			protected.POST("/clash-templates", h.CreateClashTemplate)
			protected.PUT("/clash-templates/:id", h.UpdateClashTemplate)
			protected.DELETE("/clash-templates/:id", h.DeleteClashTemplate)

			protected.GET("/tasks", h.ListTasks)
			protected.GET("/tasks/:id", h.GetTask)
			protected.GET("/operation-logs", h.ListOperationLogs)

			admin := protected.Group("/admin")
			admin.Use(middleware.AdminOnly())
			{
				admin.GET("/users", h.AdminListUsers)
				admin.GET("/servers", h.AdminListServers)
				admin.GET("/nodes", h.AdminListNodes)
				admin.GET("/subscriptions", h.AdminListSubscriptions)
				admin.GET("/tasks", h.AdminListTasks)
				admin.GET("/tasks/:id", h.AdminGetTask)
				admin.GET("/operation-logs", h.AdminListOperationLogs)
			}
		}
	}

	r.GET("/sub/:token", h.PublicSubscription)

	return r
}

func cors(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	allowAll := false
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
			continue
		}
		allowed[origin] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowOrigin := ""
		if allowAll {
			allowOrigin = origin
			if allowOrigin == "" {
				allowOrigin = "*"
			}
		} else if _, ok := allowed[origin]; ok {
			allowOrigin = origin
		}

		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
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
