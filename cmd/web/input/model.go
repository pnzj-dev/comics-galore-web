package input

type SelectOption struct {
	Value    int    `json:"value"`
	Title    string `json:"title"`
	Disabled bool   `json:"disabled"`
}
