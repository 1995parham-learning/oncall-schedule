package db

import (
	"context"
	"fmt"

	"github.com/1995parham-learning/oncall-schedule/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides database dependencies.
var Module = fx.Module("db",
	fx.Provide(New),
)

// DB wraps the pgxpool.Pool with additional functionality.
type DB struct {
	Pool *pgxpool.Pool
	log  *zap.Logger
}

// New creates a new database connection pool and runs migrations.
func New(lc fx.Lifecycle, cfg *config.Config, logger *zap.Logger) (*DB, error) {
	log := logger.Named("db")

	// Build connection string
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
		cfg.Database.SSLMode,
	)

	// Configure connection pool
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	poolConfig.MaxConns = cfg.Database.MaxConnections
	poolConfig.MinConns = cfg.Database.MinConnections

	var pool *pgxpool.Pool

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Create connection pool
			p, err := pgxpool.NewWithConfig(ctx, poolConfig)
			if err != nil {
				return fmt.Errorf("unable to create connection pool: %w", err)
			}

			pool = p

			// Test connection
			if err := pool.Ping(ctx); err != nil {
				return fmt.Errorf("unable to ping database: %w", err)
			}

			log.Info("database connection established",
				zap.String("host", cfg.Database.Host),
				zap.Int("port", cfg.Database.Port),
				zap.String("database", cfg.Database.Database),
			)

			// Run migrations
			if err := runMigrations(connString, cfg.Database.MigrationsPath, log); err != nil {
				return fmt.Errorf("failed to run migrations: %w", err)
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			if pool != nil {
				log.Info("closing database connection")
				pool.Close()
			}
			return nil
		},
	})

	db := &DB{
		Pool: pool,
		log:  log,
	}

	return db, nil
}

// runMigrations runs database migrations using golang-migrate.
func runMigrations(connString, migrationsPath string, log *zap.Logger) error {
	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		connString,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Warn("failed to close migration source", zap.Error(srcErr))
		}
		if dbErr != nil {
			log.Warn("failed to close migration database", zap.Error(dbErr))
		}
	}()

	// Run migrations up
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if err == migrate.ErrNilVersion {
		log.Info("no migrations applied yet")
	} else {
		log.Info("migrations applied successfully",
			zap.Uint("version", version),
			zap.Bool("dirty", dirty),
		)
	}

	return nil
}

// Health checks the database connection health.
func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}
