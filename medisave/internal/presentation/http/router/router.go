package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/medisave/app/config"
	"github.com/medisave/app/internal/presentation/http/middleware"
	pkgjwt "github.com/medisave/app/pkg/jwt"
)

type Router struct {
	engine     *gin.Engine
	jwtManager *pkgjwt.Manager
	cfg        *config.Config
}

func New(cfg *config.Config, jwtManager *pkgjwt.Manager) *Router {
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestLogger())
	engine.Use(middleware.RateLimit(cfg.Rate.RequestsPerMinute))
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Serve static assets
	engine.Static("/static", "./web/static")

	// Health check
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "medisave"})
	})

	return &Router{engine: engine, jwtManager: jwtManager, cfg: cfg}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}

func (r *Router) V1() *gin.RouterGroup {
	return r.engine.Group("/api/v1")
}

func (r *Router) Authenticated() gin.HandlerFunc {
	return middleware.Auth(r.jwtManager)
}
