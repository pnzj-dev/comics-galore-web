window.supportModal = function () {
    return {
        activeTab: 'send',
        messageContent: '',
        messageSubject: '',
        isSubmitting: false,
        formError: '',
        formSuccess: false,
        turnstileToken: null,

        // This initialization runs when the component is created
        init() {
            // Define global callback for Turnstile
            window.onTurnstileSuccess = (token) => {
                this.turnstileToken = token;
            };
        },

        async submitMessage(form) {
            this.isSubmitting = true;
            this.formError = '';
            this.formSuccess = false;

            const payload = {
                message: this.messageContent,
                subject: this.messageSubject,
                'cf-turnstile-response': this.turnstileToken
            };

            try {
                const response = await fetch('/api/v1/support/messages', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Accept': 'application/json',
                    },
                    body: JSON.stringify(payload)
                });

                if (!response.ok) {
                    const errorData = await response.json();
                    throw new Error(errorData.message || 'An unknown error occurred.');
                }

                this.formSuccess = true;
                this.messageContent = '';
                this.messageSubject = '';

                // Reset Turnstile widget
                if (window.turnstile) {
                    window.turnstile.reset();
                }
                this.turnstileToken = null;

            } catch (error) {
                this.formError = error.message;
            } finally {
                this.isSubmitting = false;
            }
        }
    }
}