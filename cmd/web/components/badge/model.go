package badge

const (
	VariantDefault Variant = "default"
	VariantPurple  Variant = "purple"
	VariantBlue    Variant = "blue"
	VariantGreen   Variant = "green"
	VariantYellow  Variant = "yellow"
	VariantRed     Variant = "red"
	VariantGray    Variant = "gray"
)

type Variant string

type Props struct {
	Text    string
	Variant Variant
	Size    string // "xs", "sm", "md" (default)
	Outline bool
	Margin  string
}
