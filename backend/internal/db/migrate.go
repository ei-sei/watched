package db

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Migrate(dsn string) error {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	err = m.Up()
	if err == nil || err == migrate.ErrNoChange {
		return nil
	}

	// If a previous migration was left dirty (e.g. the DDL statement failed),
	// force the version back to the last clean state and retry.
	var dirtyErr migrate.ErrDirty
	if errors.As(err, &dirtyErr) {
		if ferr := m.Force(dirtyErr.Version - 1); ferr != nil {
			return fmt.Errorf("force migration version: %w", ferr)
		}
		if err = m.Up(); err != nil && err != migrate.ErrNoChange {
			return fmt.Errorf("run migrations after dirty recovery: %w", err)
		}
		return nil
	}

	return fmt.Errorf("run migrations: %w", err)
}
