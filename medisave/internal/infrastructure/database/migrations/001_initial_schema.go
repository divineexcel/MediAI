package migrations

import (
	"gorm.io/gorm"
)

// migration001InitialSchema creates every table in dependency order.
// Foreign keys are enforced via SQLite PRAGMA foreign_keys=ON (set at connect time).
func migration001InitialSchema() Migration {
	return Migration{
		Version:     "001",
		Description: "initial schema — all core tables",
		Up:          up001,
		Down:        down001,
	}
}

func up001(db *gorm.DB) error {
	statements := []string{
		// ─── USERS ──────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS users (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			uuid             TEXT    UNIQUE NOT NULL,
			first_name       TEXT    NOT NULL,
			last_name        TEXT    NOT NULL,
			email            TEXT    UNIQUE NOT NULL,
			phone            TEXT    UNIQUE NOT NULL,
			password_hash    TEXT    NOT NULL,
			role             TEXT    NOT NULL CHECK(role IN ('patient','doctor','admin')),
			is_verified      INTEGER NOT NULL DEFAULT 0,
			is_active        INTEGER NOT NULL DEFAULT 1,
			profile_photo_url TEXT,
			fcm_token        TEXT,
			created_at       DATETIME NOT NULL,
			updated_at       DATETIME NOT NULL
		)`,

		// ─── PATIENTS ───────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS patients (
			id                  INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id             INTEGER UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			date_of_birth       DATETIME,
			gender              TEXT,
			blood_group         TEXT,
			genotype            TEXT,
			allergies           TEXT,
			chronic_conditions  TEXT,
			address             TEXT,
			state               TEXT,
			lga                 TEXT,
			nhis_number         TEXT,
			health_score        INTEGER NOT NULL DEFAULT 0,
			created_at          DATETIME NOT NULL,
			updated_at          DATETIME NOT NULL
		)`,

		// ─── DOCTORS ────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS doctors (
			id                   INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id              INTEGER UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			license_number       TEXT    UNIQUE NOT NULL,
			specialty            TEXT    NOT NULL,
			sub_specialty        TEXT,
			years_of_experience  INTEGER NOT NULL DEFAULT 0,
			hospital             TEXT,
			consultation_fee     REAL    NOT NULL DEFAULT 0,
			is_available         INTEGER NOT NULL DEFAULT 0,
			status               TEXT    NOT NULL DEFAULT 'pending'
			                     CHECK(status IN ('pending','verified','suspended')),
			bio                  TEXT,
			education            TEXT,
			certifications       TEXT,
			languages            TEXT,
			rating               REAL    NOT NULL DEFAULT 0,
			total_reviews        INTEGER NOT NULL DEFAULT 0,
			total_consultations  INTEGER NOT NULL DEFAULT 0,
			created_at           DATETIME NOT NULL,
			updated_at           DATETIME NOT NULL
		)`,

		// ─── WALLETS ────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS wallets (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id     INTEGER UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			owner_type  TEXT    NOT NULL CHECK(owner_type IN ('patient','doctor')),
			balance     REAL    NOT NULL DEFAULT 0 CHECK(balance >= 0),
			escrow      REAL    NOT NULL DEFAULT 0 CHECK(escrow >= 0),
			currency    TEXT    NOT NULL DEFAULT 'NGN',
			is_active   INTEGER NOT NULL DEFAULT 1,
			created_at  DATETIME NOT NULL,
			updated_at  DATETIME NOT NULL
		)`,

		// ─── TRANSACTIONS ───────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS transactions (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			reference         TEXT    UNIQUE NOT NULL,
			wallet_id         INTEGER NOT NULL REFERENCES wallets(id),
			type              TEXT    NOT NULL
			                  CHECK(type IN ('deposit','withdrawal','payment','refund',
			                                 'consultation_credit','savings')),
			amount            REAL    NOT NULL CHECK(amount > 0),
			balance_before    REAL    NOT NULL,
			balance_after     REAL    NOT NULL,
			status            TEXT    NOT NULL DEFAULT 'pending'
			                  CHECK(status IN ('pending','success','failed','reversed')),
			description       TEXT,
			metadata          TEXT,
			paystack_ref      TEXT,
			related_entity_id INTEGER,
			created_at        DATETIME NOT NULL,
			updated_at        DATETIME NOT NULL
		)`,

		// ─── APPOINTMENTS ───────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS appointments (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			patient_id       INTEGER NOT NULL REFERENCES patients(id),
			doctor_id        INTEGER NOT NULL REFERENCES doctors(id),
			type             TEXT    NOT NULL DEFAULT 'chat'
			                 CHECK(type IN ('chat','voice','video')),
			status           TEXT    NOT NULL DEFAULT 'pending'
			                 CHECK(status IN ('pending','confirmed','in_progress',
			                                  'completed','cancelled','no_show')),
			scheduled_at     DATETIME NOT NULL,
			started_at       DATETIME,
			completed_at     DATETIME,
			consultation_fee REAL    NOT NULL,
			transaction_id   INTEGER,
			chief_complaint  TEXT,
			notes            TEXT,
			cancel_reason    TEXT,
			created_at       DATETIME NOT NULL,
			updated_at       DATETIME NOT NULL
		)`,

		// ─── CONSULTATIONS ──────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS consultations (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			appointment_id  INTEGER UNIQUE NOT NULL REFERENCES appointments(id),
			doctor_notes    TEXT,
			diagnosis       TEXT,
			treatment       TEXT,
			follow_up_date  DATETIME,
			follow_up_notes TEXT,
			created_at      DATETIME NOT NULL,
			updated_at      DATETIME NOT NULL
		)`,

		// ─── CONSULTATION MESSAGES ───────────────────────────────────
		`CREATE TABLE IF NOT EXISTS consultation_messages (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			appointment_id INTEGER NOT NULL REFERENCES appointments(id) ON DELETE CASCADE,
			sender_id      INTEGER NOT NULL REFERENCES users(id),
			sender_role    TEXT    NOT NULL CHECK(sender_role IN ('patient','doctor')),
			message        TEXT    NOT NULL,
			message_type   TEXT    NOT NULL DEFAULT 'text'
			               CHECK(message_type IN ('text','image','file','prescription')),
			attachment_url TEXT,
			is_read        INTEGER NOT NULL DEFAULT 0,
			created_at     DATETIME NOT NULL
		)`,

		// ─── MEDICAL RECORDS ────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS medical_records (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			patient_id      INTEGER NOT NULL REFERENCES patients(id),
			consultation_id INTEGER REFERENCES consultations(id),
			doctor_id       INTEGER REFERENCES doctors(id),
			record_type     TEXT    NOT NULL
			                CHECK(record_type IN ('consultation_note','prescription',
			                                      'lab_report','imaging','vaccination',
			                                      'discharge_summary','other')),
			title           TEXT    NOT NULL,
			description     TEXT,
			file_url        TEXT,
			record_date     DATETIME NOT NULL,
			is_shared       INTEGER NOT NULL DEFAULT 0,
			created_at      DATETIME NOT NULL,
			updated_at      DATETIME NOT NULL
		)`,

		// ─── PRESCRIPTIONS ──────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS prescriptions (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			consultation_id INTEGER NOT NULL REFERENCES consultations(id),
			patient_id      INTEGER NOT NULL REFERENCES patients(id),
			doctor_id       INTEGER NOT NULL REFERENCES doctors(id),
			medicine_name   TEXT    NOT NULL,
			dosage          TEXT    NOT NULL,
			frequency       TEXT    NOT NULL,
			duration        TEXT    NOT NULL,
			instructions    TEXT,
			is_filled       INTEGER NOT NULL DEFAULT 0,
			created_at      DATETIME NOT NULL
		)`,

		// ─── AI CONVERSATIONS ───────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS ai_conversations (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			patient_id INTEGER NOT NULL REFERENCES patients(id),
			session_id TEXT    NOT NULL,
			is_active  INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,

		// ─── AI MESSAGES ────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS ai_messages (
			id                 INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id    INTEGER NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
			role               TEXT    NOT NULL CHECK(role IN ('user','assistant')),
			content            TEXT    NOT NULL,
			severity           TEXT    CHECK(severity IN ('low','moderate','high')),
			possible_condition TEXT,
			recommended_action TEXT,
			lab_tests          TEXT,
			home_care          TEXT,
			emergency_warning  TEXT,
			show_book_doctor   INTEGER NOT NULL DEFAULT 0,
			show_find_hospital INTEGER NOT NULL DEFAULT 0,
			show_emergency_sos INTEGER NOT NULL DEFAULT 0,
			created_at         DATETIME NOT NULL
		)`,

		// ─── EMERGENCIES ────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS emergencies (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			patient_id        INTEGER NOT NULL REFERENCES patients(id),
			status            TEXT    NOT NULL DEFAULT 'active'
			                  CHECK(status IN ('active','resolved','false_alarm')),
			latitude          REAL,
			longitude         REAL,
			address           TEXT,
			nearest_hospital  TEXT,
			description       TEXT,
			contacts_notified INTEGER NOT NULL DEFAULT 0,
			sms_sent          INTEGER NOT NULL DEFAULT 0,
			push_sent         INTEGER NOT NULL DEFAULT 0,
			resolved_at       DATETIME,
			created_at        DATETIME NOT NULL,
			updated_at        DATETIME NOT NULL
		)`,

		// ─── EMERGENCY CONTACTS ─────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS emergency_contacts (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			patient_id   INTEGER NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
			name         TEXT    NOT NULL,
			phone        TEXT    NOT NULL,
			relationship TEXT,
			is_primary   INTEGER NOT NULL DEFAULT 0,
			created_at   DATETIME NOT NULL,
			updated_at   DATETIME NOT NULL
		)`,

		// ─── MEDICATION REMINDERS ────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS medication_reminders (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			patient_id      INTEGER NOT NULL REFERENCES patients(id),
			prescription_id INTEGER REFERENCES prescriptions(id),
			medicine_name   TEXT    NOT NULL,
			dosage          TEXT    NOT NULL,
			frequency       TEXT    NOT NULL
			                CHECK(frequency IN ('once','daily','twice_daily',
			                                    'three_times_daily','weekly')),
			morning_time    TEXT,
			afternoon_time  TEXT,
			night_time      TEXT,
			start_date      DATETIME NOT NULL,
			end_date        DATETIME NOT NULL,
			instructions    TEXT,
			is_active       INTEGER NOT NULL DEFAULT 1,
			created_at      DATETIME NOT NULL,
			updated_at      DATETIME NOT NULL
		)`,

		// ─── REMINDER LOGS ──────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS reminder_logs (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			reminder_id  INTEGER  NOT NULL REFERENCES medication_reminders(id) ON DELETE CASCADE,
			patient_id   INTEGER  NOT NULL REFERENCES patients(id),
			scheduled_at DATETIME NOT NULL,
			action_at    DATETIME,
			status       TEXT     NOT NULL DEFAULT 'missed'
			             CHECK(status IN ('taken','skipped','missed')),
			notes        TEXT,
			created_at   DATETIME NOT NULL
		)`,

		// ─── HEALTH SAVINGS GOALS ───────────────────────────────────
		`CREATE TABLE IF NOT EXISTS health_savings_goals (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			patient_id       INTEGER NOT NULL REFERENCES patients(id),
			wallet_id        INTEGER NOT NULL REFERENCES wallets(id),
			title            TEXT    NOT NULL,
			description      TEXT,
			target_amount    REAL    NOT NULL CHECK(target_amount > 0),
			saved_amount     REAL    NOT NULL DEFAULT 0 CHECK(saved_amount >= 0),
			frequency        TEXT    NOT NULL,
			auto_save_amount REAL,
			status           TEXT    NOT NULL DEFAULT 'active'
			                 CHECK(status IN ('active','completed','cancelled')),
			target_date      DATETIME NOT NULL,
			completed_at     DATETIME,
			created_at       DATETIME NOT NULL,
			updated_at       DATETIME NOT NULL
		)`,

		// ─── NOTIFICATIONS ──────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS notifications (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			type       TEXT    NOT NULL
			           CHECK(type IN ('appointment','medication','wallet','emergency',
			                          'ai','consultation','system')),
			channel    TEXT    NOT NULL CHECK(channel IN ('push','sms','in_app')),
			title      TEXT    NOT NULL,
			body       TEXT    NOT NULL,
			data       TEXT,
			is_read    INTEGER NOT NULL DEFAULT 0,
			is_sent    INTEGER NOT NULL DEFAULT 0,
			sent_at    DATETIME,
			created_at DATETIME NOT NULL
		)`,

		// ─── REVIEWS ────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS reviews (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			patient_id     INTEGER NOT NULL REFERENCES patients(id),
			doctor_id      INTEGER NOT NULL REFERENCES doctors(id),
			appointment_id INTEGER UNIQUE NOT NULL REFERENCES appointments(id),
			rating         INTEGER NOT NULL CHECK(rating BETWEEN 1 AND 5),
			comment        TEXT,
			is_visible     INTEGER NOT NULL DEFAULT 1,
			created_at     DATETIME NOT NULL,
			updated_at     DATETIME NOT NULL
		)`,
	}

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}

	return nil
}

func down001(db *gorm.DB) error {
	// Drop in reverse dependency order
	tables := []string{
		"reviews", "notifications", "health_savings_goals",
		"reminder_logs", "medication_reminders", "emergency_contacts",
		"emergencies", "ai_messages", "ai_conversations",
		"prescriptions", "medical_records", "consultation_messages",
		"consultations", "appointments", "transactions",
		"wallets", "doctors", "patients", "users",
	}
	for _, t := range tables {
		if err := db.Exec("DROP TABLE IF EXISTS " + t).Error; err != nil {
			return err
		}
	}
	return nil
}
