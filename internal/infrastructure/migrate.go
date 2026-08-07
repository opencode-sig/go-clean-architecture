package infrastructure

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrate runs schema auto-migration on the idempotency (infrastructure-owned)
// table plus all business-model tables passed by the composition root.
// Existing tables from before the version field existed carry NULL versions,
// which would break optimistic locking; they are normalized to 0 before
// AutoMigrate re-stamps the column NOT NULL DEFAULT 0.
func Migrate(db *gorm.DB, businessModels ...any) error {
	if err := db.AutoMigrate(&idempotencyEntry{}); err != nil {
		return fmt.Errorf("migrate infrastructure tables: %w", err)
	}

	normalizeVersions(db)

	if len(businessModels) > 0 {
		if err := db.AutoMigrate(businessModels...); err != nil {
			return fmt.Errorf("migrate business tables: %w", err)
		}
	}

	return nil
}

// normalizeVersions sets NULL versions to 0 on tables that already exist with a
// nullable version column (rows created before the field was added). Fresh
// databases have no such column and are skipped.
func normalizeVersions(db *gorm.DB) {
	for _, table := range []string{"users", "articles", "comments"} {
		var count int64
		if err := db.Raw(
			"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = 'version' AND IS_NULLABLE = 'YES'",
			table,
		).Scan(&count).Error; err != nil {
			continue
		}
		if count == 0 {
			continue
		}
		_ = db.Exec(fmt.Sprintf("UPDATE `%s` SET version = 0 WHERE version IS NULL", table)).Error
	}
}
