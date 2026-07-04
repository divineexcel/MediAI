package migrations

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/medisave/app/pkg/logger"
)

// Migration represents a single versioned schema change.
type Migration struct {
	Version     string
	Description string
	Up          func(db *gorm.DB) error
	Down        func(db *gorm.DB) error
}

// schemaMigration is the tracking table stored in the DB.
type schemaMigration struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	Version     string    `gorm:"uniqueIndex;not null"`
	Description string    `gorm:"not null"`
	AppliedAt   time.Time `gorm:"not null"`
}

// all is the ordered registry of every migration.
// New migrations are appended — never reordered or removed.
var all = []Migration{
	migration001InitialSchema(),
	migration002Indexes(),
	migration003DoctorDocuments(),
	migration004HealthCampaigns(),
	migration005USSDSessions(),
	migration006ConsultationRooms(),
	migration007DoctorRejectionStatus(),
	migration008CallDuration(),
}

// Run applies every pending migration in order, exactly once.
func Run(db *gorm.DB) error {
	// Bootstrap the tracking table itself
	if err := db.AutoMigrate(&schemaMigration{}); err != nil {
		return fmt.Errorf("bootstrap migration table: %w", err)
	}

	for _, m := range all {
		var existing schemaMigration
		result := db.Where("version = ?", m.Version).First(&existing)

		if result.Error == nil {
			continue // already applied
		}

		logger.Info("applying migration",
			zap.String("version", m.Version),
			zap.String("description", m.Description),
		)

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := m.Up(tx); err != nil {
				return err
			}
			return tx.Create(&schemaMigration{
				Version:     m.Version,
				Description: m.Description,
				AppliedAt:   time.Now(),
			}).Error
		}); err != nil {
			return fmt.Errorf("migration %s failed: %w", m.Version, err)
		}

		logger.Info("migration applied", zap.String("version", m.Version))
	}

	return nil
}
