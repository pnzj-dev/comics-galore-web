package button

import "github.com/a-h/templ"

func FavoriteButtonXData(id string, isFavorite bool) (string, error) {

	type FavoriteButton struct {
		ID         string `json:"id"`
		IsLoading  bool   `json:"isLoading"`
		IsFavorite bool   `json:"isFavorite"`
	}

	return templ.JSONString(FavoriteButton{
		ID:         id,
		IsLoading:  false,
		IsFavorite: isFavorite,
	})
}
