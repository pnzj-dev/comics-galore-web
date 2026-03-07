package blog

import (
	"comics-galore-web/internal/database"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"log/slog"
)

type Service interface {
	Save(ctx context.Context, params database.UpsertPostParams) (uuid.UUID, error)

	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*Post, error)
	List(ctx context.Context, limit, offset int32) ([]Post, int64, error)
	GetArchives(ctx context.Context, id uuid.UUID) ([]Archive, error)

	//IncrementView Increment view count of a post
	IncrementView(ctx context.Context, params database.IncrementPostStatsParams) error

	// ListRelated List related post by category or tags
	ListRelated(ctx context.Context, params database.ListRelatedPostsParams) ([]Post, error)

	Search(ctx context.Context, params database.SearchPostsParams) ([]Post, int64, error)
}

type service struct {
	queries *database.Queries
	logger  *slog.Logger
}

func (s *service) GetArchives(ctx context.Context, id uuid.UUID) ([]Archive, error) {
	l := s.logger.With("op", "GetArchives", "post_id", id)

	rows, err := s.queries.ListArchivesByPostID(ctx, id)
	if err != nil {
		l.Error("failed to list archives", "error", err)
		return nil, err
	}

	archives := make([]Archive, len(rows))
	for i, row := range rows {
		var locations []Location
		if len(row.Archive.Locations) > 0 {
			if err := json.Unmarshal(row.Archive.Locations, &locations); err != nil {
				l.Warn("failed to unmarshal locations", "archive_id", row.Archive.ID, "error", err)
			}
		}
		archives[i] = Archive{
			ID:        row.Archive.ID,
			Name:      row.Archive.Name,
			Locations: locations,
		}
	}
	return archives, nil
}

func (s *service) IncrementView(ctx context.Context, params database.IncrementPostStatsParams) error {
	l := s.logger.With("op", "IncrementView", "post_id", params.PostID)

	if err := s.queries.IncrementPostStats(ctx, params); err != nil {
		l.Error("failed to increment view count", "error", err)
		return err
	}

	l.Debug("view count incremented")
	return nil
}

func (s *service) ListRelated(ctx context.Context, params database.ListRelatedPostsParams) ([]Post, error) {
	l := s.logger.With("op", "ListRelated", "category_id", params.CategoryID)

	rows, err := s.queries.ListRelatedPosts(ctx, params)
	if err != nil {
		l.Error("failed to fetch related posts", "error", err)
		return nil, err
	}

	items := make([]Post, len(rows))
	for i, row := range rows {
		items[i] = *s.mapRowToPost(row.Blogpost, &row.BlogpostStat, &row.Category)
	}
	return items, nil
}

func NewService(querier *database.Queries, logger *slog.Logger) Service {
	return &service{
		queries: querier,
		logger:  logger.With("component", "blog_service"),
	}
}

func (s *service) Save(ctx context.Context, params database.UpsertPostParams) (uuid.UUID, error) {
	l := s.logger.With("op", "Save", "title", params.Title)
	updated, err := s.queries.UpsertPost(ctx, params)
	if err != nil {
		l.Error("failed to upsert post", "error", err)
		return uuid.Nil, err
	}
	l.Info("post persisted", "id", updated.String())
	return updated, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	l := s.logger.With("op", "Delete", "id", id)
	if err := s.queries.DeletePost(ctx, id); err != nil {
		l.Error("failed to delete post", "error", err)
		return err
	}
	l.Warn("post deleted")
	return nil
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (*Post, error) {
	l := s.logger.With("op", "Get", "id", id)
	row, err := s.queries.GetPost(ctx, id)
	if err != nil {
		l.Error("failed to fetch post", "error", err)
		return nil, err
	}
	return s.mapRowToPost(row.Blogpost, &row.BlogpostStat, &row.Category), nil
}

func (s *service) List(ctx context.Context, limit, offset int32) ([]Post, int64, error) {
	l := s.logger.With("op", "List", "limit", limit, "offset", offset)
	rows, err := s.queries.ListPosts(ctx, database.ListPostsParams{Limit: limit, Offset: offset})
	if err != nil {
		l.Error("failed to list posts", "error", err)
		return nil, 0, err
	}

	count, _ := s.queries.CountPosts(ctx, database.CountPostsParams{SearchQuery: ""})
	l.Info("posts retrieved", "count", len(rows), "total", count)

	items := make([]Post, len(rows))
	for i, row := range rows {
		items[i] = *s.mapRowToPost(row.Blogpost, &row.BlogpostStat, &row.Category)
	}
	return items, count, nil
}

func (s *service) Search(ctx context.Context, params database.SearchPostsParams) ([]Post, int64, error) {
	l := s.logger.With("op", "Search", "query", params.SearchQuery, "tags_len", len(params.Tags))

	rows, err := s.queries.SearchPosts(ctx, params)
	if err != nil {
		l.Error("search query failed", "error", err)
		return nil, 0, err
	}

	count, _ := s.queries.CountPosts(ctx, database.CountPostsParams{
		SearchQuery:       params.SearchQuery,
		SearchTitle:       params.SearchTitle,
		SearchAuthor:      params.SearchAuthor,
		SearchDescription: params.SearchDescription,
		SearchCategory:    params.SearchCategory,
		Tags:              params.Tags,
		MatchAll:          params.MatchAll,
	})

	l.Debug("search completed", "results", len(rows), "total", count)

	items := make([]Post, len(rows))
	for i, row := range rows {
		items[i] = *s.mapRowToPost(row.Blogpost, &row.BlogpostStat, &row.Category)
	}
	return items, count, nil
}

func (s *service) mapRowToPost(post database.Blogpost, stats *database.BlogpostStat, category *database.Category) *Post {

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
		CategoryID:   category.ID.String(),
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
