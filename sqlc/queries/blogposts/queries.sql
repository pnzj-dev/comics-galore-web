-- name: ListRelatedPosts :many
WITH random_posts AS (SELECT id
                      FROM blogposts
                      WHERE (blogposts.category_id = sqlc.arg('category_id')
                          OR blogposts.tags && sqlc.arg('tags')::text[])
                        AND blogposts.id != sqlc.arg('id')
                      ORDER BY RANDOM()
                      LIMIT 6)
SELECT sqlc.embed(b), sqlc.embed(c), sqlc.embed(s)
FROM blogposts b
         JOIN categories c ON b.category_id = c.id
         LEFT JOIN blogpost_stats s ON b.id = s.post_id
WHERE b.id IN (SELECT id FROM random_posts)
ORDER BY b.created_at DESC;


-- name: UpsertPost :one
INSERT INTO blogposts (id, title, author_name, uploader_id, category_id, description,
                       tags, cover, previews, rating, language_code, pages,
                       size_bytes, mime_types, updated_at)
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
RETURNING id;

-- name: DeletePost :exec
DELETE
FROM blogposts
WHERE id = $1;

-- name: GetPost :one
SELECT sqlc.embed(b), sqlc.embed(c), sqlc.embed(s)
FROM blogposts b
         JOIN categories c ON b.category_id = c.id
         LEFT JOIN blogpost_stats s ON b.id = s.post_id
WHERE b.id = $1;

-- name: ListPosts :many
SELECT sqlc.embed(b), sqlc.embed(c), sqlc.embed(s)
FROM blogposts b
         JOIN categories c ON b.category_id = c.id
         LEFT JOIN blogpost_stats s ON b.id = s.post_id
ORDER BY b.created_at DESC
LIMIT $1 OFFSET $2;

-- name: SearchPosts :many
SELECT sqlc.embed(b), sqlc.embed(c), sqlc.embed(s)
FROM blogposts b
         JOIN categories c ON b.category_id = c.id
         LEFT JOIN blogpost_stats s ON b.id = s.post_id
WHERE (sqlc.arg('search_query')::text = '' OR (
    (sqlc.arg('search_title')::bool AND b.title ILIKE '%' || sqlc.arg('search_query')::text || '%') OR
    (sqlc.arg('search_author')::bool AND b.author_name ILIKE '%' || sqlc.arg('search_query')::text || '%') OR
    (sqlc.arg('search_description')::bool AND b.description ILIKE '%' || sqlc.arg('search_query')::text || '%') OR
    (sqlc.arg('search_category')::bool AND c.display_name ILIKE '%' || sqlc.arg('search_query')::text || '%')
    ))
  AND (cardinality(sqlc.arg('tags')::text[]) = 0 OR
       (sqlc.arg('match_all')::bool AND b.tags @> sqlc.arg('tags')::text[]) OR
       (NOT sqlc.arg('match_all')::bool AND b.tags && sqlc.arg('tags')::text[])
    )
ORDER BY b.created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountPosts :one
SELECT count(*)
FROM blogposts b
         JOIN categories c ON b.category_id = c.id
WHERE (sqlc.arg('search_query')::text = '' OR (
    (sqlc.arg('search_title')::bool AND b.title ILIKE '%' || sqlc.arg('search_query')::text || '%') OR
    (sqlc.arg('search_author')::bool AND b.author_name ILIKE '%' || sqlc.arg('search_query')::text || '%') OR
    (sqlc.arg('search_description')::bool AND b.description ILIKE '%' || sqlc.arg('search_query')::text || '%') OR
    (sqlc.arg('search_category')::bool AND c.name ILIKE '%' || sqlc.arg('search_query')::text || '%')
    ))
  AND (cardinality(sqlc.arg('tags')::text[]) = 0 OR
       (sqlc.arg('match_all')::bool AND b.tags @> sqlc.arg('tags')::text[]) OR
       (NOT sqlc.arg('match_all')::bool AND b.tags && sqlc.arg('tags')::text[])
    );

-- *** ARCHIVE QUERIES ***
-- name: GetArchive :one
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
    INSERT INTO archives (id, post_id, name, size_bytes, pages, locations, updated_at)
        VALUES (COALESCE(sqlc.narg('id')::uuid, uuid_generate_v7()), $1, $2, $3, $4, $5, NOW())
        ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, size_bytes = EXCLUDED.size_bytes, pages = EXCLUDED.pages, locations = EXCLUDED.locations, updated_at = NOW()
        RETURNING *)
INSERT
INTO archive_stats (archive_id, downloads)
SELECT id, 0
FROM upserted_archive
ON CONFLICT (archive_id) DO NOTHING
RETURNING (SELECT id FROM upserted_archive);

-- name: DeleteArchive :exec
DELETE
FROM archives
WHERE id = $1;

-- *** STATISTICS & COUNTERS ***
-- name: IncrementPostStats :exec
INSERT INTO blogpost_stats (post_id, auth_views, anon_views)
VALUES ($1, $2, $3)
ON CONFLICT (post_id)
    DO UPDATE SET auth_views = blogpost_stats.auth_views + EXCLUDED.auth_views,
                  anon_views = blogpost_stats.anon_views + EXCLUDED.anon_views;

-- name: IncrementArchiveDownloads :exec
INSERT INTO archive_stats (archive_id, downloads)
VALUES ($1, $2)
ON CONFLICT (archive_id)
    DO UPDATE SET downloads = archive_stats.downloads + EXCLUDED.downloads;

-- name: SocialEngagementUpdate :exec
INSERT INTO social_engagement (date, comments, messages, reactions)
VALUES ($1, $2, $3, $4)
ON CONFLICT (date)
    DO UPDATE SET comments   = EXCLUDED.comments,
                  messages   = EXCLUDED.messages,
                  reactions  = EXCLUDED.reactions,
                  updated_at = NOW();