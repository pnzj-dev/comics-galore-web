package blog

import (
	"comics-galore-web/internal/database"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"log/slog"
)

type Service interface {
	Save(ctx context.Context, params database.UpsertBlogPostParams) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Post, error)
	List(ctx context.Context, limit, offset int32) ([]Post, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListArchives(ctx context.Context, postId uuid.UUID) ([]Archive, error)
	IncrementPostView(ctx context.Context, params database.IncrementPostViewParams) error
}

type service struct {
	queries *database.Queries
	logger  *slog.Logger
}

func (s *service) IncrementPostView(ctx context.Context, params database.IncrementPostViewParams) error {
	// 1. Create a contextual logger for this operation
	l := s.logger.With(
		"op", "IncrementPostView",
		"post_id", params.PostID,
		"auth_views", params.AuthViews,
		"anon_views", params.AnonViews,
	)

	// 2. Execute query
	err := s.queries.IncrementPostView(ctx, params)
	if err != nil {
		// Log the error with structured fields for easier filtering in Grafana/Loki
		l.Error("failed to increment post view", "error", err)
		return fmt.Errorf("database error: %w", err)
	}

	// 3. Optional: Trace successful high-value operations at Debug level
	l.Debug("post view incremented successfully")

	return nil
}

func NewService(queries *database.Queries, logger *slog.Logger) Service {
	return &service{
		queries: queries,
		logger:  logger,
	}
}

func (s *service) ListArchives(ctx context.Context, postId uuid.UUID) ([]Archive, error) {
	l := s.logger.With("post_id", postId, "op", "ListArchives")

	rows, err := s.queries.ListArchivesByPostID(ctx, postId)
	if err != nil {
		l.Error("failed to list archives", "error", err)
		return nil, fmt.Errorf("list archives: %w", err)
	}

	archives := make([]Archive, len(rows))
	for i, row := range rows {
		var locations []Location
		if len(row.Archive.Locations) > 0 {
			if err := json.Unmarshal(row.Archive.Locations, &locations); err != nil {
				// We log the specific archive ID that failed, but don't break the whole loop
				l.Error("failed to unmarshal archive locations",
					"archive_id", row.Archive.ID,
					"error", err)
			}
		}

		archives[i] = Archive{
			ID:        row.Archive.ID,
			Name:      row.Archive.Name,
			SizeBytes: row.Archive.SizeBytes,
			Pages:     row.Archive.Pages,
			Locations: locations,
		}
	}

	l.Debug("archives retrieved", "count", len(archives))
	return archives, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Post, error) {
	l := s.logger.With("id", id, "op", "GetByID")

	row, err := s.queries.GetPostByID(ctx, id)
	if err != nil {
		l.Error("blog post not found in database", "error", err)
		return nil, fmt.Errorf("get post: %w", err)
	}

	return s.mapRowToPost(row.Blogpost, row.BlogpostStat, row.Category), nil
}

func (s *service) List(ctx context.Context, limit, offset int32) ([]Post, error) {
	l := s.logger.With("op", "List", "limit", limit, "offset", offset)

	rows, err := s.queries.ListPosts(ctx, database.ListPostsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		l.Error("failed to fetch posts page", "error", err)
		return nil, fmt.Errorf("list posts: %w", err)
	}

	items := make([]Post, 0, len(rows))
	for _, row := range rows {
		items = append(items, *s.mapRowToPost(row.Blogpost, row.BlogpostStat, row.Category))
	}

	l.Info("posts batch retrieved", "count", len(items))
	return items, nil
}

func (s *service) Save(ctx context.Context, params database.UpsertBlogPostParams) (uuid.UUID, error) {
	l := s.logger.With("op", "Save", "title", params.Title)

	updatedPost, err := s.queries.UpsertBlogPost(ctx, params)
	if err != nil {
		l.Error("database upsert failed", "error", err)
		return uuid.Nil, fmt.Errorf("save post: %w", err)
	}

	// Cleaner UUID conversion from pgtype
	resID, _ := uuid.Parse(updatedPost.ID.String())

	l.Info("blog post persistence successful", "id", resID)
	return resID, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	l := s.logger.With("id", id, "op", "Delete")

	if err := s.queries.DeletePost(ctx, id); err != nil {
		l.Error("failed to remove post from database", "error", err)
		return fmt.Errorf("delete post: %w", err)
	}

	l.Warn("blog post permanently deleted")
	return nil
}

func (s *service) mapRowToPost(post database.Blogpost, stats database.BlogpostStat, category database.Category) *Post {
	// We create a sub-logger for the mapping to track potential JSON corruption
	l := s.logger.With("post_id", post.ID, "op", "mapRowToPost")

	var cover Image
	if len(post.Cover) > 0 {
		if err := json.Unmarshal(post.Cover, &cover); err != nil {
			l.Error("invalid JSON in cover field", "error", err)
		}
	}

	var previews []Image
	if len(post.Previews) > 0 {
		if err := json.Unmarshal(post.Previews, &previews); err != nil {
			l.Error("invalid JSON in previews field", "error", err)
		}
	}

	return &Post{
		ID:           post.ID.String(),
		Title:        post.Title,
		AuthorName:   post.AuthorName,
		UploaderID:   post.UploaderID,
		Description:  post.Description,
		Tags:         post.Tags,
		Cover:        cover,
		Previews:     previews,
		Rating:       NumericToFloat32(post.Rating),
		LanguageCode: post.LanguageCode,
		Pages:        post.Pages,
		SizeBytes:    post.SizeBytes,
		MimeTypes:    post.MimeTypes,
		CreatedAt:    post.CreatedAt.Time,
		UpdatedAt:    post.UpdatedAt.Time,
		CategorySlug: category.Slug,
		CategoryName: category.DisplayName,
		AuthViews:    stats.AuthViews,
		AnonViews:    stats.AnonViews,
		Downloads:    stats.Downloads,
		Comments:     stats.Comments,
	}
}

func NumericToFloat32(n pgtype.Numeric) float32 {
	if !n.Valid {
		return 0.0
	}
	f, err := n.Float64Value()
	if err != nil {
		return 0.0
	}
	return float32(f.Float64)
}
