package icons

func starClasses(props IconProps) string {
	if props.Class == "" {
		return "w-6 h-6"
	}
	return props.Class
}

func closeClasses(props IconProps) string {
	sizeClass := "h-5 w-5" // default
	switch props.Size {
	case "sm":
		sizeClass = "h-4 w-4"
	case "lg":
		sizeClass = "h-6 w-6"
	}

	base := "shrink-0 " + sizeClass
	if props.Class != "" {
		return base + " " + props.Class
	}
	return base
}
