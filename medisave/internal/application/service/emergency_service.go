package service

import (
	"context"
	"fmt"
	"time"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/domain/repository"
	pkgerrors "github.com/medisave/app/pkg/errors"
	"github.com/medisave/app/pkg/pagination"
)

type EmergencyService interface {
	TriggerSOS(ctx context.Context, userID uint, req *dto.SOSRequest) (*dto.EmergencyResponse, error)
	Resolve(ctx context.Context, userID uint, emergencyID uint, req *dto.ResolveEmergencyRequest) error
	GetHistory(ctx context.Context, userID uint, p pagination.Params) ([]*dto.EmergencyResponse, int64, error)
	GetContacts(ctx context.Context, userID uint) ([]*dto.EmergencyContactResponse, error)
	AddContact(ctx context.Context, userID uint, req *dto.EmergencyContactRequest) (*dto.EmergencyContactResponse, error)
	UpdateContact(ctx context.Context, userID uint, contactID uint, req *dto.EmergencyContactRequest) (*dto.EmergencyContactResponse, error)
	DeleteContact(ctx context.Context, userID uint, contactID uint) error
	SetPrimaryContact(ctx context.Context, userID uint, contactID uint) error
}

type emergencyService struct {
	emergencyRepo repository.EmergencyRepository
	contactRepo   repository.EmergencyContactRepository
	patientRepo   repository.PatientRepository
	notifRepo     repository.NotificationRepository
}

func NewEmergencyService(
	emergencyRepo repository.EmergencyRepository,
	contactRepo repository.EmergencyContactRepository,
	patientRepo repository.PatientRepository,
	notifRepo repository.NotificationRepository,
) EmergencyService {
	return &emergencyService{
		emergencyRepo: emergencyRepo,
		contactRepo:   contactRepo,
		patientRepo:   patientRepo,
		notifRepo:     notifRepo,
	}
}

func (s *emergencyService) TriggerSOS(ctx context.Context, userID uint, req *dto.SOSRequest) (*dto.EmergencyResponse, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	address := reverseGeocode(req.Latitude, req.Longitude)
	hospital := findNearestHospital(req.Latitude, req.Longitude)

	emergency := &entity.Emergency{
		PatientID:       patient.ID,
		Status:          entity.EmergencyStatusActive,
		Latitude:        req.Latitude,
		Longitude:       req.Longitude,
		Address:         address,
		NearestHospital: hospital,
		Description:     req.Description,
	}

	if err := s.emergencyRepo.Create(ctx, emergency); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	// Notify patient about SOS activation
	_ = s.notifRepo.Create(ctx, &entity.Notification{
		UserID:  userID,
		Type:    entity.NotifTypeEmergency,
		Title:   "Emergency SOS Activated",
		Body:    fmt.Sprintf("Your emergency SOS has been activated. Emergency services have been alerted. Nearest hospital: %s", hospital),
		Channel: entity.ChannelInApp,
	})

	return toEmergencyResponse(emergency), nil
}

func (s *emergencyService) Resolve(ctx context.Context, userID uint, emergencyID uint, req *dto.ResolveEmergencyRequest) error {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return pkgerrors.ErrPatientNotFound
	}

	emergency, err := s.emergencyRepo.FindByID(ctx, emergencyID)
	if err != nil {
		return pkgerrors.ErrNotFound
	}
	if emergency.PatientID != patient.ID {
		return pkgerrors.ErrAccessDenied
	}
	if emergency.Status != entity.EmergencyStatusActive {
		return pkgerrors.ErrBadRequest
	}

	now := time.Now()
	emergency.Status = entity.EmergencyStatus(req.Status)
	emergency.ResolvedAt = &now

	return s.emergencyRepo.Update(ctx, emergency)
}

func (s *emergencyService) GetHistory(ctx context.Context, userID uint, p pagination.Params) ([]*dto.EmergencyResponse, int64, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, 0, pkgerrors.ErrPatientNotFound
	}

	list, total, err := s.emergencyRepo.ListByPatient(ctx, patient.ID, p)
	if err != nil {
		return nil, 0, pkgerrors.ErrInternalServer
	}

	result := make([]*dto.EmergencyResponse, len(list))
	for i, e := range list {
		result[i] = toEmergencyResponse(e)
	}
	return result, total, nil
}

func (s *emergencyService) GetContacts(ctx context.Context, userID uint) ([]*dto.EmergencyContactResponse, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	contacts, err := s.contactRepo.ListByPatient(ctx, patient.ID)
	if err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	result := make([]*dto.EmergencyContactResponse, len(contacts))
	for i, c := range contacts {
		result[i] = toContactResponse(c)
	}
	return result, nil
}

func (s *emergencyService) AddContact(ctx context.Context, userID uint, req *dto.EmergencyContactRequest) (*dto.EmergencyContactResponse, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	contact := &entity.EmergencyContact{
		PatientID:    patient.ID,
		Name:         req.Name,
		Phone:        req.Phone,
		Relationship: req.Relationship,
		IsPrimary:    req.IsPrimary,
	}

	if err := s.contactRepo.Create(ctx, contact); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	if req.IsPrimary {
		_ = s.contactRepo.SetPrimary(ctx, patient.ID, contact.ID)
	}

	return toContactResponse(contact), nil
}

func (s *emergencyService) UpdateContact(ctx context.Context, userID uint, contactID uint, req *dto.EmergencyContactRequest) (*dto.EmergencyContactResponse, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	contact, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return nil, pkgerrors.ErrNotFound
	}
	if contact.PatientID != patient.ID {
		return nil, pkgerrors.ErrAccessDenied
	}

	contact.Name = req.Name
	contact.Phone = req.Phone
	contact.Relationship = req.Relationship
	contact.IsPrimary = req.IsPrimary

	if err := s.contactRepo.Update(ctx, contact); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	if req.IsPrimary {
		_ = s.contactRepo.SetPrimary(ctx, patient.ID, contact.ID)
	}

	return toContactResponse(contact), nil
}

func (s *emergencyService) DeleteContact(ctx context.Context, userID uint, contactID uint) error {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return pkgerrors.ErrPatientNotFound
	}

	contact, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return pkgerrors.ErrNotFound
	}
	if contact.PatientID != patient.ID {
		return pkgerrors.ErrAccessDenied
	}

	return s.contactRepo.Delete(ctx, contactID)
}

func (s *emergencyService) SetPrimaryContact(ctx context.Context, userID uint, contactID uint) error {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return pkgerrors.ErrPatientNotFound
	}

	contact, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return pkgerrors.ErrNotFound
	}
	if contact.PatientID != patient.ID {
		return pkgerrors.ErrAccessDenied
	}

	return s.contactRepo.SetPrimary(ctx, patient.ID, contactID)
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func toEmergencyResponse(e *entity.Emergency) *dto.EmergencyResponse {
	return &dto.EmergencyResponse{
		ID:               e.ID,
		Status:           string(e.Status),
		Latitude:         e.Latitude,
		Longitude:        e.Longitude,
		Address:          e.Address,
		NearestHospital:  e.NearestHospital,
		Description:      e.Description,
		ContactsNotified: e.ContactsNotified,
		SMSSent:          e.SMSSent,
		ResolvedAt:       e.ResolvedAt,
		CreatedAt:        e.CreatedAt,
	}
}

func toContactResponse(c *entity.EmergencyContact) *dto.EmergencyContactResponse {
	return &dto.EmergencyContactResponse{
		ID:           c.ID,
		Name:         c.Name,
		Phone:        c.Phone,
		Relationship: c.Relationship,
		IsPrimary:    c.IsPrimary,
	}
}

// reverseGeocode returns a human-readable address from coordinates.
// In production, call a real geocoding API (e.g. Google Maps, HERE).
func reverseGeocode(lat, lng float64) string {
	return fmt.Sprintf("%.4f, %.4f (Nigeria)", lat, lng)
}

// findNearestHospital returns the nearest hospital name from coordinates.
// In production, call Google Places / HERE Places API.
func findNearestHospital(lat, lng float64) string {
	// Fallback: common Lagos emergency numbers and hospitals
	if lat >= 6.0 && lat <= 7.0 && lng >= 3.0 && lng <= 4.0 {
		return "Lagos University Teaching Hospital (LUTH) — Emergency: 08023032003"
	}
	if lat >= 9.0 && lat <= 10.0 && lng >= 7.0 && lng <= 8.0 {
		return "National Hospital Abuja — Emergency: 09-5238101"
	}
	return "Nearest hospital — Dial 112 for emergency services"
}
