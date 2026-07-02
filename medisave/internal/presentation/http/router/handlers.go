package router

import "github.com/medisave/app/internal/presentation/http/handler"

// Handlers holds every HTTP handler, fully initialized with their dependencies.
// Populated by the DI container and passed to RegisterAll.
type Handlers struct {
	Page         *handler.PageHandler
	Auth         *handler.AuthHandler
	Patient      *handler.PatientHandler
	Doctor       *handler.DoctorHandler
	Wallet       *handler.WalletHandler
	Appointment  *handler.AppointmentHandler
	Consultation *handler.ConsultationHandler
	Record       *handler.MedicalRecordHandler
	AI           *handler.AIHandler
	Emergency    *handler.EmergencyHandler
	Reminder     *handler.ReminderHandler
	Maps         *handler.MapsHandler
	SMS          *handler.SMSHandler
	USSD         *handler.USSDHandler
	Admin        *handler.AdminHandler
	Call         *handler.CallHandler
	Room         *handler.RoomHandler
}
