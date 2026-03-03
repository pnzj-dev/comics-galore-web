package websocket

import (
	"comics-galore-web/internal/config"
	"log/slog"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

type handler struct {
	logger *slog.Logger
}

type Handler interface {
	RegisterRoutes(app *fiber.App)
	Listen(c *websocket.Conn)
}

func NewHandler(cfg config.Service) Handler {
	return &handler{
		logger: cfg.GetLogger().With("component", "websocket_handler"),
	}
}

func (w *handler) RegisterRoutes(app *fiber.App) {
	// 1. Grouping your websocket routes is cleaner
	wsGroup := app.Group("/ws")

	// 2. Apply the upgrade check only to this group
	wsGroup.Use(func(c fiber.Ctx) error {
		// In Fiber v3, this check is robust across the interface
		if websocket.IsWebSocketUpgrade(c) {
			w.logger.Debug("valid websocket upgrade request", "path", c.Path())
			return c.Next()
		}
		return fiber.ErrUpgradeRequired // Returns 426 Upgrade Required
	})

	// 3. Define the actual connection endpoint
	// Note: The path here is "" because it inherits "/ws" from the group
	wsGroup.Get("", websocket.New(w.Listen))
}

func (w *handler) Listen(conn *websocket.Conn) {
	// Extract connection info for structured logging
	l := w.logger.With(
		"remote_addr", conn.RemoteAddr().String(),
		"local_addr", conn.LocalAddr().String(),
	)

	l.Info("new connection established")

	defer func() {
		if err := conn.Close(); err != nil {
			l.Error("failed to close websocket connection", "error", err)
		}
		l.Info("connection closed")
	}()

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil {
			// Check if it's a normal closure or an actual error
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				l.Error("read error", "error", err)
			} else {
				l.Debug("client disconnected normally")
			}
			break
		}

		l.Debug("message received", "type", mt, "size", len(msg))

		// Echo the message back
		if err = conn.WriteMessage(mt, msg); err != nil {
			l.Error("write error", "error", err)
			break
		}
	}
	/*
		select {
		case <-ctx.Done():
			return
		default:
			payload := fmt.Sprintf("server timestamp: %d", time.Now().UnixNano())
			if err := con.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
				log.Printf("could not write to socket: %v", err)
				return
			}
			time.Sleep(time.Second * 2)
		}
	*/
}
