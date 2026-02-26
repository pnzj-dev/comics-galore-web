package broadcaster

import (
	"comics-galore-web/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"sync"
)

type Service interface {
	Get(id string) *Broadcaster
}

type service struct {
	querier      *database.Queries
	broadcasters sync.Map
	logger       *slog.Logger
}

// NewService initializes the broadcaster service with structured logging.
func NewService(pool *pgxpool.Pool, logger *slog.Logger) Service {
	return &service{
		querier:      database.New(pool),
		broadcasters: sync.Map{},
		logger:       logger.With("service", "broadcaster"),
	}
}

func (b *service) Get(id string) *Broadcaster {
	l := b.logger.With("broadcaster_id", id, "op", "Get")

	// 1. Attempt to load existing broadcaster
	if val, ok := b.broadcasters.Load(id); ok {
		l.Debug("broadcaster cache hit")
		return val.(*Broadcaster)
	}

	// 2. Initialize new one if not found
	// Using LoadOrStore is safer for high-concurrency environments
	// to ensure two goroutines don't accidentally create two broadcasters for the same ID.
	newBroadcaster := New()
	val, loaded := b.broadcasters.LoadOrStore(id, newBroadcaster)

	if loaded {
		l.Debug("broadcaster created by concurrent request", "status", "reused_existing")
		return val.(*Broadcaster)
	}

	l.Info("new broadcaster initialized", "status", "created")
	return newBroadcaster
}
