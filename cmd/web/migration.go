package main

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func (app *AppRunner) Migrate() error {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		app.Cfg.PostgresConfig.Auth.User,
		app.Cfg.PostgresConfig.Auth.Password,
		app.Cfg.PostgresConfig.Host,
		app.Cfg.PostgresConfig.Port,
		app.Cfg.PostgresConfig.DBName,
		app.Cfg.PostgresConfig.SslMode,
	)
	app.Logger.Infof("Running migrations with database: %s", app.Cfg.PostgresConfig.DBName)
	m, err := migrate.New("file://api/web-api/migrations", dsn)
	if err != nil {
		return fmt.Errorf("migration init failed: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}
	app.Logger.Infof("Migrations completed successfully")
	return nil
}
