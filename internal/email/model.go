package email

// Request defines the payload for sending a basic email.
type Request struct {
	ToName      string
	ToEmail     string
	Subject     string
	PlainText   string
	HTMLContent string
}

// TemplateRequest defines the payload for SendGrid Dynamic Templates.
type TemplateRequest struct {
	ToName       string
	ToEmail      string
	TemplateID   string
	TemplateData map[string]interface{}
}
