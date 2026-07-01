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

type MedicalRecordService interface {
	List(ctx context.Context, userID uint, p pagination.Params) ([]*entity.MedicalRecord, int64, error)
	GetByID(ctx context.Context, userID uint, recordID uint) (*entity.MedicalRecord, error)
	Create(ctx context.Context, userID uint, req *dto.CreateMedicalRecordRequest) (*entity.MedicalRecord, error)
	Delete(ctx context.Context, userID uint, recordID uint) error
	ListPrescriptions(ctx context.Context, userID uint, p pagination.Params) ([]*entity.Prescription, int64, error)
	MarkPrescriptionFilled(ctx context.Context, userID uint, prescriptionID uint) error
}

type medicalRecordService struct {
	recordRepo  repository.MedicalRecordRepository
	prescRepo   repository.PrescriptionRepository
	patientRepo repository.PatientRepository
}

func NewMedicalRecordService(
	recordRepo repository.MedicalRecordRepository,
	prescRepo repository.PrescriptionRepository,
	patientRepo repository.PatientRepository,
) MedicalRecordService {
	return &medicalRecordService{
		recordRepo:  recordRepo,
		prescRepo:   prescRepo,
		patientRepo: patientRepo,
	}
}

func (s *medicalRecordService) List(ctx context.Context, userID uint, p pagination.Params) ([]*entity.MedicalRecord, int64, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, 0, pkgerrors.ErrPatientNotFound
	}
	return s.recordRepo.ListByPatient(ctx, patient.ID, p)
}

func (s *medicalRecordService) GetByID(ctx context.Context, userID uint, recordID uint) (*entity.MedicalRecord, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	rec, err := s.recordRepo.FindByID(ctx, recordID)
	if err != nil {
		return nil, err
	}
	if rec.PatientID != patient.ID {
		return nil, pkgerrors.ErrAccessDenied
	}
	return rec, nil
}

func (s *medicalRecordService) Create(ctx context.Context, userID uint, req *dto.CreateMedicalRecordRequest) (*entity.MedicalRecord, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	recordDate, err := time.Parse("2006-01-02", req.RecordDate)
	if err != nil {
		recordDate = time.Now()
	}

	rec := &entity.MedicalRecord{
		PatientID:   patient.ID,
		RecordType:  req.RecordType,
		Title:       req.Title,
		Description: req.Description,
		FileURL:     req.FileURL,
		RecordDate:  recordDate,
	}
	if err := s.recordRepo.Create(ctx, rec); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}
	return rec, nil
}

func (s *medicalRecordService) Delete(ctx context.Context, userID uint, recordID uint) error {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return pkgerrors.ErrPatientNotFound
	}

	rec, err := s.recordRepo.FindByID(ctx, recordID)
	if err != nil {
		return err
	}
	if rec.PatientID != patient.ID {
		return pkgerrors.ErrAccessDenied
	}
	return s.recordRepo.Delete(ctx, recordID)
}

func (s *medicalRecordService) ListPrescriptions(ctx context.Context, userID uint, p pagination.Params) ([]*entity.Prescription, int64, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, 0, pkgerrors.ErrPatientNotFound
	}
	return s.prescRepo.ListByPatient(ctx, patient.ID, p)
}

func (s *medicalRecordService) MarkPrescriptionFilled(ctx context.Context, userID uint, prescriptionID uint) error {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return pkgerrors.ErrPatientNotFound
	}

	presc, err := s.prescRepo.FindByID(ctx, prescriptionID)
	if err != nil {
		return pkgerrors.ErrNotFound
	}
	if presc.PatientID != patient.ID {
		return pkgerrors.ErrAccessDenied
	}
	return s.prescRepo.MarkFilled(ctx, prescriptionID)
}
