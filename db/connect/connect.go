package connect

import (
	"database/sql"
	"net/url"
	"path/filepath"

	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"go.uber.org/zap"
	"golang.org/x/net/context"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

func ConnectAndMigrate(ctx context.Context, sqliteDBPath, migrationsPath string) (*sql.DB, error) {
	dbConn, err := sql.Open("sqlite3", sqliteDBPath)
	if err != nil {
		lmt.Logger(ctx).Fatal("Error connecting to db", zap.Error(err))
	}

	driver, err := sqlite3.WithInstance(dbConn, &sqlite3.Config{})
	if err != nil {
		lmt.Logger(ctx).Fatal("Error creating migration driver", zap.Error(err))
	}

	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		lmt.Logger(ctx).Fatal("Error determining migrations filepath", zap.Error(err))
	}
	fileURL := url.URL{Scheme: "file", Path: absPath}
	m, err := migrate.NewWithDatabaseInstance(fileURL.String(), "sqlite3", driver)
	if err != nil {
		lmt.Logger(ctx).Fatal("Error creating migration", zap.Error(err))
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		lmt.Logger(ctx).Fatal("Error applying migration", zap.Error(err))
	}
	return dbConn, nil
}
