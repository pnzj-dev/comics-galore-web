package dialog

import "github.com/a-h/templ"

func getModalStyles() templ.Attributes {
	return templ.Attributes{
		"class": "bg-white rounded-xl shadow-2xl overflow-hidden transform transition-all duration-200",
		":class": `{
            'scale-95 opacity-0': !$store.ui.modal.open,
            'scale-100 opacity-100': $store.ui.modal.open,
            'w-fit': $store.ui.modal.size === 'auto',
            'max-w-sm': $store.ui.modal.size === 'sm',
            'max-w-md': $store.ui.modal.size === 'md',
            'max-w-lg': $store.ui.modal.size === 'lg',
            'max-w-2xl': $store.ui.modal.size === 'xl',
            'w-full': $store.ui.modal.size !== 'auto'
        }`,
	}
}
