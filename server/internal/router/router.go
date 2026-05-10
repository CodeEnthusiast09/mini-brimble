package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeploymentHandler interface {
	Create(*gin.Context)
	List(*gin.Context)
	Get(*gin.Context)
	Stop(*gin.Context)
	GetLogs(*gin.Context)
	StreamLogs(*gin.Context)
}

type Handlers struct {
	Deployment DeploymentHandler
}

// Setup registers HTTP routes.
func Setup(r *gin.Engine, h Handlers, db *gorm.DB) {
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/ready", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(503, gin.H{"status": "error", "message": "database unavailable"})
			return
		}

		if err := sqlDB.Ping(); err != nil {
			c.JSON(503, gin.H{"status": "error", "message": "database unreachable"})
			return
		}

		c.JSON(200, gin.H{"status": "ready"})
	})

	api := r.Group("/api/v1")

	// ── Deployment routes ─────────────────────────────────────────────────────
	deploymentRoutes := api.Group("/deployments")
	{
		deploymentRoutes.POST("", h.Deployment.Create)
		deploymentRoutes.GET("", h.Deployment.List)
		deploymentRoutes.GET("/:id", h.Deployment.Get)
		deploymentRoutes.DELETE("/:id", h.Deployment.Stop)

		deploymentRoutes.GET("/:id/logs", h.Deployment.GetLogs)
		deploymentRoutes.GET("/:id/logs/stream", h.Deployment.StreamLogs)
	}
}
