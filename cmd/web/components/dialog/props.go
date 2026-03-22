package dialog

type HeaderProps struct {
	Title     string `json:"title"`
	Subtitle  string `json:"subtitle"`
	ShowClose bool   `json:"showClose"`
	Icon      bool   `json:"icon"`
}
