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
	"github.com/medisave/app/pkg/utils"
)

type AppointmentService interface {
	Book(ctx context.Context, userID uint, req *dto.BookAppointmentRequest) (*dto.AppointmentBookedResponse, error)
	GetByID(ctx context.Context, userID uint, role entity.Role, apptID uint) (*entity.Appointment, error)
	List(ctx context.Context, userID uint, role entity.Role, p pagination.Params) ([]*entity.Appointment, int64, error)
	Cancel(ctx context.Context, userID uint, role entity.Role, apptID uint, reason string) error
	Start(ctx context.Context, userID uint, apptID uint) error
	Complete(ctx context.Context, userID uint, apptID uint) error
	LeaveReview(ctx context.Context, userID uint, apptID uint, req *dto.CreateReviewRequest) error

	// Consultation
	GetConsultation(ctx context.Context, userID uint, apptID uint) (*entity.Consultation, []*entity.Prescription, error)
	GetMessages(ctx context.Context, userID uint, apptID uint) ([]*entity.ConsultationMessage, error)
	SendMessage(ctx context.Context, userID uint, role entity.Role, apptID uint, req *dto.SendMessageRequest) (*entity.ConsultationMessage, error)
	SaveNotes(ctx context.Context, userID uint, apptID uint, req *dto.ConsultationNotesRequest) (*entity.Consultation, error)
	AddPrescription(ctx context.Context, userID uint, apptID uint, req *dto.AddPrescriptionRequest) (*entity.Prescription, error)
	GetPrescriptions(ctx context.Context, userID uint, apptID uint) ([]*entity.Prescription, error)
}

type appointmentService struct {
	apptRepo    repository.AppointmentRepository
	consultRepo repository.ConsultationRepository
	prescRepo   repository.PrescriptionRepository
	reviewRepo  repository.ReviewRepository
	patientRepo repository.PatientRepository
	doctorRepo  repository.DoctorRepository
	walletRepo  repository.WalletRepository
	txRepo      repository.TransactionRepository
	notifRepo   repository.NotificationRepository
}

func NewAppointmentService(
	apptRepo repository.AppointmentRepository,
	consultRepo repository.ConsultationRepository,
	prescRepo repository.PrescriptionRepository,
	reviewRepo repository.ReviewRepository,
	patientRepo repository.PatientRepository,
	doctorRepo repository.DoctorRepository,
	walletRepo repository.WalletRepository,
	txRepo repository.TransactionRepository,
	notifRepo repository.NotificationRepository,
) AppointmentService {
	return &appointmentService{
		apptRepo:    apptRepo,
		consultRepo: consultRepo,
		prescRepo:   prescRepo,
		reviewRepo:  reviewRepo,
		patientRepo: patientRepo,
		doctorRepo:  doctorRepo,
		walletRepo:  walletRepo,
		txRepo:      txRepo,
		notifRepo:   notifRepo,
	}
}

// ─── Appointment: Book ────────────────────────────────────────────────────────

func (s *appointmentService) Book(ctx context.Context, userID uint, req *dto.BookAppointmentRequest) (*dto.AppointmentBookedResponse, error) {
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrPatientNotFound
	}

	doctor, err := s.doctorRepo.FindByID(ctx, req.DoctorID)
	if err != nil {
		return nil, pkgerrors.ErrDoctorNotFound
	}
	if doctor.Status != entity.DoctorStatusVerified {
		return nil, pkgerrors.ErrDoctorNotVerified
	}
	if !doctor.IsAvailable {
		return nil, pkgerrors.ErrDoctorUnavailable
	}

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		return nil, pkgerrors.ErrInvalidScheduleFormat
	}

	conflict, err := s.apptRepo.FindConflict(ctx, doctor.ID, scheduledAt)
	if err != nil {
		return nil, pkgerrors.ErrInternalServer
	}
	if conflict != nil {
		return nil, pkgerrors.ErrAppointmentConflict
	}

	wallet, err := s.walletRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, pkgerrors.ErrWalletNotFound
	}
	if !wallet.IsActive {
		return nil, pkgerrors.ErrWalletInactive
	}
	fee := doctor.ConsultationFee
	if wallet.Balance < fee {
		return nil, pkgerrors.ErrInsufficientFunds
	}

	// Find doctor wallet
	doctorWallet, err := s.walletRepo.FindByUserID(ctx, doctor.UserID)
	if err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	// Debit patient → credit doctor immediately (no escrow)
	if err := s.walletRepo.UpdateBalance(ctx, wallet.ID, -fee); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}
	if err := s.walletRepo.UpdateBalance(ctx, doctorWallet.ID, fee); err != nil {
		_ = s.walletRepo.UpdateBalance(ctx, wallet.ID, fee) // rollback patient debit
		return nil, pkgerrors.ErrInternalServer
	}

	// Record patient debit
	ref := utils.GenerateReference("PAY")
	tx := &entity.Transaction{
		Reference:     ref,
		WalletID:      wallet.ID,
		Type:          entity.TxTypePayment,
		Amount:        fee,
		BalanceBefore: wallet.Balance,
		BalanceAfter:  wallet.Balance - fee,
		Status:        entity.TxStatusSuccess,
		Description:   fmt.Sprintf("Consultation fee — Dr. %s %s", doctor.User.FirstName, doctor.User.LastName),
	}
	_ = s.txRepo.Create(ctx, tx)

	// Record doctor credit
	_ = s.txRepo.Create(ctx, &entity.Transaction{
		Reference:     utils.GenerateReference("CRD"),
		WalletID:      doctorWallet.ID,
		Type:          entity.TxTypeConsultationCredit,
		Amount:        fee,
		BalanceBefore: doctorWallet.Balance,
		BalanceAfter:  doctorWallet.Balance + fee,
		Status:        entity.TxStatusSuccess,
		Description:   fmt.Sprintf("Consultation fee from %s %s", patient.User.FirstName, patient.User.LastName),
	})

	appt := &entity.Appointment{
		PatientID:       patient.ID,
		DoctorID:        doctor.ID,
		Type:            entity.AppointmentType(req.Type),
		Status:          entity.AppointmentStatusPending,
		ScheduledAt:     scheduledAt,
		ConsultationFee: fee,
		TransactionID:   tx.ID,
		ChiefComplaint:  req.ChiefComplaint,
	}
	if err := s.apptRepo.Create(ctx, appt); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}

	// Notify doctor
	_ = s.notifRepo.Create(ctx, &entity.Notification{
		UserID:  doctor.UserID,
		Title:   "New Appointment Booked",
		Body:    fmt.Sprintf("%s %s booked a %s consultation for %s", patient.User.FirstName, patient.User.LastName, req.Type, scheduledAt.Format("Jan 2, 2006 at 3:04 PM")),
		Type:    entity.NotifTypeAppointment,
		Channel: entity.ChannelInApp,
	})

	// Reload appointment with associations
	appt, _ = s.apptRepo.FindByID(ctx, appt.ID)

	return &dto.AppointmentBookedResponse{
		Appointment: buildApptResponse(appt),
		Transaction: dto.TransactionResponse{
			ID:        tx.ID,
			Reference: tx.Reference,
			Type:      tx.Type,
			Amount:    tx.Amount,
			Status:    tx.Status,
		},
		Message: "Appointment booked. Consultation fee has been paid to the doctor.",
	}, nil
}

// ─── Appointment: Get ─────────────────────────────────────────────────────────

func (s *appointmentService) GetByID(ctx context.Context, userID uint, role entity.Role, apptID uint) (*entity.Appointment, error) {
	appt, err := s.apptRepo.FindByID(ctx, apptID)
	if err != nil {
		return nil, err
	}
	if err := s.assertParticipant(ctx, userID, role, appt); err != nil {
		return nil, err
	}
	return appt, nil
}

// ─── Appointment: List ────────────────────────────────────────────────────────

func (s *appointmentService) List(ctx context.Context, userID uint, role entity.Role, p pagination.Params) ([]*entity.Appointment, int64, error) {
	if role == entity.RoleDoctor {
		doctor, err := s.doctorRepo.FindByUserID(ctx, userID)
		if err != nil {
			return nil, 0, err
		}
		return s.apptRepo.ListByDoctor(ctx, doctor.ID, p)
	}
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return s.apptRepo.ListByPatient(ctx, patient.ID, p)
}

// ─── Appointment: Cancel ──────────────────────────────────────────────────────

func (s *appointmentService) Cancel(ctx context.Context, userID uint, role entity.Role, apptID uint, reason string) error {
	appt, err := s.apptRepo.FindByID(ctx, apptID)
	if err != nil {
		return err
	}
	if err := s.assertParticipant(ctx, userID, role, appt); err != nil {
		return err
	}
	if appt.Status != entity.AppointmentStatusPending && appt.Status != entity.AppointmentStatusConfirmed {
		return pkgerrors.ErrAppointmentNotPending
	}

	// Refund patient: debit doctor wallet, credit patient wallet
	patientWallet, pErr := s.walletRepo.FindByUserID(ctx, appt.Patient.UserID)
	docWallet, dErr := s.walletRepo.FindByUserID(ctx, appt.Doctor.UserID)
	if pErr == nil && dErr == nil {
		fee := appt.ConsultationFee
		_ = s.walletRepo.UpdateBalance(ctx, docWallet.ID, -fee)
		_ = s.walletRepo.UpdateBalance(ctx, patientWallet.ID, fee)
		_ = s.txRepo.Create(ctx, &entity.Transaction{
			Reference:     utils.GenerateReference("REF"),
			WalletID:      patientWallet.ID,
			Type:          entity.TxTypeRefund,
			Amount:        fee,
			BalanceBefore: patientWallet.Balance,
			BalanceAfter:  patientWallet.Balance + fee,
			Status:        entity.TxStatusSuccess,
			Description:   "Consultation fee refunded — appointment cancelled",
		})
		_ = s.txRepo.Create(ctx, &entity.Transaction{
			Reference:     utils.GenerateReference("DRF"),
			WalletID:      docWallet.ID,
			Type:          entity.TxTypeRefund,
			Amount:        fee,
			BalanceBefore: docWallet.Balance,
			BalanceAfter:  docWallet.Balance - fee,
			Status:        entity.TxStatusSuccess,
			Description:   "Refund issued — appointment cancelled",
		})
	}

	appt.Status = entity.AppointmentStatusCancelled
	appt.CancelReason = reason
	if err := s.apptRepo.Update(ctx, appt); err != nil {
		return pkgerrors.ErrInternalServer
	}

	// Notify the other party
	otherUserID := appt.Doctor.UserID
	notifMsg := fmt.Sprintf("Your appointment scheduled for %s has been cancelled.", appt.ScheduledAt.Format("Jan 2, 2006 at 3:04 PM"))
	if role == entity.RoleDoctor {
		otherUserID = appt.Patient.UserID
	}
	_ = s.notifRepo.Create(ctx, &entity.Notification{
		UserID:  otherUserID,
		Title:   "Appointment Cancelled",
		Body:    notifMsg,
		Type:    entity.NotifTypeAppointment,
		Channel: entity.ChannelInApp,
	})

	return nil
}

// ─── Appointment: Start ───────────────────────────────────────────────────────

func (s *appointmentService) Start(ctx context.Context, userID uint, apptID uint) error {
	appt, err := s.apptRepo.FindByID(ctx, apptID)
	if err != nil {
		return err
	}

	doctor, err := s.doctorRepo.FindByUserID(ctx, userID)
	if err != nil || doctor.ID != appt.DoctorID {
		return pkgerrors.ErrForbidden
	}
	if appt.Status != entity.AppointmentStatusPending && appt.Status != entity.AppointmentStatusConfirmed {
		return pkgerrors.ErrAppointmentNotPending
	}

	now := time.Now()
	appt.Status = entity.AppointmentStatusInProgress
	appt.StartedAt = &now
	if err := s.apptRepo.Update(ctx, appt); err != nil {
		return pkgerrors.ErrInternalServer
	}

	// Create consultation record
	_, err = s.consultRepo.FindByAppointmentID(ctx, appt.ID)
	if err == pkgerrors.ErrConsultationNotFound {
		_ = s.consultRepo.Create(ctx, &entity.Consultation{AppointmentID: appt.ID})
	}

	_ = s.notifRepo.Create(ctx, &entity.Notification{
		UserID:  appt.Patient.UserID,
		Title:   "Consultation Started",
		Body:    fmt.Sprintf("Dr. %s %s has started your consultation.", appt.Doctor.User.FirstName, appt.Doctor.User.LastName),
		Type:    entity.NotifTypeAppointment,
		Channel: entity.ChannelInApp,
	})

	return nil
}

// ─── Appointment: Complete ────────────────────────────────────────────────────

func (s *appointmentService) Complete(ctx context.Context, userID uint, apptID uint) error {
	appt, err := s.apptRepo.FindByID(ctx, apptID)
	if err != nil {
		return err
	}

	doctor, err := s.doctorRepo.FindByUserID(ctx, userID)
	if err != nil || doctor.ID != appt.DoctorID {
		return pkgerrors.ErrForbidden
	}
	if appt.Status != entity.AppointmentStatusInProgress {
		return pkgerrors.ErrAppointmentNotInProgress
	}

	// Payment was already transferred at booking — nothing to do here

	now := time.Now()
	appt.Status = entity.AppointmentStatusCompleted
	appt.CompletedAt = &now
	if err := s.apptRepo.Update(ctx, appt); err != nil {
		return pkgerrors.ErrInternalServer
	}

	_ = s.doctorRepo.IncrementConsultations(ctx, doctor.ID)

	_ = s.notifRepo.Create(ctx, &entity.Notification{
		UserID:  appt.Patient.UserID,
		Title:   "Consultation Completed",
		Body:    fmt.Sprintf("Your consultation with Dr. %s %s is complete. How was your experience?", doctor.User.FirstName, doctor.User.LastName),
		Type:    entity.NotifTypeAppointment,
		Channel: entity.ChannelInApp,
	})

	return nil
}

// ─── Appointment: Review ──────────────────────────────────────────────────────

func (s *appointmentService) LeaveReview(ctx context.Context, userID uint, apptID uint, req *dto.CreateReviewRequest) error {
	appt, err := s.apptRepo.FindByID(ctx, apptID)
	if err != nil {
		return err
	}
	if appt.Status != entity.AppointmentStatusCompleted {
		return pkgerrors.ErrCompletedOnly
	}

	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil || patient.ID != appt.PatientID {
		return pkgerrors.ErrForbidden
	}

	_, err = s.reviewRepo.FindByAppointmentID(ctx, apptID)
	if err == nil {
		return pkgerrors.ErrReviewExists
	}

	review := &entity.Review{
		PatientID:     patient.ID,
		DoctorID:      appt.DoctorID,
		AppointmentID: apptID,
		Rating:        req.Rating,
		Comment:       req.Comment,
		IsVisible:     true,
	}
	if err := s.reviewRepo.Create(ctx, review); err != nil {
		return pkgerrors.ErrInternalServer
	}

	// Recalculate doctor rating
	avg, count, err := s.reviewRepo.AverageRatingByDoctor(ctx, appt.DoctorID)
	if err == nil {
		_ = s.doctorRepo.UpdateRating(ctx, appt.DoctorID, avg, count)
	}

	return nil
}

// ─── Consultation ─────────────────────────────────────────────────────────────

func (s *appointmentService) GetConsultation(ctx context.Context, userID uint, apptID uint) (*entity.Consultation, []*entity.Prescription, error) {
	appt, err := s.apptRepo.FindByID(ctx, apptID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.assertParticipantByID(ctx, userID, appt); err != nil {
		return nil, nil, err
	}

	consult, err := s.consultRepo.FindByAppointmentID(ctx, apptID)
	if err != nil {
		return nil, nil, err
	}

	prescriptions, _ := s.prescRepo.ListByConsultation(ctx, consult.ID)
	return consult, prescriptions, nil
}

func (s *appointmentService) GetMessages(ctx context.Context, userID uint, apptID uint) ([]*entity.ConsultationMessage, error) {
	appt, err := s.apptRepo.FindByID(ctx, apptID)
	if err != nil {
		return nil, err
	}
	if err := s.assertParticipantByID(ctx, userID, appt); err != nil {
		return nil, err
	}
	msgs, err := s.consultRepo.ListMessages(ctx, apptID)
	if err != nil {
		return nil, err
	}
	_ = s.consultRepo.MarkMessagesRead(ctx, apptID, userID)
	return msgs, nil
}

func (s *appointmentService) SendMessage(ctx context.Context, userID uint, role entity.Role, apptID uint, req *dto.SendMessageRequest) (*entity.ConsultationMessage, error) {
	appt, err := s.apptRepo.FindByID(ctx, apptID)
	if err != nil {
		return nil, err
	}
	if err := s.assertParticipantByID(ctx, userID, appt); err != nil {
		return nil, err
	}
	if appt.Status != entity.AppointmentStatusInProgress {
		return nil, pkgerrors.ErrConsultationInactive
	}

	msgType := req.MessageType
	if msgType == "" {
		msgType = "text"
	}

	msg := &entity.ConsultationMessage{
		AppointmentID: apptID,
		SenderID:      userID,
		SenderRole:    role,
		Message:       req.Message,
		MessageType:   msgType,
	}
	if err := s.consultRepo.CreateMessage(ctx, msg); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}
	return msg, nil
}

func (s *appointmentService) SaveNotes(ctx context.Context, userID uint, apptID uint, req *dto.ConsultationNotesRequest) (*entity.Consultation, error) {
	appt, err := s.apptRepo.FindByID(ctx, apptID)
	if err != nil {
		return nil, err
	}

	doctor, err := s.doctorRepo.FindByUserID(ctx, userID)
	if err != nil || doctor.ID != appt.DoctorID {
		return nil, pkgerrors.ErrForbidden
	}

	consult, err := s.consultRepo.FindByAppointmentID(ctx, apptID)
	if err != nil {
		return nil, err
	}

	consult.DoctorNotes = req.DoctorNotes
	if req.Diagnosis != "" {
		consult.Diagnosis = req.Diagnosis
	}
	if req.Treatment != "" {
		consult.Treatment = req.Treatment
	}
	if req.FollowUpDate != "" {
		if t, err := time.Parse("2006-01-02", req.FollowUpDate); err == nil {
			consult.FollowUpDate = &t
		}
	}
	if req.FollowUpNotes != "" {
		consult.FollowUpNotes = req.FollowUpNotes
	}

	if err := s.consultRepo.Update(ctx, consult); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}
	return consult, nil
}

func (s *appointmentService) AddPrescription(ctx context.Context, userID uint, apptID uint, req *dto.AddPrescriptionRequest) (*entity.Prescription, error) {
	appt, err := s.apptRepo.FindByID(ctx, apptID)
	if err != nil {
		return nil, err
	}

	doctor, err := s.doctorRepo.FindByUserID(ctx, userID)
	if err != nil || doctor.ID != appt.DoctorID {
		return nil, pkgerrors.ErrForbidden
	}

	consult, err := s.consultRepo.FindByAppointmentID(ctx, apptID)
	if err != nil {
		return nil, err
	}

	p := &entity.Prescription{
		ConsultationID: consult.ID,
		PatientID:      appt.PatientID,
		DoctorID:       doctor.ID,
		MedicineName:   req.MedicineName,
		Dosage:         req.Dosage,
		Frequency:      req.Frequency,
		Duration:       req.Duration,
		Instructions:   req.Instructions,
	}
	if err := s.prescRepo.Create(ctx, p); err != nil {
		return nil, pkgerrors.ErrInternalServer
	}
	return p, nil
}

func (s *appointmentService) GetPrescriptions(ctx context.Context, userID uint, apptID uint) ([]*entity.Prescription, error) {
	appt, err := s.apptRepo.FindByID(ctx, apptID)
	if err != nil {
		return nil, err
	}
	if err := s.assertParticipantByID(ctx, userID, appt); err != nil {
		return nil, err
	}

	consult, err := s.consultRepo.FindByAppointmentID(ctx, apptID)
	if err != nil {
		return nil, err
	}
	return s.prescRepo.ListByConsultation(ctx, consult.ID)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// assertParticipant checks that a JWT-authenticated user owns a side of the appointment.
func (s *appointmentService) assertParticipant(ctx context.Context, userID uint, role entity.Role, appt *entity.Appointment) error {
	if role == entity.RoleDoctor {
		doctor, err := s.doctorRepo.FindByUserID(ctx, userID)
		if err != nil || doctor.ID != appt.DoctorID {
			return pkgerrors.ErrForbidden
		}
		return nil
	}
	patient, err := s.patientRepo.FindByUserID(ctx, userID)
	if err != nil || patient.ID != appt.PatientID {
		return pkgerrors.ErrForbidden
	}
	return nil
}

// assertParticipantByID checks by comparing userID directly to stored UserIDs.
func (s *appointmentService) assertParticipantByID(ctx context.Context, userID uint, appt *entity.Appointment) error {
	if appt.Patient.UserID == userID || appt.Doctor.UserID == userID {
		return nil
	}
	return pkgerrors.ErrForbidden
}

func buildApptResponse(a *entity.Appointment) dto.AppointmentResponse {
	return dto.AppointmentResponse{
		ID:              a.ID,
		Patient:         buildPatientProfileResponse(&a.Patient),
		Doctor:          buildDoctorProfileResponse(&a.Doctor),
		Type:            a.Type,
		Status:          a.Status,
		ScheduledAt:     a.ScheduledAt,
		StartedAt:       a.StartedAt,
		CompletedAt:     a.CompletedAt,
		ConsultationFee: a.ConsultationFee,
		ChiefComplaint:  a.ChiefComplaint,
		Notes:           a.Notes,
		CreatedAt:       a.CreatedAt,
	}
}
