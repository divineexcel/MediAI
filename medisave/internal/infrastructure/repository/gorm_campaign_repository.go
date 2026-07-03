package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/medisave/app/internal/domain/entity"
	domainrepo "github.com/medisave/app/internal/domain/repository"
)

type GORMCampaignRepository struct {
	db *gorm.DB
}

func NewGORMCampaignRepository(db *gorm.DB) domainrepo.CampaignRepository {
	return &GORMCampaignRepository{db: db}
}

func (r *GORMCampaignRepository) dbc(ctx context.Context) *gorm.DB {
	if tx, ok := domainrepo.GetTransaction(ctx).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *GORMCampaignRepository) Create(ctx context.Context, c *entity.HealthCampaign) error {
	return r.dbc(ctx).Create(c).Error
}

func (r *GORMCampaignRepository) List(ctx context.Context) ([]*entity.HealthCampaign, error) {
	var items []*entity.HealthCampaign
	err := r.dbc(ctx).Order("created_at desc").Find(&items).Error
	return items, err
}
