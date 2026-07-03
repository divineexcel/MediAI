package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/medisave/app/internal/domain/entity"
	domainrepo "github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/pagination"
)

func dbc(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := domainrepo.GetTransaction(ctx).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}

// ─── Appointment Repository ───────────────────────────────────────────────────

type GORMAppointmentRepository struct {
	db *gorm.DB
}

func NewGORMAppointmentRepository(db *gorm.DB) domainrepo.AppointmentRepository {
	return &GORMAppointmentRepository{db: db}
}

func (r *GORMAppointmentRepository) Create(ctx context.Context, a *entity.Appointment) error {
	return dbc(ctx, r.db).Create(a).Error
}

func (r *GORMAppointmentRepository) FindByID(ctx context.Context, id uint) (*entity.Appointment, error) {
	var a entity.Appointment
	err := dbc(ctx, r.db).
		Preload("Patient.User").
		Preload("Doctor.User").
		First(&a, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrAppointmentNotFound
	}
	return &a, err
}

func (r *GORMAppointmentRepository) Update(ctx context.Context, a *entity.Appointment) error {
	return dbc(ctx, r.db).Save(a).Error
}

func (r *GORMAppointmentRepository) ListByPatient(ctx context.Context, patientID uint, status string, p pagination.Params) ([]*entity.Appointment, int64, error) {
	var list []*entity.Appointment
	var total int64

	q := dbc(ctx, r.db).Model(&entity.Appointment{}).Where("patient_id = ?", patientID)
	if status != "" && status != "all" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Preload("Doctor.User").
		Order("scheduled_at DESC").
		Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

func (r *GORMAppointmentRepository) ListByDoctor(ctx context.Context, doctorID uint, status string, p pagination.Params) ([]*entity.Appointment, int64, error) {
	var list []*entity.Appointment
	var total int64

	q := dbc(ctx, r.db).Model(&entity.Appointment{}).Where("doctor_id = ?", doctorID)
	if status != "" && status != "all" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Preload("Patient.User").
		Order("scheduled_at DESC").
		Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

func (r *GORMAppointmentRepository) ListTodayByDoctor(ctx context.Context, doctorID uint) ([]*entity.Appointment, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)

	var list []*entity.Appointment
	err := dbc(ctx, r.db).
		Preload("Patient.User").
		Where("doctor_id = ? AND scheduled_at >= ? AND scheduled_at < ? AND status NOT IN ?",
			doctorID, start, end, []string{"cancelled", "no_show"}).
		Order("scheduled_at ASC").
		Find(&list).Error
	return list, err
}

func (r *GORMAppointmentRepository) FindConflict(ctx context.Context, doctorID uint, scheduled time.Time) (*entity.Appointment, error) {
	window := time.Hour
	var a entity.Appointment
	err := dbc(ctx, r.db).
		Where("doctor_id = ? AND status NOT IN ? AND scheduled_at BETWEEN ? AND ?",
			doctorID,
			[]string{string(entity.AppointmentStatusCancelled), string(entity.AppointmentStatusCompleted), string(entity.AppointmentStatusNoShow)},
			scheduled.Add(-window), scheduled.Add(window),
		).First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &a, err
}

func (r *GORMAppointmentRepository) UpdateStatus(ctx context.Context, id uint, status entity.AppointmentStatus) error {
	return dbc(ctx, r.db).
		Model(&entity.Appointment{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *GORMAppointmentRepository) CountByDoctor(ctx context.Context, doctorID uint) (int64, error) {
	var count int64
	err := dbc(ctx, r.db).
		Model(&entity.Appointment{}).
		Where("doctor_id = ? AND status NOT IN ?", doctorID, []string{"cancelled", "no_show"}).
		Count(&count).Error
	return count, err
}

func (r *GORMAppointmentRepository) CountAll(ctx context.Context) (int64, error) {
	var count int64
	err := dbc(ctx, r.db).Model(&entity.Appointment{}).Count(&count).Error
	return count, err
}

func (r *GORMAppointmentRepository) ListAll(ctx context.Context, p pagination.Params) ([]*entity.Appointment, int64, error) {
	var list []*entity.Appointment
	var total int64
	q := dbc(ctx, r.db).Model(&entity.Appointment{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := q.Preload("Patient").Preload("Doctor").Order("created_at DESC").Offset(p.Offset).Limit(p.Limit).Find(&list).Error
	return list, total, err
}

// ─── Consultation Repository ──────────────────────────────────────────────────

type GORMConsultationRepository struct {
	db *gorm.DB
}

func NewGORMConsultationRepository(db *gorm.DB) domainrepo.ConsultationRepository {
	return &GORMConsultationRepository{db: db}
}

func (r *GORMConsultationRepository) Create(ctx context.Context, c *entity.Consultation) error {
	return dbc(ctx, r.db).Create(c).Error
}

func (r *GORMConsultationRepository) FindByAppointmentID(ctx context.Context, appointmentID uint) (*entity.Consultation, error) {
	var c entity.Consultation
	err := dbc(ctx, r.db).Where("appointment_id = ?", appointmentID).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, pkgerrors.ErrConsultationNotFound
	}
	return &c, err
}

func (r *GORMConsultationRepository) Update(ctx context.Context, c *entity.Consultation) error {
	return dbc(ctx, r.db).Save(c).Error
}

func (r *GORMConsultationRepository) CreateMessage(ctx context.Context, m *entity.ConsultationMessage) error {
	return dbc(ctx, r.db).Create(m).Error
}

func (r *GORMConsultationRepository) ListMessages(ctx context.Context, appointmentID uint) ([]*entity.ConsultationMessage, error) {
	var msgs []*entity.ConsultationMessage
	err := dbc(ctx, r.db).
		Where("appointment_id = ?", appointmentID).
		Order("created_at ASC").
		Find(&msgs).Error
	return msgs, err
}

func (r *GORMConsultationRepository) MarkMessagesRead(ctx context.Context, appointmentID uint, readerID uint) error {
	return dbc(ctx, r.db).
		Model(&entity.ConsultationMessage{}).
		Where("appointment_id = ? AND sender_id != ? AND is_read = false", appointmentID, readerID).
		Update("is_read", true).Error
}
