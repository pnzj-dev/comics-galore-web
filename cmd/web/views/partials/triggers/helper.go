package triggers

import (
	"fmt"
	"github.com/a-h/templ"
)

func getBeforeRequest(props Props) string {
	skeleton := props.CustomSkeleton
	if skeleton == "" {
		skeleton = getSkeleton()
	}
	return fmt.Sprintf(`
        let modal = Alpine.store('ui').modal;
        modal.title=%q;
        modal.size=%q;
        modal.closeOnBackdrop=%t;
        modal.confirmClose=%t;
        modal.open=true;
        modal.loading=true;
        modal.isDirty=false;
        document.getElementById('modal-content').innerHTML = %q;
        %s`, props.Title, props.Size, props.CloseOnBackdrop, props.ConfirmClose, skeleton, props.ExtraClick)
}

func getAfterRequest() string {
	return fmt.Sprintf(`
		let modal = Alpine.store('ui').modal;
    	modal.loading = false;
    	modal.initialData = JSON.stringify(Object.fromEntries(new FormData(document.querySelector('#modal-content form'))));`)
}

func getError(errorState string) string {
	return fmt.Sprintf(`
		let modal = Alpine.store('ui').modal;
		modal.loading = false;document.getElementById('modal-content').innerHTML = %q;`, errorState)
}

func getModalAttributes(props Props, errorState string) templ.Attributes {
	return templ.Attributes{
		"hx-on::before-request": getBeforeRequest(props),
		"hx-on::after-request":  getAfterRequest(),
		"hx-on::error":          getError(errorState),
	}
}
