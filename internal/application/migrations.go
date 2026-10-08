package application

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/zxchelik/room-booking-service/migrations"
)

var (
	gooseSetupOnce sync.Once
	gooseSetupErr  error
)

func RunMigrations(ctx context.Context, databaseURL string) error {
	gooseSetupOnce.Do(func() {
		goose.SetBaseFS(migrations.Files)
		gooseSetupErr = goose.SetDialect("postgres")
	})
	if gooseSetupErr != nil {
		return fmt.Errorf("configure goose: %w", gooseSetupErr)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open migration db: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping migration db: %w", err)
	}

	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
