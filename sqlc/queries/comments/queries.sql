-- =============================================================================
-- COMMENT QUERIES (v3 Compatible)
-- =============================================================================

-- name: CreateComment :one
-- Note: Depth is calculated automatically by the DB trigger.
INSERT INTO comments (post_id,
                      parent_id,
                      user_id,
                      content)
VALUES ($1, $2, $3, $4)
RETURNING id, post_id, parent_id, depth, user_id, content, created_at;

-- name: GetCommentsByPostID :many
-- Uses a Recursive CTE to fetch the entire hierarchy for a post in one trip.
WITH RECURSIVE comment_tree AS (
    -- 1. Anchor: Root level comments (parent_id is NULL)
    SELECT c.id,
           c.post_id,
           c.parent_id,
           c.depth,
           c.user_id,
           c.content,
           c.created_at
    FROM comments c
    WHERE c.post_id = $1
      AND c.parent_id IS NULL

    UNION ALL

    -- 2. Recursive: Join children to their established parents
    SELECT child.id,
           child.post_id,
           child.parent_id,
           child.depth,
           child.user_id,
           child.content,
           child.created_at
    FROM comments child
             INNER JOIN comment_tree ct ON child.parent_id = ct.id)
SELECT ct.id,
       ct.post_id,
       ct.parent_id,
       ct.depth,
       ct.user_id,
       ct.content,
       ct.created_at
FROM comment_tree ct
ORDER BY ct.depth ASC, ct.created_at ASC;

-- name: GetCommentByID :one
SELECT *
FROM comments
WHERE id = $1;

-- name: GetCommentDepth :one
-- Useful if your Go code wants to enforce a max nesting limit (e.g., max 5 levels).
SELECT depth
FROM comments
WHERE id = $1;

-- name: UpdateComment :one
UPDATE comments
SET content = $2
WHERE id = $1
RETURNING id, post_id, parent_id, depth, user_id, content, created_at;

-- name: DeleteComment :exec
-- Cascade handles nested replies.
DELETE
FROM comments
WHERE id = $1;