package pagination

import "fmt"

func pageURL(endpoint string, page int) string {
	return fmt.Sprintf("%s?page=%d", endpoint, page)
}

func hxPushURL(push bool, endpoint string, page int) string {
	if push {
		return pageURL(endpoint, page)
	}
	return "false"
}
