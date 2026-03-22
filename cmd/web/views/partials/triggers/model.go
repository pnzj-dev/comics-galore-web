package triggers

type Props struct {
	Url             string
	Size            string
	Title           string
	Label           string
	Classes         string
	ExtraClick      string
	CustomSkeleton  string
	ConfirmClose    bool
	CloseOnBackdrop bool
}

func NewLoginProps() Props {
	return Props{
		Url:             "/auth/modal/login",
		Title:           "Sign In",
		Size:            "auto",
		Label:           "Sign In",
		Classes:         "text-sm text-gray-600 hover:text-indigo-600 font-medium px-3 py-1.5 rounded-lg hover:bg-gray-100 transition-colors",
		ExtraClick:      "mobileOpen = false",
		CloseOnBackdrop: true,
		ConfirmClose:    true,
	}
}

func NewSignupProps() Props {
	return Props{
		Url:             "/auth/modal/signup",
		Size:            "auto",
		Title:           "Create Account",
		Label:           "Sign Up",
		Classes:         "text-sm text-white bg-indigo-600 hover:bg-indigo-700 font-medium px-4 py-1.5 rounded-lg transition-colors",
		ExtraClick:      "mobileOpen = false",
		ConfirmClose:    true,
		CloseOnBackdrop: true,
	}
}
