package database

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v3/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"time"
)

// Resources holds the main application pool and a dedicated
// stateful connection for LISTEN/NOTIFY or other blocking operations.
type resources struct {
	Pool       *pgxpool.Pool
	ListenConn *pgx.Conn
}

func (r *resources) Close(ctx context.Context) {
	// 1. Handle the Listener Connection
	if r.ListenConn != nil {
		cleanupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		if ctx.Err() != nil {
			cleanupCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		}
		defer cancel()

		if err := r.ListenConn.Close(cleanupCtx); err != nil {
			log.Errorf("Database listener connection closed with error: %v", err)
		} else {
			log.Info("Database listener connection closed cleanly")
		}
	}

	// 2. Close the Connection Pool
	if r.Pool != nil {
		r.Pool.Close()
	}

	log.Info("All database resources released successfully")
}

func (r *resources) GetPool() *pgxpool.Pool {
	return r.Pool
}

func (r *resources) GetConn() *pgx.Conn {
	return r.ListenConn
}

type Resources interface {
	GetPool() *pgxpool.Pool
	GetConn() *pgx.Conn
	Close(ctx context.Context)
}

// NewResources initializes the application's database resources.
// It sets up a pool for standard queries and a single connection for stateful operations.
func NewResources(ctx context.Context, dsn string, logger *slog.Logger) (Resources, error) {
	if dsn == "" {
		// Log the error before returning
		logger.ErrorContext(ctx, "database initialization failed", "error", "DSN string is empty")
		return nil, fmt.Errorf("database DSN string is empty")
	}

	// Using slog.Info with context
	logger.InfoContext(ctx, "Initializing database infrastructure...")

	// 1. Configure the connection pool
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		logger.ErrorContext(ctx, "failed to parse pool config", "error", err)
		return nil, fmt.Errorf("parsing pool config: %w", err)
	}

	// Performance Tuning
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnIdleTime = 30 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	// Setup short-lived context for the connection phase
	initCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 2. Create the Pool
	pool, err := pgxpool.NewWithConfig(initCtx, poolConfig)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create connection pool", "error", err)
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// Verify pool health immediately
	if err := pool.Ping(initCtx); err != nil {
		logger.ErrorContext(ctx, "database ping failed", "error", err)
		pool.Close()
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	// 3. Create dedicated Connection for LISTEN/NOTIFY
	listenConn, err := pgx.Connect(initCtx, dsn)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create listener connection", "error", err)
		pool.Close()
		return nil, fmt.Errorf("creating listener connection: %w", err)
	}

	// Structured success log with metadata
	logger.InfoContext(ctx, "Database pool and listener connection established",
		"max_conns", poolConfig.MaxConns,
		"min_conns", poolConfig.MinConns,
	)

	return &resources{
		Pool:       pool,
		ListenConn: listenConn,
	}, nil
}
