package server

import (
	"github.com/gofiber/fiber/v2"

	"comics-galore-web/internal/database"
)

type FiberServer struct {
	*fiber.App

	db database.Service
}

func New() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "comics-galore-web",
			AppName:      "comics-galore-web",
		}),

		db: database.New(),
	}

	return server
}
