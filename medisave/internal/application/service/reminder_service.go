package service

import (
	"context"
	"time"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/pagination"
)

type ReminderService interface {
	List(ctx context.Context, userID uint, p pagination.Params) ([]*dto.ReminderResponse, int64, error)
	GetByID(ctx context.Context, userID uint, reminderID uint) (*dto.ReminderResponse, error)
	Create(ctx context.Context, userID uint, req *dto.CreateReminderRequest) (*dto.ReminderResponse, error)
	Update(ctx context.Context, userID uint, reminderID uint, req *dto.CreateReminderRequest) (*dto.ReminderResponse, error)
	Deactivate(ctx context.Context, userID uint, reminderID uint) error
	LogAction(ctx context.Context, userID uint, reminderID uint, req *dto.ReminderLogActionRequest) error
	GetAnalytics(ctx context.Context, userID uint) (*dto.ReminderAdherenceResponse, error)
}

type reminderService struct {
	reminderRepo repository.ReminderRepository
	patientRepo  repository.PatientRepository
}

func NewReminderService(
	reminderRepo repository.ReminderRepository,
	patientRepo repository.PatientRepository,
) ReminderService {
	return &reminderService{
		reminderRepo: reminderRepo,
		patientRepo:  patientRepo,
	}
}

func (s *reminderService) List(ctx context.Context, userID uint, p pagination.Params) ([]*dto.ReminderResponse, int64, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, 0, pkgerrors.ErrPatientNotFound
	}

	list, total, err := s.reminderRepo.ListByPatient(ctx, patient.ID, p)
	if err != nil {
		return nil, 0, pkgerrors.ErrInternalServer
	}

	result := make([]*dto.ReminderResponse, len(list))
	for i, r := range list {
		result[i] = toReminderResponse(r)
	}
	return result, total, nil
}

func (s *reminderService) GetByID(ctx context.Context, userID uint, reminderID uint) (*dto.ReminderResponse, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	rem, err := s.reminderRepo.FindByID(ctx, reminderID)
	if err != nil {
		return nil, err
	}
	if rem.PatientID != patient.ID {
		return nil, pkgerrors.ErrAccessDenied
	}
	return toReminderResponse(rem), nil
}

func (s *reminderService) Create(ctx context.Context, userID uint, req *dto.CreateReminderRequest) (*dto.ReminderResponse, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, pkgerrors.ErrBadRequest
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, pkgerrors.ErrBadRequest
	}
	if end.Before(start) {
		return nil, pkgerrors.ErrBadRequest
	}

	rem := &entity.MedicationReminder{
		PatientID:     patient.ID,
		MedicineName:  req.MedicineName,
		Dosage:        req.Dosage,
		Frequency:     entity.ReminderFrequency(req.Frequency),
		MorningTime:   req.MorningTime,
		AfternoonTime: req.AfternoonTime,
		NightTime:     req.NightTime,
		StartDate:     start,
		EndDate:       end,
		Instructions:  req.Instructions,
		IsActive:      true,
	}

	if err := s.reminderRepo.Create(ctx, rem); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}
	return toReminderResponse(rem), nil
}

func (s *reminderService) Update(ctx context.Context, userID uint, reminderID uint, req *dto.CreateReminderRequest) (*dto.ReminderResponse, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	rem, err := s.reminderRepo.FindByID(ctx, reminderID)
	if err != nil {
		return nil, err
	}
	if rem.PatientID != patient.ID {
		return nil, pkgerrors.ErrAccessDenied
	}

	start, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, pkgerrors.ErrBadRequest
	}
	end, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, pkgerrors.ErrBadRequest
	}

	rem.MedicineName = req.MedicineName
	rem.Dosage = req.Dosage
	rem.Frequency = entity.ReminderFrequency(req.Frequency)
	rem.MorningTime = req.MorningTime
	rem.AfternoonTime = req.AfternoonTime
	rem.NightTime = req.NightTime
	rem.StartDate = start
	rem.EndDate = end
	rem.Instructions = req.Instructions

	if err := s.reminderRepo.Update(ctx, rem); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}
	return toReminderResponse(rem), nil
}

func (s *reminderService) Deactivate(ctx context.Context, userID uint, reminderID uint) error {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return pkgerrors.ErrPatientNotFound
	}

	rem, err := s.reminderRepo.FindByID(ctx, reminderID)
	if err != nil {
		return err
	}
	if rem.PatientID != patient.ID {
		return pkgerrors.ErrAccessDenied
	}
	return s.reminderRepo.Deactivate(ctx, reminderID)
}

func (s *reminderService) LogAction(ctx context.Context, userID uint, reminderID uint, req *dto.ReminderLogActionRequest) error {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return pkgerrors.ErrPatientNotFound
	}

	rem, err := s.reminderRepo.FindByID(ctx, reminderID)
	if err != nil {
		return err
	}
	if rem.PatientID != patient.ID {
		return pkgerrors.ErrAccessDenied
	}

	log := &entity.ReminderLog{
		ReminderID:  reminderID,
		PatientID:   patient.ID,
		ScheduledAt: time.Now(),
		Status:      entity.ReminderLogStatus(req.Status),
		Notes:       req.Notes,
	}
	return s.reminderRepo.CreateLog(ctx, log)
}

func (s *reminderService) GetAnalytics(ctx context.Context, userID uint) (*dto.ReminderAdherenceResponse, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	// Get all logs for this patient's reminders across all pages
	p := pagination.Params{Limit: 1000, Offset: 0}
	reminders, _, err := s.reminderRepo.ListByPatient(ctx, patient.ID, p)
	if err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	var taken, skipped, missed int
	for _, rem := range reminders {
		logs, _, _ := s.reminderRepo.ListLogs(ctx, rem.ID, pagination.Params{Limit: 500, Offset: 0})
		for _, l := range logs {
			switch l.Status {
			case entity.ReminderTaken:
				taken++
			case entity.ReminderSkipped:
				skipped++
			case entity.ReminderMissed:
				missed++
			}
		}
	}

	total := taken + skipped + missed
	var rate float64
	if total > 0 {
		rate = float64(taken) / float64(total) * 100
	}

	return &dto.ReminderAdherenceResponse{
		TotalDoses:    total,
		TakenDoses:    taken,
		SkippedDoses:  skipped,
		MissedDoses:   missed,
		AdherenceRate: rate,
	}, nil
}

func toReminderResponse(r *entity.MedicationReminder) *dto.ReminderResponse {
	return &dto.ReminderResponse{
		ID:            r.ID,
		MedicineName:  r.MedicineName,
		Dosage:        r.Dosage,
		Frequency:     r.Frequency,
		MorningTime:   r.MorningTime,
		AfternoonTime: r.AfternoonTime,
		NightTime:     r.NightTime,
		StartDate:     r.StartDate,
		EndDate:       r.EndDate,
		Instructions:  r.Instructions,
		IsActive:      r.IsActive,
	}
}
