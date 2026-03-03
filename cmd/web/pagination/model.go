package pagination

import (
	"comics-galore-web/internal/blog"
	"sort"
)

type Pagination struct {
	Posts []blog.Post `json:"posts"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
	Total int         `json:"total"`
	Pages int         `json:"pages"`
}

func NewPaginationDTO(page, limit, total, pages int, posts []blog.Post) *Pagination {
	return &Pagination{
		Page:  page,
		Total: total,
		Pages: pages,
		Limit: limit,
		Posts: posts,
	}
}

// GetPageButtonClass generates the Tailwind class string based on the current page.
func (p Pagination) GetPageButtonClass(buttonPage int) string {
	base := "px-4 py-2 rounded-lg shadow transition duration-150 ease-in-out font-medium"
	if p.Page == buttonPage {
		// Active Page: Primary color
		return base + " bg-indigo-600 text-white"
	}
	// Inactive Page: White button with border and hover effect
	return base + " bg-white text-indigo-500 border-2 border-indigo-300 hover:bg-indigo-50"
}

// CalculateVisiblePages determines which page numbers should be rendered,
// using '0' as a sentinel value for the ellipsis (...).
func (p Pagination) CalculateVisiblePages() []int {
	totalPages := p.Pages
	if totalPages <= 1 {
		return nil
	}
	currentPage := p.Page

	pages := make(map[int]bool)

	// Always include page 1 and the last page
	pages[1] = true
	pages[totalPages] = true

	// Include current page and up to 2 neighbors on each side
	for i := -2; i <= 2; i++ {
		page := currentPage + i
		if page > 0 && page <= totalPages {
			pages[page] = true
		}
	}

	// 1. Collect unique, sorted keys
	var visiblePages []int
	for page := range pages {
		visiblePages = append(visiblePages, page)
	}
	sort.Ints(visiblePages)

	// 2. Insert ellipsis (0) placeholders
	var finalPages []int
	if len(visiblePages) > 0 {
		finalPages = append(finalPages, visiblePages[0])
		for i := 1; i < len(visiblePages); i++ {
			if visiblePages[i] > visiblePages[i-1]+1 {
				finalPages = append(finalPages, 0) // Add ellipsis
			}
			finalPages = append(finalPages, visiblePages[i])
		}
	}

	return finalPages
}
