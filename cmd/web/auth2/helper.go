package auth2

func GetActiveTabClass(isActive bool) string {
	if isActive {
		return "tab-active bg-white text-black"
	}
	return ""
}
