package views

import "fmt"

type ModalDefaults struct {
	Size  string // "sm" | "md" | "lg" | "xl"
	Title string
}

func layoutXData(modal ModalDefaults) string {
	size := modal.Size
	if size == "" {
		size = "md"
	}
	title := modal.Title
	if title == "" {
		title = "Modal"
	}
	return fmt.Sprintf(`{
        ui: {
            modal: {
                open: false,
                size: %q,
                title: %q,
                loading: false
            }
        },
        turnstileToken: '',
        turnstileReady: false
    }`, size, title)
}

func layoutXInit() string {
	return `
        if (typeof window.turnstile !== 'undefined') {
            window.turnstile.ready(() => { turnstileReady = true })
        } else {
            document.querySelector('script[src*="turnstile"]')?.addEventListener('load', () => {
                window.turnstile.ready(() => { turnstileReady = true })
            })
        }
    `
}
