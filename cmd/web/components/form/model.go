package form

type ButtonProps struct {
	Label     string
	Variant   string
	FullWidth bool
}

type FileUploadProps struct {
	Name       string
	Label      string
	Accept     string // e.g., "image/*,application/pdf"
	Required   bool
	PresignUrl string // The backend endpoint to fetch S3 URL
	UploadPath string // The Alpine variable to store the final S3 key
	ErrorPath  string
}
