package cloudflare

import "time"

type ImageResponse struct {
	Result   ImageResult    `json:"result"`
	Success  bool           `json:"success"`
	Errors   []ResponseInfo `json:"errors"`
	Messages []ResponseInfo `json:"messages"`
}

type ImageResult struct {
	ID                string            `json:"id"`
	Filename          string            `json:"filename"`
	Meta              map[string]string `json:"meta"`
	RequireSignedURLs bool              `json:"requireSignedURLs"`
	Uploaded          time.Time         `json:"uploaded"`
	Variants          []string          `json:"variants"`
}

type ResponseInfo struct {
	Code             int    `json:"code"`
	Message          string `json:"message"`
	DocumentationURL string `json:"documentation_url,omitempty"`
	Source           struct {
		Pointer string `json:"pointer,omitempty"`
	} `json:"source,omitempty"`
}

type ListImagesResponse struct {
	Errors   []ResponseInfo   `json:"errors"`
	Messages []ResponseInfo   `json:"messages"`
	Success  bool             `json:"success"`
	Result   ListImagesResult `json:"result"`
}

type ListImagesResult struct {
	Images []ImageResult `json:"images"`
}

type SigningKey struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SigningKeysResponse struct {
	Success bool `json:"success"`
	Result  struct {
		Keys []SigningKey `json:"keys"`
	} `json:"result"`
	Errors []ResponseInfo `json:"errors"`
}

type StatsResponse struct {
	Success bool `json:"success"`
	Result  struct {
		Count struct {
			Allowed int `json:"allowed"`
			Current int `json:"current"`
		} `json:"count"`
	} `json:"result"`
	Errors []ResponseInfo `json:"errors"`
}

type VariantOptions struct {
	Fit      string `json:"fit"` // e.g., "scale-down", "contain", "cover"
	Height   int    `json:"height"`
	Width    int    `json:"width"`
	Metadata string `json:"metadata"` // "none", "keep", "copyright"
}

type Variant struct {
	ID                     string         `json:"id"`
	Options                VariantOptions `json:"options"`
	NeverRequireSignedURLs bool           `json:"neverRequireSignedURLs"`
}

type VariantResponse struct {
	Success bool           `json:"success"`
	Errors  []ResponseInfo `json:"errors"`
	Result  struct {
		Variant  *Variant           `json:"variant,omitempty"`  // For single variant ops
		Variants map[string]Variant `json:"variants,omitempty"` // For List variants
	} `json:"result"`
}

type TurnstileResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts,omitempty"` // ISO8601 timestamp
	Hostname    string   `json:"hostname,omitempty"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
	Action      string   `json:"action,omitempty"`
	CData       string   `json:"cdata,omitempty"`
}
