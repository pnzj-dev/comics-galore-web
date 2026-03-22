package badge

import "github.com/a-h/templ"

func badgeClasses(props Props) templ.CSSClasses {
	sizeClass := "text-xs px-2.5 py-0.5"
	switch props.Size {
	case "xs":
		sizeClass = "text-[10px] px-2 py-0.5"
	case "md":
		sizeClass = "text-sm px-3 py-1"
	}

	variantColors := map[Variant]string{
		VariantPurple: "bg-purple-100 text-purple-800 border-purple-200",
		VariantBlue:   "bg-blue-100 text-blue-800 border-blue-200",
		VariantGreen:  "bg-green-100 text-green-800 border-green-200",
		VariantYellow: "bg-yellow-100 text-yellow-800 border-yellow-200",
		VariantRed:    "bg-red-100 text-red-800 border-red-200",
		VariantGray:   "bg-gray-100 text-gray-800 border-gray-200",
	}

	variantClass, ok := variantColors[props.Variant]
	if !ok || variantClass == "" {
		variantClass = "bg-gray-100 text-gray-800 border-gray-200"
	}

	if props.Outline {
		switch props.Variant {
		case VariantPurple:
			variantClass = "bg-white text-purple-800 border border-purple-200"
		case VariantBlue:
			variantClass = "bg-white text-blue-800 border border-blue-200"
		case VariantGreen:
			variantClass = "bg-white text-green-800 border border-green-200"
		case VariantYellow:
			variantClass = "bg-white text-yellow-800 border border-yellow-200"
		case VariantRed:
			variantClass = "bg-white text-red-800 border border-red-200"
		default:
			variantClass = "bg-white text-gray-800 border border-gray-200"
		}
	}

	base := "inline-flex items-center rounded-full font-medium transition-colors"
	return templ.Classes(base, sizeClass, variantClass, props.Margin)
}
