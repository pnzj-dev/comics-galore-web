package pagination

type Props struct {
	CurrentPage int
	TotalPages  int
	HxTarget    string
	HxEndpoint  string
	HxPushURL   bool
}

func (p Props) pages() []int {
	var result []int
	for i := 1; i <= p.TotalPages; i++ {
		result = append(result, i)
	}
	return result
}

// Returns a window of page numbers with ellipsis markers (-1)
func (p Props) pageWindow() []int {
	total := p.TotalPages
	cur := p.CurrentPage
	var pages []int

	if total <= 7 {
		for i := 1; i <= total; i++ {
			pages = append(pages, i)
		}
		return pages
	}

	pages = append(pages, 1)
	if cur > 3 {
		pages = append(pages, -1) // ellipsis
	}
	for i := maximum(2, cur-1); i <= minimum(total-1, cur+1); i++ {
		pages = append(pages, i)
	}
	if cur < total-2 {
		pages = append(pages, -1) // ellipsis
	}
	pages = append(pages, total)
	return pages
}

func maximum(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minimum(a, b int) int {
	if a < b {
		return a
	}
	return b
}
