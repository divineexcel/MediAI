package repository

import (
	"context"

	"github.com/medisave/app/internal/domain/entity"
)

type CampaignRepository interface {
	Create(ctx context.Context, c *entity.HealthCampaign) error
	List(ctx context.Context) ([]*entity.HealthCampaign, error)
}
