package error

import "fmt"

type message struct {
	code   string
	text   string
	field  string
	status int
}

// Error implements the error interface
func (m *message) Error() string {
	if m.field != "" {
		return fmt.Sprintf("[%s] %s: %s", m.code, m.field, m.text)
	}
	return fmt.Sprintf("[%s] %s", m.code, m.text)
}

// New creates a basic error message
func New(code, text string, status int) error {
	return &message{
		code:   code,
		text:   text,
		status: status,
	}
}

// NewField creates an error tied to a specific form field
func NewField(code, text, field string, status int) error {
	return &message{
		code:   code,
		text:   text,
		field:  field,
		status: status,
	}
}

// Status returns the HTTP status code associated with the error
func (m *message) Status() int {
	return m.status
}
