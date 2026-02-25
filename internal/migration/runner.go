package migration

import (
	"fmt"
	"strings"

	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Run applies all pending up migrations from migrationsPath against databaseURL.
// It is idempotent: if no migrations are pending, it returns nil.
func Run(databaseURL, migrationsPath string) error {
	migrateURL := strings.Replace(databaseURL, "postgresql://", "pgx5://", 1)
	migrateURL = strings.Replace(migrateURL, "postgres://", "pgx5://", 1)

	m, err := migrate.New("file://"+migrationsPath, migrateURL)
	if err != nil {
		return fmt.Errorf("migration.New: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration.Up: %w", err)
	}
	return nil
}
