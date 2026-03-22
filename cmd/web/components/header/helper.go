package header

import "github.com/a-h/templ"

func getNavbarClass(isActive bool) templ.CSSClasses {
	return templ.Classes(
		"inline-block px-4 py-3 text-sm font-medium border-b-2 transition-colors",
		templ.KV("border-indigo-600 text-indigo-600", isActive),
		templ.KV("border-transparent text-gray-600 hover:text-indigo-600 hover:border-indigo-300", !isActive),
	)
}

func getNavbarClassMobile(isActive bool) templ.CSSClasses {
	return templ.Classes("block px-3 py-2 rounded-lg text-sm font-medium transition-colors",
		templ.KV("bg-indigo-50 text-indigo-600", isActive),
		templ.KV("text-gray-600 hover:bg-gray-50 hover:text-indigo-600", !isActive))
}
