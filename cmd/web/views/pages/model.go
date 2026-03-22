package pages

import "comics-galore-web/cmd/web/components/footer"

var footerSection = []footer.FooterSection{
	{Title: "Explore", Links: []footer.FooterLink{
		{Label: "New Releases", Href: "/new"},
		{Label: "Top Rated", Href: "/top"},
		{Label: "Genres", Href: "/genres"},
	}},
	{Title: "Account", Links: []footer.FooterLink{
		{Label: "My Collection", Href: "/collection"},
		{Label: "Wishlist", Href: "/wishlist"},
		{Label: "Settings", Href: "/settings"},
	}},
	{Title: "Support", Links: []footer.FooterLink{
		{Label: "FAQ", Href: "/faq"},
		{Label: "Contact Us", Href: "/contact"},
	}}}

var footerLink = []footer.FooterLink{
	{Label: "Twitter", Href: "https://twitter.com/comicsgalore"},
	{Label: "Instagram", Href: "https://instagram.com/comicsgalore"}}
