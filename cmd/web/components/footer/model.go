package footer

type FooterLink struct {
	Label string
	Href  string
}

type FooterSection struct {
	Title string
	Links []FooterLink
}
