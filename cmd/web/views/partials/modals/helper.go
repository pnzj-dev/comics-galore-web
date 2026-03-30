package modals

import (
	"encoding/json"
	"fmt"
)

func signinXData(siteKey string, turnstileEnabled bool) string {
	safeSiteKey, err := json.Marshal(siteKey)
	if err != nil {
		safeSiteKey = []byte(`""`)
	}

	// We use a raw string literal to make the JS more readable
	return fmt.Sprintf(`{
    email: '',
    password: '',
    turnstileToken: '',
    loading: false,
    turnstileEnabled: %t, // New flag passed from Go
    formErrors: { email: '', password: '', global: '' },

    validate() {
        this.formErrors = { email: '', password: '', global: '' };
        let valid = true;
        if (!this.email.trim()) {
            this.formErrors.email = 'Email is required';
            valid = false;
        } else if (!this.email.includes('@')) {
            this.formErrors.email = 'Please enter a valid email';
            valid = false;
        }
        if (!this.password || this.password.length < 8) {
            this.formErrors.password = 'Password must be at least 8 characters';
            valid = false;
        }
        return valid;
    },

    submitForm() {
       if (!this.validate()) return;
       
       this.loading = true;
       this.formErrors.global = "";

       // LOGIC BRANCH: If Turnstile is disabled, skip the challenge
       if (!this.turnstileEnabled) {
          this.turnstileToken = "disabled-token"; // Placeholder for backend
		  // Ensure Alpine finishes internal updates before HTMX triggers
		  this.$nextTick(() => {
		  	htmx.trigger('#login-form-trigger', 'submit-auth');
		  });
		  return;
       }
    
       if (typeof window.turnstile === 'undefined') {
          this.formErrors.global = 'Security service unavailable.';
          this.loading = false;
          return;
       }

       window.turnstile.execute('#turnstile-container', {
          sitekey: %s,
          action: 'login',
          callback: (token) => {
             this.turnstileToken = token;
             //htmx.trigger('#login-form-trigger', 'submit-auth');
			 this.$nextTick(() => {
                const el = document.querySelector('#login-form-trigger');
                if (el) htmx.trigger(el, 'submit-auth');
             });
          },
          'expired-callback': () => {
             this.loading = false;
             this.turnstileToken = '';
             this.formErrors.global = 'Token expired. Please try again.';
          },
          'error-callback': () => {
             this.loading = false;
             this.formErrors.global = 'Verification failed.';
          }
       });
    }
}`, turnstileEnabled, string(safeSiteKey))
}

func resetPasswordXData(siteKey string, turnstileEnabled bool) string {
	safeSiteKey, err := json.Marshal(siteKey)
	if err != nil {
		safeSiteKey = []byte(`""`)
	}

	return fmt.Sprintf(`{
        email: '',
        turnstileToken: '',
        loading: false,
        turnstileEnabled: %t,
        formErrors: { email: '', global: '' },

        validate() {
            this.formErrors = { email: '', global: '' };
            let valid = true;
            if (!this.email.trim()) {
                this.formErrors.email = 'Email is required';
                valid = false;
            } else if (!this.email.includes('@')) {
                this.formErrors.email = 'Please enter a valid email';
                valid = false;
            }
            return valid;
        },

        submitForm() {
            if (!this.validate()) return;
            
            this.loading = true;
            this.formErrors.global = "";

            // Bypass Turnstile if disabled in config
            if (!this.turnstileEnabled) {
                this.turnstileToken = "disabled-token";
				// Ensure Alpine finishes internal updates before HTMX triggers
				this.$nextTick(() => {
					htmx.trigger('#forgot-form-trigger', 'submit-forgot');
				});
				return;
            }

            if (typeof window.turnstile === 'undefined') {
                this.formErrors.global = 'Security service unavailable.';
                this.loading = false;
                return;
            }

            // Programmatic execution (Safe pattern for dynamic components)
            window.turnstile.execute('#turnstile-forgot', {
                sitekey: %s,
                action: 'forgot_password',
                callback: (token) => {
                    this.turnstileToken = token;
                    //htmx.trigger('#forgot-form-trigger', 'submit-forgot');
					this.$nextTick(() => {
						const el = document.querySelector('#forgot-form-trigger');
						if (el) htmx.trigger(el, 'submit-forgot');
					});
                },
                'expired-callback': () => {
                    this.loading = false;
                    this.turnstileToken = '';
                    this.formErrors.global = 'Security token expired. Please try again.';
                },
                'error-callback': () => {
                    this.loading = false;
                    this.formErrors.global = 'Verification failed. Please try again.';
                }
            });
        }
}`, turnstileEnabled, string(safeSiteKey))
}

func signupXData(siteKey string, turnstileEnabled bool) string {
	safeSiteKey, err := json.Marshal(siteKey)
	if err != nil {
		safeSiteKey = []byte(`""`)
	}

	return fmt.Sprintf(`{
        name: '',
        email: '',
        password: '',
        passwordConfirm: '',
        turnstileToken: '',
        loading: false,
        turnstileEnabled: %t,
        formErrors: { name: '', email: '', password: '', passwordConfirm: '', global: '' },

        validate() {
            this.formErrors = { name: '', email: '', password: '', passwordConfirm: '', global: '' };
            let valid = true;
            if (!this.name.trim()) {
                this.formErrors.name = 'Full name is required';
                valid = false;
            }
            if (!this.email.trim()) {
                this.formErrors.email = 'Email is required';
                valid = false;
            } else if (!this.email.includes('@')) {
                this.formErrors.email = 'Please enter a valid email';
                valid = false;
            }
            if (!this.password) {
                this.formErrors.password = 'Password is required';
                valid = false;
            } else if (this.password.length < 8) {
                this.formErrors.password = 'Password must be at least 8 characters';
                valid = false;
            }
            if (this.password !== this.passwordConfirm) {
                this.formErrors.passwordConfirm = 'Passwords do not match';
                valid = false;
            }
            return valid;
        },

        submitForm() {
			console.log("1. Alpine: starting validation...");
            if (!this.validate()) {
				console.log("   Alpine: validation failed.");
				return;
			}
            
            this.loading = true;
            this.formErrors.global = "";

			console.log("2. Alpine: validation passed. Requesting Turnstile token...");

            // Logic Branch: Skip Turnstile if disabled
            if (!this.turnstileEnabled) {
                this.turnstileToken = "disabled-token";
				console.log("3. Alpine: Turnstile disabled. Manual trigger firing now.");
                // Ensure Alpine finishes internal updates before HTMX triggers
    			this.$nextTick(() => {
        			htmx.trigger('#signup-form-trigger', 'submit-signup');
    			});
    			return;
			}

            if (typeof window.turnstile === 'undefined') {
                this.formErrors.global = 'Security service unavailable.';
                this.loading = false;
                return;
            }

            // Programmatic execution (Safe pattern)
            window.turnstile.execute('#turnstile-signup', {
                sitekey: %s,
                action: 'signup',
                callback: (token) => {
					console.log("3. Turnstile: Token received! Handing off to HTMX...");
                    this.turnstileToken = token;
                    //htmx.trigger('#signup-form-trigger', 'submit-signup');
					this.$nextTick(() => {
                		const el = document.querySelector('#signup-form-trigger');
                		if (el) htmx.trigger(el, 'submit-signup');
						console.log("4. HTMX: Event 'submit-signup' fired on #signup-form-trigger.");
             		});
                },
                'expired-callback': () => {
                    this.loading = false;
                    this.turnstileToken = '';
                    this.formErrors.global = 'Security token expired. Please try again.';
                },
                'error-callback': () => {
					console.error("   Turnstile: Error occurred.");
                    this.loading = false;
                    this.formErrors.global = 'Verification failed. Please try again.';
                }
            });
        }
    }`, turnstileEnabled, string(safeSiteKey))
}
