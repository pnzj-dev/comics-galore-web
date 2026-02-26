-- =============================================================================
-- MESSAGING QUERIES (v3 Compatible)
-- =============================================================================

-- name: CreateConversation :one
INSERT INTO conversations (name, is_group)
VALUES ($1, $2)
RETURNING id;

-- name: AddParticipant :exec
INSERT INTO conversation_participants (conversation_id, user_id)
VALUES ($1, $2)
ON CONFLICT (conversation_id, user_id) DO NOTHING;

-- name: GetDirectConversation :one
-- Finds an existing 1-on-1 conversation between exactly two users.
-- Better-Auth user IDs are TEXT.
SELECT c.id AS conversation_id
FROM conversations c
WHERE c.is_group = FALSE
  AND EXISTS (SELECT 1 FROM conversation_participants cp WHERE cp.conversation_id = c.id AND cp.user_id = $1)
  AND EXISTS (SELECT 1 FROM conversation_participants cp WHERE cp.conversation_id = c.id AND cp.user_id = $2)
  AND (SELECT COUNT(*) FROM conversation_participants WHERE conversation_id = c.id) = 2
LIMIT 1;

-- name: InsertMessage :one
-- Using UUID v7 for the ID (handled by DEFAULT in schema)
INSERT INTO messages (conversation_id, sender_id, content)
VALUES ($1, $2, $3)
RETURNING id, conversation_id, sender_id, content, sent_at;

-- name: UpdateLastRead :exec
UPDATE conversation_participants
SET last_read_message_id = $1
WHERE conversation_id = $2
  AND user_id = $3;

-- name: GetMessagesBefore :many
-- Cursor-based pagination using UUID v7 (Time-ordered)
SELECT m.id,
       m.conversation_id,
       m.sender_id,
       u.name  AS sender_name,
       u.image AS sender_image,
       m.content,
       m.sent_at
FROM messages m
         JOIN users u ON u.id = m.sender_id
WHERE m.conversation_id = $1
  -- If cursor is null, get latest. If provided, get older (ID < cursor)
  AND (sqlc.narg('cursor')::uuid IS NULL OR m.id < sqlc.narg('cursor'))
ORDER BY m.id DESC
LIMIT $2;

-- name: GetOrCreateDirectConversation :one
SELECT get_or_create_direct_conversation($1, $2) AS conversation_id;

-- name: DeleteConversation :exec
DELETE
FROM conversations
WHERE id = $1;

-- name: ListUserConversations :many
-- Fetches chat list for a user with unread counts and the last message content.
SELECT c.id                                                                                       AS conversation_id,
       c.name                                                                                     AS conversation_name,
       c.is_group,
       c.updated_at,
       m.content                                                                                  AS last_message_content,
       m.sender_id                                                                                AS last_message_sender,
       (SELECT COUNT(*)
        FROM messages msg
        WHERE msg.conversation_id = c.id
          AND msg.id > COALESCE(cp.last_read_message_id, '00000000-0000-0000-0000-000000000000')) AS unread_count
FROM conversation_participants cp
         JOIN conversations c ON cp.conversation_id = c.id
         LEFT JOIN messages m ON c.last_message_id = m.id
WHERE cp.user_id = $1
ORDER BY c.updated_at DESC;