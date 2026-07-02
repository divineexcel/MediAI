package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/domain/repository"
)

type GORMConsultationRoomRepository struct {
	db *gorm.DB
}

func NewGORMConsultationRoomRepository(db *gorm.DB) repository.ConsultationRoomRepository {
	return &GORMConsultationRoomRepository{db: db}
}

func (r *GORMConsultationRoomRepository) FindByAppointmentID(ctx context.Context, appointmentID uint) (*entity.ConsultationRoom, error) {
	var room entity.ConsultationRoom
	err := r.db.WithContext(ctx).Where("appointment_id = ?", appointmentID).First(&room).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *GORMConsultationRoomRepository) Create(ctx context.Context, room *entity.ConsultationRoom) error {
	return r.db.WithContext(ctx).Create(room).Error
}

func (r *GORMConsultationRoomRepository) End(ctx context.Context, appointmentID uint) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entity.ConsultationRoom{}).
		Where("appointment_id = ? AND status = ?", appointmentID, entity.ConsultationRoomStatusActive).
		Updates(map[string]interface{}{
			"status":   entity.ConsultationRoomStatusEnded,
			"ended_at": &now,
		}).Error
}
