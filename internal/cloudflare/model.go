package cloudflare

import "time"

type ImageResponse struct {
	Result   ImageResult `json:"result"`
	Success  bool        `json:"success"`
	Errors   []Error     `json:"errors"`
	Messages []Message   `json:"messages"`
}

type ImageResult struct {
	ID                string            `json:"id"`
	Filename          string            `json:"filename"`
	Meta              map[string]string `json:"meta"`
	RequireSignedURLs bool              `json:"requireSignedURLs"`
	Uploaded          time.Time         `json:"uploaded"`
	Variants          []string          `json:"variants"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type Message struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ListImagesResponse struct {
	Errors   []Error          `json:"errors"`
	Messages []Message        `json:"messages"`
	Success  bool             `json:"success"`
	Result   ListImagesResult `json:"result"`
}

type ListImagesResult struct {
	Images []ImageResult `json:"images"`
}
