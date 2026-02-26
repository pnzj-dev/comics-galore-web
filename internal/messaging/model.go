package messaging

import "github.com/google/uuid"

type Request struct {
	Content        string
	SenderID       string
	ConversationID uuid.UUID
}
