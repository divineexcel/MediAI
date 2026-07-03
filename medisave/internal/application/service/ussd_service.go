package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/medisave/app/internal/application/dto"
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/domain/repository"
	mapsclient "github.com/medisave/app/internal/infrastructure/external/maps"
	"github.com/medisave/app/pkg/pagination"
)

// ─── MENU STATE CONSTANTS ────────────────────────────────────────────────────

const (
	stateHome           = "home"
	stateAISymptoms     = "ai_symptoms"
	stateAIDuration     = "ai_duration"
	stateAISeverity     = "ai_severity"
	stateAIResult       = "ai_result"
	stateBookSpecialty  = "book_specialty"
	stateBookList       = "book_list"
	stateBookConfirm    = "book_confirm"
	stateBookTime       = "book_time"
	stateWalletMenu     = "wallet_menu"
	stateWalletTxns     = "wallet_txns"
	stateWalletSavings  = "wallet_savings"
	stateSOSConfirm     = "sos_confirm"
	stateReminders      = "reminders_today"
	stateReminderAction = "reminder_action"
	stateApptMenu       = "appt_menu"
	stateApptList       = "appt_list"
	stateApptCancel     = "appt_cancel"
	ussdTimeout         = 3 * time.Minute
)

// ─── SESSION DATA BLOB ───────────────────────────────────────────────────────

// ussdSessionData is serialised as JSON into ussd_sessions.data.
// It holds all transient menu state between gateway callbacks.
type ussdSessionData struct {
	// AI Symptom Checker
	Symptom  string `json:"symptom,omitempty"`
	Duration string `json:"duration,omitempty"`
	Severity string `json:"severity,omitempty"`
	AIAdvice string `json:"ai_advice,omitempty"`
	AIRisk   string `json:"ai_risk,omitempty"`

	// Doctor Booking
	Specialty  string  `json:"specialty,omitempty"`
	DoctorID   uint    `json:"doctor_id,omitempty"`
	DoctorName string  `json:"doctor_name,omitempty"`
	DoctorFee  float64 `json:"doctor_fee,omitempty"`
	DoctorPage int     `json:"doctor_page"`
	TimeSlot1  string  `json:"ts1,omitempty"` // RFC3339
	TimeSlot2  string  `json:"ts2,omitempty"`
	TimeSlot3  string  `json:"ts3,omitempty"`

	// Appointments
	ApptIDs    []uint `json:"appt_ids,omitempty"`
	ApptFilter string `json:"appt_filter,omitempty"` // "pending" | "completed"

	// Reminders
	ReminderIDs []uint `json:"reminder_ids,omitempty"`
	SelectedID  uint   `json:"selected_id,omitempty"`
}

func parseData(s string) *ussdSessionData {
	var d ussdSessionData
	_ = json.Unmarshal([]byte(s), &d)
	return &d
}

func marshalData(d *ussdSessionData) string {
	b, _ := json.Marshal(d)
	return string(b)
}

// ─── ADVICE TABLE ────────────────────────────────────────────────────────────

type symptomAdvice struct {
	possible string
	action   string
	risk     string // "LOW" | "MODERATE" | "HIGH"
}

var adviceTable = map[string]symptomAdvice{
	"Fever/Headache": {"Malaria or Typhoid", "See doctor within 24h", "MODERATE"},
	"Stomach Pain":   {"Gastritis/Food Poisoning", "Avoid heavy meals, rest", "LOW"},
	"Chest Pain":     {"Cardiac Emergency", "Call 112, go to ER NOW", "HIGH"},
	"Cough/Cold":     {"Viral Infection", "Rest, hydrate, honey+lemon", "LOW"},
	"Body Weakness":  {"Anemia or Malaria", "See doctor within 24h", "MODERATE"},
	"Other":          {"Unknown — needs evaluation", "See a doctor for diagnosis", "MODERATE"},
}

var stateCoords = map[string][2]float64{
	"Lagos":   {6.5244, 3.3792},
	"FCT":     {9.0765, 7.3986},
	"Abuja":   {9.0765, 7.3986},
	"Kano":    {12.0022, 8.5920},
	"Rivers":  {4.8156, 7.0498},
	"Enugu":   {6.4584, 7.5464},
	"Oyo":     {7.3776, 3.9470},
	"Edo":     {6.3350, 5.6268},
	"Anambra": {6.2104, 6.9742},
	"Kaduna":  {10.5264, 7.4382},
	"Ogun":    {6.9980, 3.4737},
	"Delta":   {5.5320, 5.8987},
	"Imo":     {5.4836, 7.0347},
}

// ─── INTERFACE ───────────────────────────────────────────────────────────────

type USSDService interface {
	Handle(ctx context.Context, req *dto.USSDRequest) (string, error)
}

// ─── SERVICE ─────────────────────────────────────────────────────────────────

type ussdService struct {
	sessionRepo  repository.USSDRepository
	userRepo     repository.UserRepository
	patientRepo  repository.PatientRepository
	doctorRepo   repository.DoctorRepository
	walletSvc    WalletService
	apptSvc      AppointmentService
	emergencySvc EmergencyService
	reminderSvc  ReminderService
	mapsClient   *mapsclient.Client
}

func NewUSSDService(
	sessionRepo repository.USSDRepository,
	userRepo repository.UserRepository,
	patientRepo repository.PatientRepository,
	doctorRepo repository.DoctorRepository,
	walletSvc WalletService,
	apptSvc AppointmentService,
	emergencySvc EmergencyService,
	reminderSvc ReminderService,
	mapsClient *mapsclient.Client,
) USSDService {
	return &ussdService{
		sessionRepo:  sessionRepo,
		userRepo:     userRepo,
		patientRepo:  patientRepo,
		doctorRepo:   doctorRepo,
		walletSvc:    walletSvc,
		apptSvc:      apptSvc,
		emergencySvc: emergencySvc,
		reminderSvc:  reminderSvc,
		mapsClient:   mapsClient,
	}
}

// ─── ENTRY POINT ─────────────────────────────────────────────────────────────

func (s *ussdService) Handle(ctx context.Context, req *dto.USSDRequest) (string, error) {
	sess, err := s.getOrCreateSession(ctx, req.SessionID, req.PhoneNumber)
	if err != nil {
		return end("Service unavailable. Try again."), nil
	}

	// Reset expired session state
	if time.Now().After(sess.ExpiresAt) {
		sess.MenuState = stateHome
		sess.Data = "{}"
	}

	// Look up registered user by phone number
	user, _ := s.userRepo.FindByPhone(ctx, req.PhoneNumber)
	var patient *entity.Patient
	if user != nil {
		patient, _ = s.patientRepo.FindByUserID(ctx, user.ID)
		sess.UserID = user.ID
	}

	data := parseData(sess.Data)
	input := latestInput(req.Text)

	// "00" always goes to main menu from any state
	if input == "00" {
		sess.MenuState = stateHome
		*data = ussdSessionData{}
	}

	response := s.dispatch(ctx, sess, data, input, user, patient)

	sess.Data = marshalData(data)
	sess.ExpiresAt = time.Now().Add(ussdTimeout)
	_ = s.sessionRepo.Upsert(ctx, sess)

	return response, nil
}

func (s *ussdService) getOrCreateSession(ctx context.Context, sessionID, phone string) (*entity.USSDSession, error) {
	sess, err := s.sessionRepo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess != nil {
		return sess, nil
	}
	sess = &entity.USSDSession{
		SessionID: sessionID,
		Phone:     phone,
		MenuState: stateHome,
		Data:      "{}",
		ExpiresAt: time.Now().Add(ussdTimeout),
	}
	return sess, nil
}

// ─── DISPATCH ────────────────────────────────────────────────────────────────

func (s *ussdService) dispatch(ctx context.Context, sess *entity.USSDSession, data *ussdSessionData, input string, user *entity.User, patient *entity.Patient) string {
	switch sess.MenuState {
	case stateHome:
		return s.handleHome(ctx, sess, data, input, user, patient)
	case stateAISymptoms:
		return s.handleAISymptoms(sess, data, input)
	case stateAIDuration:
		return s.handleAIDuration(sess, data, input)
	case stateAISeverity:
		return s.handleAISeverity(sess, data, input)
	case stateAIResult:
		return s.handleAIResult(ctx, sess, data, input, user, patient)
	case stateBookSpecialty:
		return s.handleBookSpecialty(ctx, sess, data, input, patient)
	case stateBookList:
		return s.handleBookList(ctx, sess, data, input)
	case stateBookConfirm:
		return s.handleBookConfirm(ctx, sess, data, input, user, patient)
	case stateBookTime:
		return s.handleBookTime(ctx, sess, data, input, user)
	case stateWalletMenu:
		return s.handleWalletMenu(ctx, sess, data, input, user)
	case stateWalletTxns:
		return s.handleWalletTxns(sess, input)
	case stateWalletSavings:
		return s.handleWalletSavings(sess, input)
	case stateSOSConfirm:
		return s.handleSOSConfirm(ctx, sess, data, input, user, patient)
	case stateReminders:
		return s.handleReminders(ctx, sess, data, input, user)
	case stateReminderAction:
		return s.handleReminderAction(ctx, sess, data, input, user)
	case stateApptMenu:
		return s.handleApptMenu(ctx, sess, data, input, user)
	case stateApptList:
		return s.handleApptList(ctx, sess, data, input, user)
	case stateApptCancel:
		return s.handleApptCancel(ctx, sess, data, input, user)
	default:
		sess.MenuState = stateHome
		return s.showHome(patient)
	}
}

// ─── HOME MENU ───────────────────────────────────────────────────────────────

func (s *ussdService) handleHome(ctx context.Context, sess *entity.USSDSession, data *ussdSessionData, input string, user *entity.User, patient *entity.Patient) string {
	switch input {
	case "", "0":
		return s.showHome(patient)
	case "1":
		sess.MenuState = stateAISymptoms
		return showAISymptoms()
	case "2":
		if patient == nil {
			return needsRegistration(sess)
		}
		sess.MenuState = stateBookSpecialty
		return showBookSpecialty()
	case "3":
		if patient == nil {
			return needsRegistration(sess)
		}
		sess.MenuState = stateWalletMenu
		return s.renderWalletMenu(ctx, user)
	case "4":
		if patient == nil {
			return needsRegistration(sess)
		}
		sess.MenuState = stateSOSConfirm
		return showSOSConfirm()
	case "5":
		return s.renderNearby(ctx, patient)
	case "6":
		if patient == nil {
			return needsRegistration(sess)
		}
		sess.MenuState = stateReminders
		return s.renderReminders(ctx, data, user)
	case "7":
		if patient == nil {
			return needsRegistration(sess)
		}
		sess.MenuState = stateApptMenu
		return showApptMenu()
	case "8":
		return healthTip()
	case "9":
		return end("Thank you for using MediSave.\nYour health is our priority.\n\nDial *384*123# anytime.\nEmergency: 112")
	default:
		return invalid(sess, stateHome, s.showHome(patient))
	}
}

func (s *ussdService) showHome(patient *entity.Patient) string {
	greeting := "Welcome to MediSave"
	if patient != nil && patient.User.FirstName != "" {
		greeting = "Hi " + patient.User.FirstName + "!"
	}
	return con(greeting + "\n1. AI Symptoms\n2. Book Doctor\n3. Wallet\n4. Emergency SOS\n5. Nearby Hospitals\n6. Reminders\n7. Appointments\n8. Health Tips\n9. Exit\n00. Main Menu")
}

// ─── AI SYMPTOM CHECKER ──────────────────────────────────────────────────────

var symptoms = []string{
	"Fever/Headache",
	"Stomach Pain",
	"Chest Pain",
	"Cough/Cold",
	"Body Weakness",
	"Other",
}

var durations = []string{
	"Today",
	"2-3 days",
	"4-7 days",
	"Over a week",
}

var severities = []string{
	"Mild",
	"Moderate",
	"Severe",
}

func showAISymptoms() string {
	return con("Select main symptom:\n1. Fever/Headache\n2. Stomach Pain\n3. Chest Pain\n4. Cough/Cold\n5. Body Weakness\n6. Other\n0. Back")
}

func (s *ussdService) handleAISymptoms(sess *entity.USSDSession, data *ussdSessionData, input string) string {
	if input == "0" {
		sess.MenuState = stateHome
		return s.showHome(nil)
	}
	idx := toInt(input) - 1
	if idx < 0 || idx >= len(symptoms) {
		return invalid(sess, stateAISymptoms, showAISymptoms())
	}
	data.Symptom = symptoms[idx]
	sess.MenuState = stateAIDuration
	return con(fmt.Sprintf("Symptom: %s\n\nHow long?\n1. Started today\n2. 2-3 days\n3. 4-7 days\n4. Over a week\n0. Back", data.Symptom))
}

func (s *ussdService) handleAIDuration(sess *entity.USSDSession, data *ussdSessionData, input string) string {
	if input == "0" {
		sess.MenuState = stateAISymptoms
		return showAISymptoms()
	}
	idx := toInt(input) - 1
	if idx < 0 || idx >= len(durations) {
		return invalid(sess, stateAIDuration, con("Duration?\n1. Today\n2. 2-3 days\n3. 4-7 days\n4. Over a week\n0. Back"))
	}
	data.Duration = durations[idx]
	sess.MenuState = stateAISeverity
	return con("How severe?\n1. Mild (bearable)\n2. Moderate (worrying)\n3. Severe (very bad)\n0. Back")
}

func (s *ussdService) handleAISeverity(sess *entity.USSDSession, data *ussdSessionData, input string) string {
	if input == "0" {
		sess.MenuState = stateAIDuration
		return con(fmt.Sprintf("Symptom: %s\n\nHow long?\n1. Today\n2. 2-3 days\n3. 4-7 days\n4. Over a week\n0. Back", data.Symptom))
	}
	idx := toInt(input) - 1
	if idx < 0 || idx >= len(severities) {
		return invalid(sess, stateAISeverity, con("How severe?\n1. Mild\n2. Moderate\n3. Severe\n0. Back"))
	}
	data.Severity = severities[idx]

	adv, ok := adviceTable[data.Symptom]
	if !ok {
		adv = adviceTable["Other"]
	}
	risk := adv.risk
	if data.Duration == "4-7 days" || data.Duration == "Over a week" {
		if risk == "LOW" {
			risk = "MODERATE"
		}
	}
	if data.Severity == "Severe" {
		if risk == "LOW" || risk == "MODERATE" {
			risk = "HIGH"
		}
	}
	data.AIAdvice = adv.action
	data.AIRisk = risk
	sess.MenuState = stateAIResult

	return s.renderAIResult(data)
}

func (s *ussdService) handleAIResult(ctx context.Context, sess *entity.USSDSession, data *ussdSessionData, input string, _ *entity.User, patient *entity.Patient) string {
	switch input {
	case "0":
		sess.MenuState = stateHome
		return s.showHome(patient)
	case "1":
		if patient == nil {
			return needsRegistration(sess)
		}
		sess.MenuState = stateBookSpecialty
		return showBookSpecialty()
	case "2":
		return s.renderNearby(ctx, patient)
	case "3":
		if patient == nil {
			return needsRegistration(sess)
		}
		sess.MenuState = stateSOSConfirm
		return showSOSConfirm()
	default:
		return s.renderAIResult(data)
	}
}

func (s *ussdService) renderAIResult(data *ussdSessionData) string {
	adv := adviceTable[data.Symptom]
	menu := fmt.Sprintf("MediSave Assessment:\n\nPossible: %s\nAction: %s\nRisk: %s\n\n*Not medical advice*", adv.possible, data.AIAdvice, data.AIRisk)

	switch data.AIRisk {
	case "HIGH":
		menu += "\n\n1. Emergency SOS\n2. Nearby Hospital\n0. Back"
	case "MODERATE":
		menu += "\n\n1. Book Doctor\n2. Nearby Hospital\n0. Back"
	default:
		menu += "\n\n0. Main Menu"
	}
	return con(menu)
}

// ─── DOCTOR BOOKING ──────────────────────────────────────────────────────────

var specialties = []string{
	"General Practice",
	"Paediatrics",
	"Gynaecology",
	"Cardiology",
	"Dermatology",
	"Internal Medicine",
}

func showBookSpecialty() string {
	return con("Select specialty:\n1. General Practice\n2. Paediatrics\n3. Gynaecology\n4. Cardiology\n5. Dermatology\n6. Internal Medicine\n0. Back")
}

func (s *ussdService) handleBookSpecialty(ctx context.Context, sess *entity.USSDSession, data *ussdSessionData, input string, patient *entity.Patient) string {
	if input == "0" {
		sess.MenuState = stateHome
		return s.showHome(patient)
	}
	idx := toInt(input) - 1
	if idx < 0 || idx >= len(specialties) {
		return invalid(sess, stateBookSpecialty, showBookSpecialty())
	}
	data.Specialty = specialties[idx]
	data.DoctorPage = 0
	sess.MenuState = stateBookList
	return s.renderDoctorList(ctx, sess, data)
}

func (s *ussdService) handleBookList(ctx context.Context, sess *entity.USSDSession, data *ussdSessionData, input string) string {
	switch input {
	case "0":
		sess.MenuState = stateBookSpecialty
		return showBookSpecialty()
	case "4": // next page
		data.DoctorPage++
		return s.renderDoctorList(ctx, sess, data)
	case "5": // prev page
		if data.DoctorPage > 0 {
			data.DoctorPage--
		}
		return s.renderDoctorList(ctx, sess, data)
	}

	doctors, _, _ := s.doctorRepo.ListAvailable(ctx, data.Specialty, pagination.Params{
		Page:   data.DoctorPage + 1,
		Limit:  3,
		Offset: data.DoctorPage * 3,
	})
	idx := toInt(input) - 1
	if idx < 0 || idx >= len(doctors) {
		return invalid(sess, stateBookList, s.renderDoctorList(ctx, sess, data))
	}
	d := doctors[idx]
	data.DoctorID = d.ID
	data.DoctorName = "Dr. " + d.User.FirstName + " " + d.User.LastName
	data.DoctorFee = d.ConsultationFee
	sess.MenuState = stateBookConfirm
	return s.renderBookConfirm(ctx, data)
}

func (s *ussdService) renderDoctorList(ctx context.Context, sess *entity.USSDSession, data *ussdSessionData) string {
	p := pagination.Params{Page: data.DoctorPage + 1, Limit: 3, Offset: data.DoctorPage * 3}
	doctors, total, _ := s.doctorRepo.ListAvailable(ctx, data.Specialty, p)
	if len(doctors) == 0 {
		sess.MenuState = stateBookSpecialty
		return con("No available doctors\nfor " + data.Specialty + " right now.\n\n0. Back")
	}

	var sb strings.Builder
	spec := data.Specialty
	if len(spec) > 10 {
		spec = spec[:10]
	}
	fmt.Fprintf(&sb, "Doctors (%s):\n", spec)
	for i, d := range doctors {
		fmt.Fprintf(&sb, "%d. Dr. %s %s\n   N%.0f\n", i+1, d.User.FirstName[:1], d.User.LastName, d.ConsultationFee)
	}
	hasMore := int64((data.DoctorPage+1)*3) < total
	if hasMore {
		sb.WriteString("4. Next page\n")
	}
	if data.DoctorPage > 0 {
		sb.WriteString("5. Prev page\n")
	}
	sb.WriteString("0. Back")
	return con(sb.String())
}

func (s *ussdService) renderBookConfirm(_ context.Context, data *ussdSessionData) string {
	// Fetch wallet balance for display
	return con(fmt.Sprintf("Confirm Booking:\n\n%s\n%s\nFee: N%.0f\n\n1. Confirm & Pay\n2. Choose Another\n0. Back", data.DoctorName, data.Specialty, data.DoctorFee))
}

func (s *ussdService) handleBookConfirm(ctx context.Context, sess *entity.USSDSession, data *ussdSessionData, input string, _ *entity.User, _ *entity.Patient) string {
	switch input {
	case "0":
		sess.MenuState = stateBookList
		return s.renderDoctorList(ctx, sess, data)
	case "1":
		sess.MenuState = stateBookTime
		return s.renderBookTime(data)
	case "2":
		sess.MenuState = stateBookList
		data.DoctorID = 0
		return s.renderDoctorList(ctx, sess, data)
	default:
		return s.renderBookConfirm(ctx, data)
	}
}

func (s *ussdService) renderBookTime(data *ussdSessionData) string {
	wat := time.FixedZone("WAT", 3600)
	now := time.Now().In(wat)

	// Slot 1: next available (current hour + 2, capped to 18:00)
	slot1 := now.Truncate(time.Hour).Add(2 * time.Hour)
	if slot1.Hour() >= 18 {
		slot1 = time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, wat)
	}
	slot2 := time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, wat)
	slot3 := time.Date(now.Year(), now.Month(), now.Day()+1, 14, 0, 0, 0, wat)

	data.TimeSlot1 = slot1.UTC().Format(time.RFC3339)
	data.TimeSlot2 = slot2.UTC().Format(time.RFC3339)
	data.TimeSlot3 = slot3.UTC().Format(time.RFC3339)

	label := func(t time.Time) string {
		return t.In(wat).Format("Jan 2, 3:04 PM")
	}
	return con(fmt.Sprintf("Select time:\n1. %s\n2. %s\n3. %s\n0. Back", label(slot1), label(slot2), label(slot3)))
}

func (s *ussdService) handleBookTime(ctx context.Context, sess *entity.USSDSession, data *ussdSessionData, input string, user *entity.User) string {
	if input == "0" {
		sess.MenuState = stateBookConfirm
		return s.renderBookConfirm(ctx, data)
	}
	slots := []string{data.TimeSlot1, data.TimeSlot2, data.TimeSlot3}
	idx := toInt(input) - 1
	if idx < 0 || idx >= len(slots) || slots[idx] == "" {
		return invalid(sess, stateBookTime, s.renderBookTime(data))
	}
	complaint := "USSD Consultation Request"
	if data.Symptom != "" {
		complaint = fmt.Sprintf("USSD: %s (%s, %s)", data.Symptom, data.Duration, data.Severity)
		if len(complaint) > 50 {
			complaint = complaint[:50]
		}
		// Pad to minimum 10 chars
		if len(complaint) < 10 {
			complaint = "USSD Consultation Request"
		}
	}
	req := &dto.BookAppointmentRequest{
		DoctorID:       data.DoctorID,
		Type:           "chat",
		ScheduledAt:    slots[idx],
		ChiefComplaint: complaint,
	}
	resp, err := s.apptSvc.Book(ctx, user.ID, req)
	if err != nil {
		sess.MenuState = stateHome
		errMsg := "Booking failed. Check your wallet balance or try again."
		if strings.Contains(err.Error(), "insufficient") {
			wallet, _ := s.walletSvc.GetWallet(ctx, user.ID)
			errMsg = fmt.Sprintf("Insufficient balance.\nBalance: N%.0f\nFee: N%.0f\n\nFund wallet at medisave.ng", wallet.Balance, data.DoctorFee)
		}
		return end(errMsg)
	}
	wat := time.FixedZone("WAT", 3600)
	return end(fmt.Sprintf("Booking Confirmed!\n\n%s\nID: #%d\nFee paid: N%.0f\n\nCheck reminders at\nmedisave.ng", data.DoctorName, resp.Appointment.ID, resp.Appointment.ConsultationFee) +
		"\nTime: " + resp.Appointment.ScheduledAt.In(wat).Format("Jan 2, 3PM"))
}

// ─── WALLET ──────────────────────────────────────────────────────────────────

func (s *ussdService) renderWalletMenu(ctx context.Context, user *entity.User) string {
	wallet, err := s.walletSvc.GetWallet(ctx, user.ID)
	if err != nil {
		return con("Health Wallet\n\nCould not load balance.\n\n1. Transactions\n2. Savings Goals\n3. Fund Wallet\n0. Back")
	}
	return con(fmt.Sprintf("Health Wallet\n\nBalance: N%.2f\n\n1. Transactions\n2. Savings Goals\n3. Fund Wallet\n0. Back", wallet.Balance))
}

func (s *ussdService) handleWalletMenu(ctx context.Context, sess *entity.USSDSession, _ *ussdSessionData, input string, user *entity.User) string {
	switch input {
	case "0":
		sess.MenuState = stateHome
		return s.showHome(nil)
	case "1":
		sess.MenuState = stateWalletTxns
		return s.renderWalletTxns(ctx, user)
	case "2":
		sess.MenuState = stateWalletSavings
		return s.renderWalletSavings(ctx, user)
	case "3":
		return end("Fund Your MediSave Wallet:\n\n1. Visit medisave.ng/patient/wallet\n2. Click 'Add Money'\n3. Use card or bank transfer\n\nMin deposit: N100\nFor help: 0700-MEDISAVE")
	default:
		return s.renderWalletMenu(ctx, user)
	}
}

func (s *ussdService) handleWalletTxns(sess *entity.USSDSession, input string) string {
	if input == "0" {
		sess.MenuState = stateWalletMenu
	}
	return "" // handled by dispatch after state change
}

func (s *ussdService) renderWalletTxns(ctx context.Context, user *entity.User) string {
	p := pagination.Params{Page: 1, Limit: 5, Offset: 0}
	txns, _, err := s.walletSvc.GetTransactions(ctx, user.ID, p)
	if err != nil || len(txns) == 0 {
		return con("No transactions yet.\n\n0. Back")
	}
	var sb strings.Builder
	sb.WriteString("Recent Transactions:\n")
	for _, tx := range txns {
		sign := "-"
		if tx.Type == entity.TxTypeDeposit || tx.Type == entity.TxTypeRefund || tx.Type == entity.TxTypeConsultationCredit {
			sign = "+"
		}
		sb.WriteString(fmt.Sprintf("\n%sN%.0f %s", sign, tx.Amount, shortTxType(tx.Type)))
	}
	sb.WriteString("\n\n0. Back")
	return con(sb.String())
}

func shortTxType(t entity.TransactionType) string {
	switch t {
	case entity.TxTypeDeposit:
		return "deposit"
	case entity.TxTypeWithdrawal:
		return "withdrawal"
	case entity.TxTypePayment:
		return "payment"
	case entity.TxTypeRefund:
		return "refund"
	case entity.TxTypeConsultationCredit:
		return "consult credit"
	case entity.TxTypeSavings:
		return "savings"
	default:
		return string(t)
	}
}

func (s *ussdService) handleWalletSavings(sess *entity.USSDSession, input string) string {
	if input == "0" {
		sess.MenuState = stateWalletMenu
	}
	return ""
}

func (s *ussdService) renderWalletSavings(ctx context.Context, user *entity.User) string {
	p := pagination.Params{Page: 1, Limit: 5, Offset: 0}
	goals, _, err := s.walletSvc.GetSavingsGoals(ctx, user.ID, p)
	if err != nil || len(goals) == 0 {
		return con("No savings goals yet.\n\nCreate one at\nmedisave.ng/patient/savings\n\n0. Back")
	}
	var sb strings.Builder
	sb.WriteString("Savings Goals:\n")
	for i, g := range goals {
		title := g.Title
		if len(title) > 15 {
			title = title[:15]
		}
		fmt.Fprintf(&sb, "\n%d. %s\n   N%.0f/N%.0f", i+1, title, g.SavedAmount, g.TargetAmount)
	}
	sb.WriteString("\n\n0. Back")
	return con(sb.String())
}

// ─── EMERGENCY SOS ───────────────────────────────────────────────────────────

func showSOSConfirm() string {
	return con("EMERGENCY SOS\n\nThis will alert your\nemergency contacts &\nnearest doctors.\n\n1. Confirm SOS\n0. Cancel")
}

func (s *ussdService) handleSOSConfirm(ctx context.Context, sess *entity.USSDSession, _ *ussdSessionData, input string, user *entity.User, patient *entity.Patient) string {
	switch input {
	case "0":
		sess.MenuState = stateHome
		return s.showHome(patient)
	case "1":
		lat, lng := 6.5244, 3.3792 // default: Lagos
		if patient != nil && patient.State != "" {
			if coords, ok := stateCoords[patient.State]; ok {
				lat, lng = coords[0], coords[1]
			}
		}
		_, err := s.emergencySvc.TriggerSOS(ctx, user.ID, &dto.SOSRequest{
			Latitude:    lat,
			Longitude:   lng,
			Description: "Emergency SOS triggered via USSD (*384*123#)",
		})
		sess.MenuState = stateHome
		if err != nil {
			return end("SOS sent!\n\nEmergency contacts notified.\nFor immediate help call 112.\n\nStay calm — help is coming.")
		}
		return end("SOS ACTIVATED!\n\nEmergency contacts notified.\nNearest hospitals alerted.\n\nDial 112 for ambulance.\nStay calm — help is coming.")
	default:
		return showSOSConfirm()
	}
}

// ─── NEARBY HOSPITALS ────────────────────────────────────────────────────────

func (s *ussdService) renderNearby(_ context.Context, patient *entity.Patient) string {
	lat, lng := 6.5244, 3.3792
	if patient != nil && patient.State != "" {
		if coords, ok := stateCoords[patient.State]; ok {
			lat, lng = coords[0], coords[1]
		}
	}
	places, err := s.mapsClient.FindNearby(lat, lng, "hospital", 10000)
	if err != nil || len(places) == 0 {
		return end("Nearby Hospitals:\n\n• LUTH Lagos: 08023032003\n• Garki Hospital Abuja: 09-2340005\n• AKTH Kano: 064-666601\n\nVisit medisave.ng/patient/nearby")
	}
	var sb strings.Builder
	sb.WriteString("Nearby Hospitals:\n")
	limit := 4
	if len(places) < limit {
		limit = len(places)
	}
	for _, p := range places[:limit] {
		name := p.Name
		if len(name) > 22 {
			name = name[:22]
		}
		sb.WriteString(fmt.Sprintf("\n• %s (%s)", name, p.Distance))
	}
	sb.WriteString("\n\nFor directions:\nmedisave.ng/patient/nearby")
	return end(sb.String())
}

// ─── MEDICATION REMINDERS ────────────────────────────────────────────────────

func (s *ussdService) renderReminders(ctx context.Context, data *ussdSessionData, user *entity.User) string {
	p := pagination.Params{Page: 1, Limit: 20, Offset: 0}
	reminders, _, err := s.reminderSvc.List(ctx, user.ID, p)
	if err != nil {
		return con("Could not load reminders.\n\n0. Back")
	}
	today := time.Now()
	var active []*dto.ReminderResponse
	for _, r := range reminders {
		if r.IsActive && !r.StartDate.After(today) && !r.EndDate.Before(today.Truncate(24*time.Hour)) {
			active = append(active, r)
			if len(active) >= 5 {
				break
			}
		}
	}
	if len(active) == 0 {
		return con("No medications due today.\n\nAdd reminders at\nmedisave.ng/patient/reminders\n\n0. Back")
	}
	data.ReminderIDs = nil
	var sb strings.Builder
	sb.WriteString("Today's Medications:\n")
	for i, r := range active {
		data.ReminderIDs = append(data.ReminderIDs, r.ID)
		name := r.MedicineName
		if len(name) > 16 {
			name = name[:16]
		}
		time_ := firstNonEmpty(r.MorningTime, r.AfternoonTime, r.NightTime)
		fmt.Fprintf(&sb, "\n%d. %s %s\n   %s", i+1, name, r.Dosage, time_)
	}
	sb.WriteString("\n\n0. Back")
	return con(sb.String())
}

func (s *ussdService) handleReminders(ctx context.Context, sess *entity.USSDSession, data *ussdSessionData, input string, user *entity.User) string {
	if input == "0" {
		sess.MenuState = stateHome
		return s.showHome(nil)
	}
	idx := toInt(input) - 1
	if idx < 0 || idx >= len(data.ReminderIDs) {
		return invalid(sess, stateReminders, s.renderReminders(ctx, data, user))
	}
	data.SelectedID = data.ReminderIDs[idx]
	sess.MenuState = stateReminderAction

	p := pagination.Params{Page: 1, Limit: 20, Offset: 0}
	reminders, _, _ := s.reminderSvc.List(ctx, user.ID, p)
	for _, r := range reminders {
		if r.ID == data.SelectedID {
			name := r.MedicineName
			if len(name) > 18 {
				name = name[:18]
			}
			t := firstNonEmpty(r.MorningTime, r.AfternoonTime, r.NightTime)
			return con(fmt.Sprintf("%s %s\nTime: %s\n\n1. Mark Taken\n2. Mark Missed\n0. Back", name, r.Dosage, t))
		}
	}
	return con("Select action:\n1. Mark Taken\n2. Mark Missed\n0. Back")
}

func (s *ussdService) handleReminderAction(ctx context.Context, sess *entity.USSDSession, data *ussdSessionData, input string, user *entity.User) string {
	if input == "0" {
		sess.MenuState = stateReminders
		return s.renderReminders(ctx, data, user)
	}
	var status string
	switch input {
	case "1":
		status = "taken"
	case "2":
		status = "skipped"
	default:
		return con("1. Mark Taken\n2. Mark Missed\n0. Back")
	}
	_ = s.reminderSvc.LogAction(ctx, user.ID, data.SelectedID, &dto.ReminderLogActionRequest{Status: status})
	sess.MenuState = stateHome
	action := "Taken"
	if status == "skipped" {
		action = "Missed"
	}
	return end(fmt.Sprintf("Medication Logged!\nStatus: %s\n\nAdherence is tracked at\nmedisave.ng/patient/reminders\n\nDial *384*123# for more.", action))
}

// ─── APPOINTMENTS ────────────────────────────────────────────────────────────

func showApptMenu() string {
	return con("My Appointments:\n\n1. Upcoming\n2. Completed\n0. Back")
}

func (s *ussdService) handleApptMenu(ctx context.Context, sess *entity.USSDSession, data *ussdSessionData, input string, user *entity.User) string {
	switch input {
	case "0":
		sess.MenuState = stateHome
		return s.showHome(nil)
	case "1":
		data.ApptFilter = "pending"
		sess.MenuState = stateApptList
		return s.renderApptList(ctx, data, user)
	case "2":
		data.ApptFilter = "completed"
		sess.MenuState = stateApptList
		return s.renderApptList(ctx, data, user)
	default:
		return showApptMenu()
	}
}

func (s *ussdService) renderApptList(ctx context.Context, data *ussdSessionData, user *entity.User) string {
	p := pagination.Params{Page: 1, Limit: 5, Offset: 0}
	appts, _, err := s.apptSvc.List(ctx, user.ID, entity.RolePatient, data.ApptFilter, p)
	if err != nil || len(appts) == 0 {
		label := "Upcoming"
		if data.ApptFilter == "completed" {
			label = "Completed"
		}
		return con(fmt.Sprintf("No %s appointments.\n\nBook via option 2\non main menu.\n\n0. Back", strings.ToLower(label)))
	}
	data.ApptIDs = nil
	wat := time.FixedZone("WAT", 3600)
	var sb strings.Builder
	label := "Upcoming"
	if data.ApptFilter == "completed" {
		label = "Completed"
	}
	fmt.Fprintf(&sb, "%s Appointments:\n", label)
	for i, a := range appts {
		data.ApptIDs = append(data.ApptIDs, a.ID)
		name := "Dr. " + a.Doctor.User.LastName
		if len(name) > 16 {
			name = name[:16]
		}
		fmt.Fprintf(&sb, "\n%d. %s\n   %s", i+1, name, a.ScheduledAt.In(wat).Format("Jan 2, 3PM"))
	}
	sb.WriteString("\n\n0. Back")
	return con(sb.String())
}

func (s *ussdService) handleApptList(ctx context.Context, sess *entity.USSDSession, data *ussdSessionData, input string, user *entity.User) string {
	if input == "0" {
		sess.MenuState = stateApptMenu
		return showApptMenu()
	}
	idx := toInt(input) - 1
	if idx < 0 || idx >= len(data.ApptIDs) {
		return invalid(sess, stateApptList, s.renderApptList(ctx, data, user))
	}
	data.SelectedID = data.ApptIDs[idx]
	sess.MenuState = stateApptCancel

	p := pagination.Params{Page: 1, Limit: 5, Offset: 0}
	appts, _, _ := s.apptSvc.List(ctx, user.ID, entity.RolePatient, data.ApptFilter, p)
	wat := time.FixedZone("WAT", 3600)
	for _, a := range appts {
		if a.ID == data.SelectedID {
			name := "Dr. " + a.Doctor.User.FirstName + " " + a.Doctor.User.LastName
			canCancel := a.Status == entity.AppointmentStatusPending || a.Status == entity.AppointmentStatusConfirmed
			if !canCancel {
				sess.MenuState = stateApptList
				return con(fmt.Sprintf("%s\n%s\nStatus: %s\n\nThis appointment cannot\nbe cancelled.\n\n0. Back", name, a.ScheduledAt.In(wat).Format("Jan 2, 3:00 PM"), a.Status))
			}
			return con(fmt.Sprintf("Cancel Appointment?\n\n%s\n%s\n\nRefund: N%.0f\n\n1. Confirm Cancel\n0. Back", name, a.ScheduledAt.In(wat).Format("Jan 2, 3:00 PM"), a.ConsultationFee))
		}
	}
	return con("Appointment details\nnot found.\n\n0. Back")
}

func (s *ussdService) handleApptCancel(ctx context.Context, sess *entity.USSDSession, data *ussdSessionData, input string, user *entity.User) string {
	switch input {
	case "0":
		sess.MenuState = stateApptList
		return s.renderApptList(ctx, data, user)
	case "1":
		err := s.apptSvc.Cancel(ctx, user.ID, entity.RolePatient, data.SelectedID, "Cancelled via USSD")
		sess.MenuState = stateHome
		if err != nil {
			return end("Could not cancel appointment.\nPlease try at medisave.ng or\ncall 0700-MEDISAVE.")
		}
		return end("Appointment cancelled.\nRefund processed to\nyour wallet.\n\nDial *384*123# for more.")
	default:
		return con("1. Confirm Cancel\n0. Back")
	}
}

// ─── HEALTH TIP ──────────────────────────────────────────────────────────────

func healthTip() string {
	tips := []string{
		"Drink 8 glasses of water daily.\nDehydration worsens many illnesses.",
		"Wash hands for 20 seconds\nwith soap to prevent infections.",
		"Exercise 30 mins daily — even\na brisk walk reduces disease risk.",
		"Sleep 7-8 hours. Poor sleep\nweakens immunity & raises BP.",
		"Malaria prevention: use treated\nnets & clear stagnant water.",
		"Check your blood pressure often.\nHypertension has no symptoms.",
		"Eat more fruits & vegetables\nto reduce cancer & heart disease.",
		"Limit sugar intake. Type 2\ndiabetes is largely preventable.",
		"Complete your antibiotic course\neven when you feel better.",
		"Register family members on\nMediSave for free health checks.",
	}
	tip := tips[time.Now().YearDay()%len(tips)]
	return end("MediSave Health Tip:\n\n" + tip + "\n\nFor personalised advice:\nmedisave.ng/patient/ai")
}

// ─── UTILITY HELPERS ─────────────────────────────────────────────────────────

func con(text string) string {
	return "CON " + text
}

func end(text string) string {
	return "END " + text
}

func latestInput(text string) string {
	if text == "" {
		return ""
	}
	parts := strings.Split(text, "*")
	return strings.TrimSpace(parts[len(parts)-1])
}

func toInt(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func needsRegistration(sess *entity.USSDSession) string {
	sess.MenuState = stateHome
	return end("This feature requires a\nMediSave account.\n\nRegister free at medisave.ng\nor open the MediSave app.\n\nDial *384*123# to try again.")
}

func invalid(sess *entity.USSDSession, currentState, menu string) string {
	sess.MenuState = currentState
	return menu
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return "—"
}
