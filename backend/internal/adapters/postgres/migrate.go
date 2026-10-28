package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Migrate applies every pending migration in fsys. It is run as its own step
// before a deploy, never implicitly on server start.
func Migrate(ctx context.Context, databaseURL string, fsys fs.FS, log *slog.Logger) error {
	connConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("parsing database url: %w", err)
	}
	db := stdlib.OpenDB(*connConfig)
	defer func() { _ = db.Close() }()
	return migrateWith(ctx, db, fsys, log)
}

func migrateWith(ctx context.Context, db *sql.DB, fsys fs.FS, log *slog.Logger) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, db, fsys)
	if err != nil {
		return fmt.Errorf("creating migration provider: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("applying migrations: %w", err)
	}
	for _, r := range results {
		log.Info("migration applied", "version", r.Source.Version, "file", r.Source.Path, "took", r.Duration)
	}
	if len(results) == 0 {
		log.Info("database is up to date")
	}
	return nil
}
