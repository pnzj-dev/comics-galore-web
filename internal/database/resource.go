package database

import (
	"context"
	"fmt"
	"github.com/gofiber/fiber/v3/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
func NewResources(ctx context.Context, dsn string) (Resources, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database DSN string is empty")
	}

	log.Info("Initializing database infrastructure...")

	// 1. Configure the connection pool
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
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
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// Verify pool health immediately
	if err := pool.Ping(initCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	// 3. Create dedicated Connection for LISTEN/NOTIFY
	// We use the same DSN but a separate pgx.Connect to ensure it's outside the pool
	listenConn, err := pgx.Connect(initCtx, dsn)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("creating listener connection: %w", err)
	}

	log.Info("Database pool and listener connection established")

	return &resources{
		Pool:       pool,
		ListenConn: listenConn,
	}, nil
}
