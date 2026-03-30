package view

import (
	"github.com/a-h/templ"
	"time"
)

type ModalConfig struct {
	IsStatic  bool
	Trigger   templ.Component
	Width     string // e.g., "sm:w-full", "sm:w-1/2"
	MinHeight string // e.g., "min-h-[50vh]", "min-h-0"
	MaxWidth  string // e.g., "sm:max-w-xl", "sm:max-w-5xl"
	MaxHeight string // e.g., "max-h-[90vh]", "max-h-[98vh]"
}

func NewModalConfig(options ...func(*ModalConfig)) *ModalConfig {
	cfg := &ModalConfig{
		Width:     "",
		MaxWidth:  "",
		MinHeight: "",
		MaxHeight: "",
		IsStatic:  false,
		Trigger:   nil,
	}
	for _, opt := range options {
		opt(cfg)
	}
	return cfg
}

type SupportMessage struct {
	ID         string    `json:"id"`
	Content    string    `json:"content"`
	SentAt     time.Time `json:"sentAt"`
	IsFromUser bool      `json:"isFromUser"`
}
