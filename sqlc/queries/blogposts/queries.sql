-- name: ListPosts :many
SELECT sqlc.embed(b),
       sqlc.embed(c),
       sqlc.embed(s)
FROM blogposts b
         JOIN categories c ON b.category_id = c.id
         LEFT JOIN blogpost_stats s ON b.id = s.post_id
ORDER BY b.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetPostByID :one
SELECT sqlc.embed(b),
       sqlc.embed(c),
       sqlc.embed(s)
FROM blogposts b
         JOIN categories c ON b.category_id = c.id
         LEFT JOIN blogpost_stats s ON b.id = s.post_id
WHERE b.id = $1;

-- name: UpsertBlogPost :one
INSERT INTO blogposts (id,
                       title,
                       author_name,
                       uploader_id,
                       category_id,
                       description,
                       tags,
                       cover,
                       previews,
                       rating,
                       language_code,
                       pages,
                       size_bytes,
                       mime_types,
                       updated_at)
VALUES (COALESCE(sqlc.narg('id')::uuid, uuid_generate_v7()),
        $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
ON CONFLICT (id) DO UPDATE SET title         = EXCLUDED.title,
                               category_id   = EXCLUDED.category_id,
                               description   = EXCLUDED.description,
                               tags          = EXCLUDED.tags,
                               cover         = EXCLUDED.cover,
                               previews      = EXCLUDED.previews,
                               rating        = EXCLUDED.rating,
                               language_code = EXCLUDED.language_code,
                               pages         = EXCLUDED.pages,
                               size_bytes    = EXCLUDED.size_bytes,
                               mime_types    = EXCLUDED.mime_types,
                               updated_at    = NOW()
RETURNING *;

-- name: GetPostArchives :many
SELECT sqlc.embed(a), sqlc.embed(s)
FROM archives a
         JOIN archive_stats s ON a.id = s.archive_id
WHERE a.post_id = $1;

-- name: IncrementDownloadCount :exec
-- Updates the separate stats table to avoid locking the main archive table.
INSERT INTO archive_stats (archive_id, downloads)
VALUES ($1, 1)
ON CONFLICT (archive_id)
    DO UPDATE SET downloads = archive_stats.downloads + 1;

-- name: IncrementPostView :exec
INSERT INTO blogpost_stats (post_id, auth_views, anon_views)
VALUES ($1, $2, $3)
ON CONFLICT (post_id) DO UPDATE SET auth_views = blogpost_stats.auth_views + EXCLUDED.auth_views,
                                    anon_views = blogpost_stats.anon_views + EXCLUDED.anon_views;

-- name: DeletePost :exec
DELETE
FROM blogposts
WHERE id = $1;

-- name: ListPostsByTags :many
SELECT sqlc.embed(b), sqlc.embed(c), sqlc.embed(s)
FROM blogposts b
         JOIN categories c ON b.category_id = c.id
         LEFT JOIN blogpost_stats s ON b.id = s.post_id
WHERE b.tags @> $1::TEXT[]
ORDER BY b.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListPostsByTagsFlexible :many
SELECT sqlc.embed(b), sqlc.embed(c), sqlc.embed(s)
FROM blogposts b
         JOIN categories c ON b.category_id = c.id
         LEFT JOIN blogpost_stats s ON b.id = s.post_id
WHERE (
          -- If match_all is true, use "Contains" (@>)
          (sqlc.arg('match_all')::boolean = true AND b.tags @> sqlc.arg('tags')::text[])
              OR
              -- If match_all is false, use "Overlaps" (&&)
          (sqlc.arg('match_all')::boolean = false AND b.tags && sqlc.arg('tags')::text[])
          )
ORDER BY b.created_at DESC
LIMIT sqlc.arg('limit_val') OFFSET sqlc.arg('offset_val');

-- name: SearchPosts :many
SELECT sqlc.embed(b), sqlc.embed(c), sqlc.embed(s)
FROM blogposts b
         JOIN categories c ON b.category_id = c.id
         LEFT JOIN blogpost_stats s ON b.id = s.post_id
WHERE
  -- 1. Optional Category Filter
    (sqlc.narg('category_id')::uuid IS NULL OR b.category_id = sqlc.narg('category_id'))
  AND
  -- 2. Flexible Tag Filter
    (
        (sqlc.arg('match_all')::boolean = true AND b.tags @> sqlc.arg('tags')::text[])
            OR
        (sqlc.arg('match_all')::boolean = false AND b.tags && sqlc.arg('tags')::text[])
            OR
        (cardinality(sqlc.arg('tags')::text[]) = 0) -- Ignore if no tags provided
        )
  AND
  -- 3. Minimum Rating Filter
    b.rating >= sqlc.arg('min_rating')::numeric
ORDER BY b.created_at DESC
LIMIT sqlc.arg('limit_val') OFFSET sqlc.arg('offset_val');

-- name: GetArchiveByID :one
SELECT sqlc.embed(a), sqlc.embed(s)
FROM archives a
         LEFT JOIN archive_stats s ON a.id = s.archive_id
WHERE a.id = $1;

-- name: ListArchivesByPostID :many
SELECT sqlc.embed(a), sqlc.embed(s)
FROM archives a
         LEFT JOIN archive_stats s ON a.id = s.archive_id
WHERE a.post_id = $1
ORDER BY a.created_at ASC;

-- name: UpsertArchive :one
WITH upserted_archive AS (
    INSERT INTO archives (
                          id, post_id, name, size_bytes, pages, locations, updated_at
        ) VALUES (COALESCE(sqlc.narg('id')::uuid, uuid_generate_v7()),
                  $1, $2, $3, $4, $5, NOW())
        ON CONFLICT (id) DO UPDATE SET
            name = EXCLUDED.name,
            size_bytes = EXCLUDED.size_bytes,
            pages = EXCLUDED.pages,
            locations = EXCLUDED.locations,
            updated_at = NOW()
        RETURNING *),
     stats_init AS (
         -- Ensure a stats row exists for the new/updated archive
         INSERT INTO archive_stats (archive_id, downloads)
             SELECT id, 0 FROM upserted_archive
             ON CONFLICT (archive_id) DO NOTHING)
SELECT u.id
FROM upserted_archive u;

-- name: DeleteArchive :exec
DELETE
FROM archives
WHERE id = $1;

-- name: UpdateArchiveDownloads :exec
-- This handles both increment (amount = 1) and decrement (amount = -1)
INSERT INTO archive_stats (archive_id, downloads)
VALUES ($1, $2)
ON CONFLICT (archive_id) DO UPDATE
    SET downloads = archive_stats.downloads + EXCLUDED.downloads;