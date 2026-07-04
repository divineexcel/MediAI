package router

import (
	"github.com/medisave/app/internal/domain/entity"
	"github.com/medisave/app/internal/presentation/http/middleware"
)

// RegisterAll mounts every module's routes onto the router.
func (r *Router) RegisterAll(h *Handlers) {
	// ─── WEB PAGES (HTML) ────────────────────────────────────────────────────
	r.engine.GET("/", h.Page.Root)
	r.engine.GET("/login", h.Page.Login)
	r.engine.GET("/register", h.Page.Register)

	// Serve uploaded documents
	r.engine.Static("/uploads", "./data/uploads")

	doctor := r.engine.Group("/doctor")
	{
		doctor.GET("/dashboard",    h.Page.DoctorDashboard)
		doctor.GET("/profile",      h.Page.DoctorProfile)
		doctor.GET("/appointments", h.Page.DoctorAppointments)
		doctor.GET("/patients",     h.Page.DoctorPatients)
		doctor.GET("/earnings",     h.Page.DoctorEarnings)
	}

	admin := r.engine.Group("/admin")
	{
		admin.GET("/dashboard", h.Page.AdminDashboard)
	}

	patient := r.engine.Group("/patient")
	{
		patient.GET("/dashboard",    h.Page.PatientDashboard)
		patient.GET("/profile",      h.Page.PatientProfile)
		patient.GET("/wallet",       h.Page.PatientWallet)
		patient.GET("/appointments", h.Page.PatientAppointments)
		patient.GET("/ai",           h.Page.PatientAI)
		patient.GET("/records",      h.Page.PatientRecords)
		patient.GET("/nearby",       h.Page.PatientNearby)
		patient.GET("/emergency",    h.Page.PatientEmergency)
		patient.GET("/reminders",    h.Page.PatientReminders)
		patient.GET("/savings",      h.Page.PatientSavings)
	}

	// Video call page (shared by patient and doctor)
	r.engine.GET("/consultation/:appointment_id/call", h.Page.ConsultationCall)

	v1 := r.V1()
	authMW := r.Authenticated()

	// ─── AUTH (/api/v1/auth) ──────────────────────────────────────────────────
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register",           h.Auth.Register)
		authGroup.POST("/register/doctor",    h.Auth.RegisterDoctor)
		authGroup.POST("/upload-document",    h.Auth.UploadDocument)
		authGroup.POST("/login",              h.Auth.Login)
		authGroup.POST("/refresh",            h.Auth.RefreshToken)

		protected := authGroup.Group("", authMW)
		{
			protected.POST("/logout",          h.Auth.Logout)
			protected.GET("/me",               h.Auth.Me)
			protected.POST("/forgot-password", h.Auth.ForgotPassword)
			protected.POST("/change-password", h.Auth.ChangePassword)
			protected.PATCH("/fcm-token",      h.Auth.UpdateFCMToken)
		}
	}

	// ─── PATIENTS (/api/v1/patients) ─────────────────────────────────────────
	patientGroup := v1.Group("/patients", authMW, middleware.RequirePatient())
	{
		patientGroup.GET("/dashboard",                h.Patient.Dashboard)
		patientGroup.GET("/profile",                  h.Patient.GetProfile)
		patientGroup.PUT("/profile",                  h.Patient.UpdateProfile)
		patientGroup.GET("/health-score",             h.Patient.GetHealthScore)
		patientGroup.GET("/notifications",            h.Patient.GetNotifications)
		patientGroup.PATCH("/notifications/:id/read", h.Patient.MarkNotificationRead)
		patientGroup.PATCH("/notifications/read-all", h.Patient.MarkAllNotificationsRead)
	}

	// ─── DOCTORS (/api/v1/doctors) ────────────────────────────────────────────
	doctorGroup := v1.Group("/doctors")
	{
		doctorGroup.GET("",              h.Doctor.List)
		doctorGroup.GET("/:id",          h.Doctor.GetByID)
		doctorGroup.GET("/:id/reviews",  h.Doctor.GetReviews)

		me := doctorGroup.Group("/me", authMW, middleware.RequireDoctor())
		{
			me.GET("/dashboard",       h.Doctor.Dashboard)
			me.GET("/profile",         h.Doctor.GetMyProfile)
			me.PUT("/profile",         h.Doctor.UpdateProfile)
			me.PATCH("/availability",  h.Doctor.ToggleAvailability)
			me.GET("/today",           h.Doctor.TodayAppointments)
			me.GET("/analytics",       h.Doctor.Analytics)
		}
	}

	// ─── WALLET (/api/v1/wallet) ──────────────────────────────────────────────
	v1.POST("/wallet/webhook", h.Wallet.PaystackWebhook)

	walletGroup := v1.Group("/wallet", authMW)
	{
		walletGroup.GET("",                         h.Wallet.GetWallet)
		walletGroup.POST("/deposit/initialize",     h.Wallet.InitializeDeposit)
		walletGroup.POST("/deposit/verify",         h.Wallet.VerifyDeposit)
		walletGroup.POST("/withdraw",               h.Wallet.Withdraw)
		walletGroup.GET("/transactions",            h.Wallet.GetTransactions)
		walletGroup.GET("/transactions/:id",        h.Wallet.GetTransaction)
		walletGroup.POST("/savings",                h.Wallet.CreateSavingsGoal)
		walletGroup.GET("/savings",                 h.Wallet.GetSavingsGoals)
		walletGroup.POST("/savings/:id/contribute", h.Wallet.ContributeToGoal)
	}

	// ─── APPOINTMENTS (/api/v1/appointments) ─────────────────────────────────
	apptGroup := v1.Group("/appointments", authMW)
	{
		apptGroup.GET("",     h.Appointment.List)
		apptGroup.GET("/:id", h.Appointment.GetByID)

		apptGroup.PATCH("/:id/start",    h.Appointment.Start)
		apptGroup.PATCH("/:id/complete", h.Appointment.Complete)
		apptGroup.PATCH("/:id/cancel",   h.Appointment.Cancel)

		patientAppt := apptGroup.Group("", middleware.RequirePatient())
		{
			patientAppt.POST("",             h.Appointment.Book)
			patientAppt.POST("/:id/review",  h.Appointment.LeaveReview)
		}

		// Room token: accessible by both doctor and patient (authenticated)
		apptGroup.GET("/:id/room-token", h.Room.GetToken)
	}

	// ─── CONSULTATIONS (/api/v1/consultations) ───────────────────────────────
	consGroup := v1.Group("/consultations", authMW, middleware.RequirePatientOrDoctor())
	{
		consGroup.GET("/:appointment_id",               h.Consultation.Get)
		consGroup.GET("/:appointment_id/messages",      h.Consultation.GetMessages)
		consGroup.POST("/:appointment_id/messages",     h.Consultation.SendMessage)
		consGroup.GET("/:appointment_id/prescriptions", h.Consultation.GetPrescriptions)

		doctorCons := consGroup.Group("", middleware.RequireDoctor())
		{
			doctorCons.PUT("/:appointment_id/notes",          h.Consultation.SaveNotes)
			doctorCons.POST("/:appointment_id/prescriptions", h.Consultation.AddPrescription)
		}
	}

	// ─── VIDEO CALL SIGNALING (WebSocket, no JWT middleware — token in query) ─
	v1.GET("/consultations/:appointment_id/call/signal", h.Call.Signal)
	v1.GET("/ws", h.Call.ConnectUserWS)

	// ─── MEDICAL RECORDS (/api/v1/records) ───────────────────────────────────
	recordGroup := v1.Group("/records", authMW, middleware.RequirePatient())
	{
		recordGroup.GET("",                          h.Record.List)
		recordGroup.POST("",                         h.Record.Create)
		recordGroup.GET("/:id",                      h.Record.GetByID)
		recordGroup.DELETE("/:id",                   h.Record.Delete)
		recordGroup.GET("/prescriptions",            h.Record.ListPrescriptions)
		recordGroup.PATCH("/prescriptions/:id/fill", h.Record.MarkPrescriptionFilled)
	}

	// ─── AI ASSISTANT (/api/v1/ai) ───────────────────────────────────────────
	aiGroup := v1.Group("/ai", authMW, middleware.RequirePatient())
	{
		aiGroup.POST("/chat",                      h.AI.Chat)
		aiGroup.GET("/conversations",              h.AI.GetConversations)
		aiGroup.GET("/conversations/:id/messages", h.AI.GetMessages)
		aiGroup.DELETE("/conversations/:id",       h.AI.ClearConversation)
	}

	// ─── EMERGENCY (/api/v1/emergency) ───────────────────────────────────────
	emergencyGroup := v1.Group("/emergency", authMW, middleware.RequirePatient())
	{
		emergencyGroup.POST("/sos",                   h.Emergency.SOS)
		emergencyGroup.PATCH("/:id/resolve",          h.Emergency.Resolve)
		emergencyGroup.GET("/history",                h.Emergency.GetHistory)
		emergencyGroup.GET("/contacts",               h.Emergency.GetContacts)
		emergencyGroup.POST("/contacts",              h.Emergency.AddContact)
		emergencyGroup.PUT("/contacts/:id",           h.Emergency.UpdateContact)
		emergencyGroup.DELETE("/contacts/:id",        h.Emergency.DeleteContact)
		emergencyGroup.PATCH("/contacts/:id/primary", h.Emergency.SetPrimaryContact)
	}

	// ─── REMINDERS (/api/v1/reminders) ───────────────────────────────────────
	reminderGroup := v1.Group("/reminders", authMW, middleware.RequirePatient())
	{
		reminderGroup.GET("",                  h.Reminder.List)
		reminderGroup.POST("",                 h.Reminder.Create)
		reminderGroup.GET("/:id",              h.Reminder.GetByID)
		reminderGroup.PUT("/:id",              h.Reminder.Update)
		reminderGroup.DELETE("/:id",           h.Reminder.Deactivate)
		reminderGroup.POST("/logs/:id/action", h.Reminder.LogAction)
		reminderGroup.GET("/analytics",        h.Reminder.Analytics)
	}

	// ─── MAPS (/api/v1/maps) ─────────────────────────────────────────────────
	mapsGroup := v1.Group("/maps", authMW)
	{
		mapsGroup.GET("/nearby",     h.Maps.Nearby)
		mapsGroup.GET("/directions", h.Maps.Directions)
	}

	// ─── SMS (/api/v1/sms) ───────────────────────────────────────────────────
	v1.POST("/sms/webhook", h.SMS.Webhook)
	smsGroup := v1.Group("/sms", authMW, middleware.RequireRole(entity.RoleAdmin))
	{
		smsGroup.POST("/send", h.SMS.Send)
	}

	// ─── USSD (/api/v1/ussd) ─────────────────────────────────────────────────
	ussdGroup := v1.Group("/ussd")
	{
		ussdGroup.POST("/session", h.USSD.Session)
		ussdGroup.GET("/test",     h.USSD.Test)
	}

	// ─── ADMIN (/api/v1/admin) ───────────────────────────────────────────────
	adminGroup := v1.Group("/admin", authMW, middleware.RequireAdmin())
	{
		adminGroup.GET("/dashboard",            h.Admin.Dashboard)
		adminGroup.GET("/analytics",            h.Admin.Analytics)
		adminGroup.GET("/patients",             h.Admin.ListPatients)
		adminGroup.GET("/patients/:id",         h.Admin.GetPatient)
		adminGroup.GET("/doctors",              h.Admin.ListDoctors)
		adminGroup.GET("/doctors/:id",          h.Admin.GetDoctor)
		adminGroup.PATCH("/doctors/:id/verify", h.Admin.VerifyDoctor)
		adminGroup.GET("/transactions",         h.Admin.ListTransactions)
		adminGroup.GET("/appointments",         h.Admin.ListAppointments)
		adminGroup.GET("/emergencies",          h.Admin.ListEmergencies)
		adminGroup.POST("/campaigns",           h.Admin.SendCampaign)
	}
}
