package blog

import (
	"cmp"
	"comics-galore-web/cmd/web/utils"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"slices"
	"time"
)

type Image struct {
	CloudflareID string   `json:"cloudflare_id"`
	DisplayOrder int      `json:"display_order"`
	BackupS3Key  string   `json:"backup_s3_key"`
	Variants     []string `json:"variants"`
}

type PreviewList []Image

func (pl *PreviewList) MarshalJSON() ([]byte, error) {
	if len(*pl) == 0 {
		return []byte("[]"), nil
	}

	sorted := make(PreviewList, len(*pl))
	copy(sorted, *pl)

	slices.SortFunc(sorted, func(a, b Image) int {
		return cmp.Compare(a.DisplayOrder, b.DisplayOrder)
	})

	type Alias PreviewList
	return json.Marshal(Alias(sorted))
}

func (pl *PreviewList) UnmarshalJSON(data []byte) error {
	type Alias PreviewList
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Sort immediately after receiving data
	slices.SortFunc(aux, func(a, b Image) int {
		return cmp.Compare(a.DisplayOrder, b.DisplayOrder)
	})

	*pl = PreviewList(aux)
	return nil
}

type Post struct {
	ID           string      `json:"id"`
	Title        string      `json:"title"`
	AuthorName   string      `json:"authorName"`
	UploaderID   string      `json:"uploaderId"`
	Description  string      `json:"description"`
	Tags         []string    `json:"tags"`
	Cover        Image       `json:"cover"`
	Previews     PreviewList `json:"previews"`
	Rating       float32     `json:"rating"`
	LanguageCode string      `json:"languageCode"`
	Pages        int32       `json:"pages"`
	SizeBytes    int64       `json:"sizeBytes"`
	MimeTypes    []string    `json:"mimeTypes"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
	CategorySlug string      `json:"slug"`
	CategoryName string      `json:"displayName"`
	AuthViews    int64       `json:"authViews"`
	AnonViews    int64       `json:"anonViews"`
	Downloads    int64       `json:"downloads"`
	Comments     int64       `json:"comments"`
}

type Archive struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	SizeBytes int64      `json:"sizeBytes"`
	Pages     int32      `json:"pages"`
	Locations []Location `json:"locations"`
}

type Location struct {
	Role     string `json:"role"` // 'main', 'backup_1', 'backup_2'
	S3Key    string `json:"s3_key"`
	S3Bucket string `json:"s3_bucket"`
	Endpoint string `json:"endpoint,omitempty"`
}

func (a *Location) DownloadUrl() string {
	return fmt.Sprintf("/storage/%s/archive/%s", a.S3Bucket, a.S3Key)
}

func (p *Post) Slug() string {
	return utils.GenerateSlug(p.Title)
}

func (p *Post) Url() string {
	return fmt.Sprintf("/post/%s/%s", p.ID, p.Slug())
}

func (p *Post) TotalViewCount() string {
	return fmt.Sprintf("%d", p.AuthViews+p.AnonViews)
}

func (p *Post) FormatFileSize() string {
	return utils.FormatSize(p.SizeBytes)
}

func (p *Post) FormatCreatedAt() string {
	return utils.FormatDateTime(p.CreatedAt)
}

func (p *Post) FormatUpdatedAt() string {
	return utils.FormatDateTime(p.UpdatedAt)
}

func (p *Post) FormatRating() string { return fmt.Sprintf("%.1f", p.Rating) }

func (p *Post) StarClass(index int) string {
	base := "w-4 h-4 "
	idx := float32(index)
	if idx <= p.Rating {
		return base + "text-yellow-400"
	}
	if idx-0.5 <= p.Rating {
		return base + "text-yellow-400 opacity-50"
	}
	return base + "text-gray-300"
}

/*func (p *Post) SortPreviews() {
	if p == nil || len(p.Previews) < 2 {
		return
	}
	slices.SortFunc(p.Previews, func(a, b Image) int {
		return cmp.Compare(a.DisplayOrder, b.DisplayOrder)
	})
}*/
