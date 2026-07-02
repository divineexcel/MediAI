package service

import (
	"context"
	"fmt"

	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/domain/repository"
	livekit "github.com/medisave/app/internal/infrastructure/external/livekit"
	pkgerrors "github.com/medisave/app/pkg/errors"
)

// RoomTokenResponse is returned to the frontend so it can connect to LiveKit.
type RoomTokenResponse struct {
	WSURL    string `json:"ws_url"`
	RoomName string `json:"room_name"`
	Token    string `json:"token"`
	Identity string `json:"identity"`
}

type ConsultationRoomService interface {
	// GetOrCreateRoom returns a LiveKit token for the caller.
	// Creates the DB room record on first call. Only the assigned doctor and patient may call this.
	GetOrCreateRoom(ctx context.Context, userID uint, role entity.Role, appointmentID uint) (*RoomTokenResponse, error)
	// EndRoom marks the room as ended. Called when the doctor completes the consultation.
	EndRoom(ctx context.Context, appointmentID uint) error
}

type consultationRoomService struct {
	roomRepo    repository.ConsultationRoomRepository
	apptRepo    repository.AppointmentRepository
	patientRepo repository.PatientRepository
	doctorRepo  repository.DoctorRepository
	lk          *livekit.Client
}

func NewConsultationRoomService(
	roomRepo repository.ConsultationRoomRepository,
	apptRepo repository.AppointmentRepository,
	patientRepo repository.PatientRepository,
	doctorRepo repository.DoctorRepository,
	lk *livekit.Client,
) ConsultationRoomService {
	return &consultationRoomService{
		roomRepo:    roomRepo,
		apptRepo:    apptRepo,
		patientRepo: patientRepo,
		doctorRepo:  doctorRepo,
		lk:          lk,
	}
}

func (s *consultationRoomService) GetOrCreateRoom(ctx context.Context, userID uint, role entity.Role, appointmentID uint) (*RoomTokenResponse, error) {
	appt, err := s.apptRepo.FindByID(ctx, appointmentID)
	if err != nil {
		return nil, pkgerrors.ErrAppointmentNotFound
	}

	// Verify caller is a participant of this appointment.
	isHost, identity, err := s.resolveParticipant(ctx, userID, role, appt)
	if err != nil {
		return nil, err
	}

	// For patients, only allow joining once the doctor has started.
	if !isHost && appt.Status != entity.AppointmentStatusInProgress {
		return nil, pkgerrors.ErrAppointmentNotInProgress
	}

	// Lazy-create the room record.
	room, err := s.roomRepo.FindByAppointmentID(ctx, appointmentID)
	if err != nil {
		return nil, pkgerrors.ErrInternalServer
	}
	if room == nil {
		room = &entity.ConsultationRoom{
			AppointmentID: appointmentID,
			RoomName:      livekit.RoomName(appointmentID),
			Status:        entity.ConsultationRoomStatusActive,
		}
		if err := s.roomRepo.Create(ctx, room); err != nil {
			return nil, pkgerrors.ErrInternalServer
		}
	}

	token, err := s.lk.TokenForRoom(room.RoomName, identity, isHost)
	if err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	return &RoomTokenResponse{
		WSURL:    s.lk.WSURL,
		RoomName: room.RoomName,
		Token:    token,
		Identity: identity,
	}, nil
}

func (s *consultationRoomService) EndRoom(ctx context.Context, appointmentID uint) error {
	return s.roomRepo.End(ctx, appointmentID)
}

// resolveParticipant confirms the user belongs to this appointment and returns
// (isHost, identity, error). identity is the string sent to LiveKit as the participant label.
func (s *consultationRoomService) resolveParticipant(ctx context.Context, userID uint, role entity.Role, appt *entity.Appointment) (bool, string, error) {
	if role == entity.RoleDoctor {
		doctor, err := s.doctorRepo.FindByUserID(ctx, userID)
		if err != nil || doctor.ID != appt.DoctorID {
			return false, "", pkgerrors.ErrForbidden
		}
		return true, fmt.Sprintf("doctor-%d", userID), nil
	}

	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil || patient.ID != appt.PatientID {
		return false, "", pkgerrors.ErrForbidden
	}
	return false, fmt.Sprintf("patient-%d", userID), nil
}
