package migrations

import "gorm.io/gorm"

// migration002Indexes adds all query-optimized indexes.
// Separated from schema so that index creation failures are isolated.
// Every index is justified by a real query pattern in the application.
func migration002Indexes() Migration {
	return Migration{
		Version:     "002",
		Description: "performance indexes for all critical query paths",
		Up:          up002,
		Down:        down002,
	}
}

func up002(db *gorm.DB) error {
	indexes := []string{
		// ─── USERS ─────────────────────────────────────────────────
		// Login lookups
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone)`,
		// Admin list by role
		`CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)`,
		// Active user filtering
		`CREATE INDEX IF NOT EXISTS idx_users_active_role ON users(is_active, role)`,

		// ─── PATIENTS ──────────────────────────────────────────────
		// Profile load by user
		`CREATE INDEX IF NOT EXISTS idx_patients_user_id ON patients(user_id)`,
		// Admin patient search by state
		`CREATE INDEX IF NOT EXISTS idx_patients_state ON patients(state)`,

		// ─── DOCTORS ───────────────────────────────────────────────
		// Profile load by user
		`CREATE INDEX IF NOT EXISTS idx_doctors_user_id ON doctors(user_id)`,
		// Patient searching for available doctors by specialty
		`CREATE INDEX IF NOT EXISTS idx_doctors_specialty_available ON doctors(specialty, is_available, status)`,
		// Admin verification queue
		`CREATE INDEX IF NOT EXISTS idx_doctors_status ON doctors(status)`,
		// Top-rated doctor listings
		`CREATE INDEX IF NOT EXISTS idx_doctors_rating ON doctors(rating DESC)`,

		// ─── WALLETS ───────────────────────────────────────────────
		// Wallet lookup for every payment operation
		`CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id)`,

		// ─── TRANSACTIONS ──────────────────────────────────────────
		// Statement/history per wallet (most common query)
		`CREATE INDEX IF NOT EXISTS idx_transactions_wallet_id_created ON transactions(wallet_id, created_at DESC)`,
		// Paystack webhook reconciliation
		`CREATE INDEX IF NOT EXISTS idx_transactions_paystack_ref ON transactions(paystack_ref)`,
		// Reference-based deduplication
		`CREATE INDEX IF NOT EXISTS idx_transactions_reference ON transactions(reference)`,
		// Admin transaction filtering by status
		`CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status)`,
		// Admin transaction filtering by type
		`CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type)`,

		// ─── APPOINTMENTS ──────────────────────────────────────────
		// Patient's appointment history
		`CREATE INDEX IF NOT EXISTS idx_appointments_patient_id ON appointments(patient_id, scheduled_at DESC)`,
		// Doctor's appointment list (today's schedule)
		`CREATE INDEX IF NOT EXISTS idx_appointments_doctor_id ON appointments(doctor_id, scheduled_at DESC)`,
		// Conflict detection: is doctor free at time X?
		`CREATE INDEX IF NOT EXISTS idx_appointments_conflict ON appointments(doctor_id, scheduled_at, status)`,
		// Status filtering (admin, analytics)
		`CREATE INDEX IF NOT EXISTS idx_appointments_status ON appointments(status)`,

		// ─── CONSULTATIONS ─────────────────────────────────────────
		// Load consultation by appointment (1-to-1)
		`CREATE INDEX IF NOT EXISTS idx_consultations_appointment_id ON consultations(appointment_id)`,

		// ─── CONSULTATION MESSAGES ──────────────────────────────────
		// Chat window load (ordered by time)
		`CREATE INDEX IF NOT EXISTS idx_messages_appointment_time ON consultation_messages(appointment_id, created_at ASC)`,
		// Unread message count per appointment
		`CREATE INDEX IF NOT EXISTS idx_messages_unread ON consultation_messages(appointment_id, is_read)`,

		// ─── MEDICAL RECORDS ───────────────────────────────────────
		// Patient timeline (most recent first)
		`CREATE INDEX IF NOT EXISTS idx_records_patient_date ON medical_records(patient_id, record_date DESC)`,
		// Filter by record type
		`CREATE INDEX IF NOT EXISTS idx_records_type ON medical_records(patient_id, record_type)`,

		// ─── PRESCRIPTIONS ─────────────────────────────────────────
		// Patient's prescription list
		`CREATE INDEX IF NOT EXISTS idx_prescriptions_patient_id ON prescriptions(patient_id)`,
		// All prescriptions from a consultation
		`CREATE INDEX IF NOT EXISTS idx_prescriptions_consultation ON prescriptions(consultation_id)`,

		// ─── AI ─────────────────────────────────────────────────────
		// Load active conversation for patient
		`CREATE INDEX IF NOT EXISTS idx_ai_conv_patient_active ON ai_conversations(patient_id, is_active)`,
		// Load messages for a conversation
		`CREATE INDEX IF NOT EXISTS idx_ai_messages_conv ON ai_messages(conversation_id, created_at ASC)`,
		// Daily usage cap check
		`CREATE INDEX IF NOT EXISTS idx_ai_messages_patient_date ON ai_messages(conversation_id)`,

		// ─── EMERGENCIES ───────────────────────────────────────────
		// Patient's emergency history
		`CREATE INDEX IF NOT EXISTS idx_emergencies_patient ON emergencies(patient_id, created_at DESC)`,
		// Admin active emergency dashboard
		`CREATE INDEX IF NOT EXISTS idx_emergencies_status ON emergencies(status)`,

		// ─── EMERGENCY CONTACTS ────────────────────────────────────
		// Load contacts for a patient (used during SOS)
		`CREATE INDEX IF NOT EXISTS idx_emergency_contacts_patient ON emergency_contacts(patient_id, is_primary DESC)`,

		// ─── MEDICATION REMINDERS ──────────────────────────────────
		// Patient's active reminders
		`CREATE INDEX IF NOT EXISTS idx_reminders_patient_active ON medication_reminders(patient_id, is_active)`,
		// Scheduler: find all due reminders in a time window
		`CREATE INDEX IF NOT EXISTS idx_reminders_active_dates ON medication_reminders(is_active, start_date, end_date)`,

		// ─── REMINDER LOGS ─────────────────────────────────────────
		// Adherence analytics per patient
		`CREATE INDEX IF NOT EXISTS idx_reminder_logs_patient ON reminder_logs(patient_id, scheduled_at DESC)`,
		// Per-reminder adherence rate
		`CREATE INDEX IF NOT EXISTS idx_reminder_logs_reminder ON reminder_logs(reminder_id, status)`,

		// ─── HEALTH SAVINGS GOALS ──────────────────────────────────
		// Patient's goals list
		`CREATE INDEX IF NOT EXISTS idx_savings_patient ON health_savings_goals(patient_id, status)`,

		// ─── NOTIFICATIONS ─────────────────────────────────────────
		// Notification bell: unread count
		`CREATE INDEX IF NOT EXISTS idx_notif_user_unread ON notifications(user_id, is_read)`,
		// Notification list ordered by time
		`CREATE INDEX IF NOT EXISTS idx_notif_user_created ON notifications(user_id, created_at DESC)`,
		// Delivery retry: unsent notifications
		`CREATE INDEX IF NOT EXISTS idx_notif_unsent ON notifications(is_sent, channel)`,

		// ─── REVIEWS ────────────────────────────────────────────────
		// Doctor's reviews list
		`CREATE INDEX IF NOT EXISTS idx_reviews_doctor ON reviews(doctor_id, created_at DESC)`,
		// Rating calculation per doctor
		`CREATE INDEX IF NOT EXISTS idx_reviews_doctor_rating ON reviews(doctor_id, rating)`,
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	return nil
}

func down002(db *gorm.DB) error {
	indexes := []string{
		"idx_users_email", "idx_users_phone", "idx_users_role", "idx_users_active_role",
		"idx_patients_user_id", "idx_patients_state",
		"idx_doctors_user_id", "idx_doctors_specialty_available", "idx_doctors_status", "idx_doctors_rating",
		"idx_wallets_user_id",
		"idx_transactions_wallet_id_created", "idx_transactions_paystack_ref",
		"idx_transactions_reference", "idx_transactions_status", "idx_transactions_type",
		"idx_appointments_patient_id", "idx_appointments_doctor_id",
		"idx_appointments_conflict", "idx_appointments_status",
		"idx_consultations_appointment_id",
		"idx_messages_appointment_time", "idx_messages_unread",
		"idx_records_patient_date", "idx_records_type",
		"idx_prescriptions_patient_id", "idx_prescriptions_consultation",
		"idx_ai_conv_patient_active", "idx_ai_messages_conv", "idx_ai_messages_patient_date",
		"idx_emergencies_patient", "idx_emergencies_status",
		"idx_emergency_contacts_patient",
		"idx_reminders_patient_active", "idx_reminders_active_dates",
		"idx_reminder_logs_patient", "idx_reminder_logs_reminder",
		"idx_savings_patient",
		"idx_notif_user_unread", "idx_notif_user_created", "idx_notif_unsent",
		"idx_reviews_doctor", "idx_reviews_doctor_rating",
	}

	for _, idx := range indexes {
		db.Exec("DROP INDEX IF EXISTS " + idx)
	}
	return nil
}
