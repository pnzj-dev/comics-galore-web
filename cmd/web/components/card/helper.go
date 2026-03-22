package card

import "fmt"

func cardXData(isFavorite bool, rating float32, downloads int64) string {
	if rating == 0 {
		rating = 0.0
	}
	if downloads == 0 {
		downloads = 0
	}
	return fmt.Sprintf(`{isFavorite:%t,userRating:%g,pendingRating:0,isDownloading:false,downloadCount:%d}`,
		isFavorite, rating, downloads)
}

func cardXInit() string {
	return fmt.Sprintf(`() => {}`)
}

func starBindClass() string {
	return "{'text-yellow-400 fill-current': (pendingRating || userRating) >= (star + 1), 'text-gray-300': (pendingRating || userRating) < (star + 1)}"
}

func dispatchFavorite(postID string) string {
	return fmt.Sprintf("isFavorite = !isFavorite; $dispatch('favorite-toggle', { postId: '%s', isFavorite: isFavorite })", postID)
}

func dispatchRating(postID string) string {
	return fmt.Sprintf("userRating = star + 1; pendingRating = star + 1; $dispatch('rate-post', { postId: '%s', rating: userRating })", postID)
}

func dispatchDownload(postID string) string {
	return fmt.Sprintf("isDownloading = true; downloadCount++; $dispatch('download-post', { postId: '%s' }); setTimeout(() => isDownloading = false, 1500)", postID)
}
