package bootstrap

import (
	"github.com/medisave/app/config"
	"github.com/medisave/app/internal/presentation/http/router"
	pkgjwt "github.com/medisave/app/pkg/jwt"
	"gorm.io/gorm"
)

func RegisterRoutes(r *router.Router, db *gorm.DB, cfg *config.Config, jwtManager *pkgjwt.Manager) {
	handlers := NewContainer(db, cfg, jwtManager)
	r.RegisterAll(handlers)
}
