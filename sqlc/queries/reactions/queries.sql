-- =============================================================================
-- REACTION QUERIES
-- =============================================================================

-- name: UpsertReaction :exec
-- Insert or update a reaction (toggle/switch like ↔ unlike)
-- user_id is TEXT (Better-Auth), target_id is UUID (Post/Comment)
INSERT INTO reactions (user_id,
                       target_id,
                       target_type,
                       reaction_type)
VALUES ($1, $2, $3, $4::reaction_type)
ON CONFLICT (user_id, target_id, target_type)
    DO UPDATE SET reaction_type = EXCLUDED.reaction_type;

-- name: RemoveReaction :exec
-- Remove any reaction from a user on a target
DELETE
FROM reactions
WHERE user_id = $1
  AND target_id = $2
  AND target_type = $3;

-- name: GetUserReaction :one
-- Get a user's current reaction on a target (NULL if none)
SELECT reaction_type
FROM reactions
WHERE user_id = $1
  AND target_id = $2
  AND target_type = $3;

-- name: GetReactionSummary :one
-- Aggregate counts for a target. target_id is UUID.
SELECT COUNT(*) FILTER (WHERE reaction_type = 'like')::bigint     AS likes,
       COUNT(*) FILTER (WHERE reaction_type = 'unlike')::bigint   AS unlikes,
       (COUNT(*) FILTER (WHERE reaction_type = 'like') -
        COUNT(*) FILTER (WHERE reaction_type = 'unlike'))::bigint AS net_likes
FROM reactions
WHERE target_id = $1
  AND target_type = $2;
