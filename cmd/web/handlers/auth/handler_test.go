package auth

import (
	"comics-galore-web/internal/cloudflare"
	"comics-galore-web/internal/config"
	"github.com/stretchr/testify/mock"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestNewHandler(t *testing.T) {
	mockCfg := config.NewMockService()
	mockTurnstile := cloudflare.NewMockTurnstile()

	mockCfg.On("Get").Return(&config.Env{
		BetterAuth:       "http://localhost",
		BetterAuthSecret: "test-secret",
	})
	mockCfg.On("GetLogger").Return(slog.Default())

	h := NewHandler(mockCfg, mockTurnstile)

	assert.NotNil(t, h)
	assert.Implements(t, (*Handler)(nil), h)
}

func TestRegisterRoutes(t *testing.T) {
	app := fiber.New()

	// 1. Setup Mock Backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"ok"}`))
		if err != nil {
			return
		}
	}))
	defer backend.Close()

	// 2. Setup Mock Config
	mSvc := config.NewMockService()
	mSvc.On("Get").Return(&config.Env{
		AppEnv:           "development",
		BetterAuth:       backend.URL,
		BetterAuthSecret: "test-secret",
	})
	mSvc.On("GetLogger").Return(slog.Default())

	mTurnstile := cloudflare.NewMockTurnstile()
	mTurnstile.On("Verify",
		mock.Anything, // context.Context
		mock.Anything, // token string
		mock.Anything, // secretKey string
		mock.Anything, // remoteIP string
	).Return(&cloudflare.TurnstileResponse{
		Success:     true,
		ChallengeTS: "2026-03-20T00:00:00Z",
		Hostname:    "localhost",
		ErrorCodes:  nil,
		Action:      "login",
		CData:       "metadata",
	}, nil)

	// 3. Initialize Handler
	h := NewHandler(mSvc, mTurnstile)
	h.RegisterRoutes(app)

	// --- SUB-TESTS ---

	t.Run("Sign-In Flow", func(t *testing.T) {
		tests := []struct {
			name       string
			body       string
			expectedId int
		}{
			{
				name:       "Valid Sign-In",
				body:       `{"email":"test@example.com", "password":"password123"}`,
				expectedId: http.StatusOK,
			},
			{
				name:       "Invalid - Missing Email",
				body:       `{"password":"password123"}`,
				expectedId: http.StatusUnprocessableEntity, // Assuming your validator catches this
			},
			{
				name:       "Invalid - Malformed JSON",
				body:       `{"email": "test@example.com",`,
				expectedId: http.StatusUnprocessableEntity,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("POST", "/api/v1/auth/sign-in/email", strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
				req.Host = strings.TrimPrefix(backend.URL, "http://")

				resp, err := app.Test(req, fiber.TestConfig{Timeout: -1})
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedId, resp.StatusCode)
			})
		}
	})

	t.Run("Sign-Up Flow", func(t *testing.T) {
		tests := []struct {
			name       string
			body       string
			expectedId int
		}{
			{
				name:       "Valid Sign-Up",
				body:       `{"email":"test@example.com", "password":"password123", "name":"John Doe", "confirm_password":"password123"}`,
				expectedId: http.StatusOK,
			},
			{
				name:       "Invalid - Missing Email",
				body:       `{"password":"password123", "name":"John Doe", "confirmPassword":"password123"}`,
				expectedId: http.StatusUnprocessableEntity, // Assuming your validator catches this
			},
			{
				name:       "Invalid - Malformed JSON",
				body:       `{"email": "test@example.com",`,
				expectedId: http.StatusUnprocessableEntity,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("POST", "/api/v1/auth/sign-up/email", strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
				req.Host = strings.TrimPrefix(backend.URL, "http://")

				resp, err := app.Test(req, fiber.TestConfig{Timeout: -1})
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedId, resp.StatusCode)
			})
		}
	})

	t.Run("Reset-Password Flow", func(t *testing.T) {
		tests := []struct {
			name       string
			body       string
			expectedId int
		}{
			{
				name:       "Valid Request",
				body:       `{"email":"test@example.com"}`,
				expectedId: http.StatusOK,
			},
			{
				name:       "Invalid Email Format",
				body:       `{"email":"not-an-email"}`,
				expectedId: http.StatusUnprocessableEntity,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := httptest.NewRequest("POST", "/api/v1/auth/reset-password", strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
				req.Host = strings.TrimPrefix(backend.URL, "http://")

				resp, err := app.Test(req, fiber.TestConfig{Timeout: -1})
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedId, resp.StatusCode)
			})
		}
	})

	t.Run("Route GET /api/v1/auth/session exists in stack", func(t *testing.T) {
		// We use a HEAD or GET request to see if the route is registered.
		// Even if the proxy fails (because there's no real server),
		// getting a response other than 404 proves the route is registered.
		req := httptest.NewRequest("HEAD", "/api/v1/auth/session", nil)
		resp, _ := app.Test(req)

		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("Wildcard capture works for sub-paths", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/sign-in/email", nil)
		resp, _ := app.Test(req)

		assert.NotEqual(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("HTMX Auth Responses", func(t *testing.T) {
		tests := []struct {
			name           string
			method         string
			path           string
			body           string
			backendStatus  int
			expectedStatus int
			expectedHeader string // HX-Trigger value
			expectedBody   string
		}{
			{
				name:           "Sign-In Success HTMX",
				method:         "POST",
				path:           "/api/v1/auth/sign-in/email",
				body:           `{"email":"test@example.com", "password":"password123"}`,
				backendStatus:  http.StatusOK,
				expectedStatus: http.StatusOK,
				expectedHeader: "loginSuccess",
				expectedBody:   "Welcome!",
			},
			{
				name:           "Sign-In Failure HTMX",
				method:         "POST",
				path:           "/api/v1/auth/sign-in/email",
				body:           `{"email":"test-example.com", "password":"wrong-password"}`,
				backendStatus:  http.StatusOK,
				expectedStatus: http.StatusOK,
				expectedHeader: "authError",
				expectedBody:   "<div style=\"width: 35em;height: 50vh;\"><div class=\"flex items-center justify-between mb-4\"><img src=\"/assets/images/logo_compact.png\" alt=\"Comics Galore\" class=\"mx-auto\"> </div><form id=\"login-form\" x-data=\"{\n        email: &#39;&#39;,\n        password: &#39;&#39;,\n        formErrors: { email: &#39;&#39;, password: &#39;&#39;, global: &#39;&#39; },\n        turnstileToken: &#39;&#39;,\n        validate() {\n            this.formErrors = { email: &#39;&#39;, password: &#39;&#39;, global: &#39;&#39; };\n            let valid = true;\n            if (!this.email.trim()) {\n                this.formErrors.email = &#39;Email is required&#39;;\n                valid = false;\n            } else if (!this.email.includes(&#39;@&#39;) || !this.email.includes(&#39;.&#39;)) {\n                this.formErrors.email = &#39;Please enter a valid email&#39;;\n                valid = false;\n            }\n            if (!this.password) {\n                this.formErrors.password = &#39;Password is required&#39;;\n                valid = false;\n            } else if (this.password.length &lt; 8) {\n                this.formErrors.password = &#39;Password must be at least 8 characters&#39;;\n                valid = false;\n            }\n            return valid;\n        },\n        async submitForm() {\n            if (!this.validate()) return;\n\n            const container = document.getElementById(&#39;turnstile-container&#39;);\n            if (container) window.turnstile.remove(container);\n\n            window.turnstile.render(&#39;#turnstile-container&#39;, {\n                sitekey: &#34;&#34;,\n                callback: (token) =&gt; {\n                    this.turnstileToken = token;\n                    htmx.trigger(&#39;#login-form-submit&#39;, &#39;submit&#39;);\n                },\n                &#39;error-callback&#39;: () =&gt; {\n                    this.formErrors.global = &#39;Verification failed. Please try again.&#39;;\n                },\n                action: &#39;login&#39;,\n                cData: &#39;login-form&#39;\n            });\n        }\n    }\" x-on:submit.prevent=\"submitForm()\" class=\"space-y-6\"><div class=\"space-y-1.5\"><label for=\"email\" class=\"block text-sm font-medium text-gray-700\">Email <span class=\"text-red-500\">*</span></label> <input id=\"email\" name=\"email\" type=\"email\" required placeholder=\"\" value=\"\" class=\"block w-full rounded-lg border-gray-300 shadow-sm w-full mt-2 px-4 py-2 outline-none focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm disabled:bg-gray-100 disabled:cursor-not-allowed transition-colors duration-150 border-red-500 focus:border-red-500 focus:ring-red-500 \" x-model=\"email\" x-bind:class=\"{&#39;border-red-500 focus:border-red-500 focus:ring-red-500&#39;: formErrors.email}\" x-effect=\"$el.setCustomValidity(formErrors.email || &#39;&#39;)\" aria-invalid=\"true\" aria-describedby=\"email-error\"><!-- Alpine client-side error (reactive) --><p x-show=\"formErrors.email\" x-text=\"formErrors.email\" id=\"email-error\" class=\"mt-1.5 text-sm text-red-600\"></p><!-- Server-side error (static, rendered once) --><p id=\"email-error\" class=\"mt-1.5 text-sm text-red-600\">Invalid email provided</p></div><div class=\"space-y-1.5\"><label for=\"password\" class=\"block text-sm font-medium text-gray-700\">Password <span class=\"text-red-500\">*</span></label> <input id=\"password\" name=\"password\" type=\"password\" required placeholder=\"\" value=\"\" class=\"block w-full rounded-lg border-gray-300 shadow-sm w-full mt-2 px-4 py-2 outline-none focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm disabled:bg-gray-100 disabled:cursor-not-allowed transition-colors duration-150 \" x-model=\"password\" x-bind:class=\"{&#39;border-red-500 focus:border-red-500 focus:ring-red-500&#39;: formErrors.password}\" x-effect=\"$el.setCustomValidity(formErrors.password || &#39;&#39;)\" aria-invalid=\"false\" aria-describedby=\"password-error\"><!-- Alpine client-side error (reactive) --><p x-show=\"formErrors.password\" x-text=\"formErrors.password\" id=\"password-error\" class=\"mt-1.5 text-sm text-red-600\"></p><!-- Server-side error (static, rendered once) --></div><!-- Turnstile container — invisible mode --><div id=\"turnstile-container\" class=\"hidden\"></div><div id=\"login-form-submit\" hx-post=\"/api/v1/auth/sign-in/email\" hx-target=\"#login-form\" hx-swap=\"outerHTML\" hx-indicator=\"#login-loading\" hx-vals='js:{ \"cf-turnstile-response\": Alpine.$data(document.getElementById(\"login-form\")).turnstileToken }' hx-trigger=\"submit\"></div><div class=\"flex justify-end gap-4 items-center\"><div id=\"login-loading\" class=\"htmx-indicator inline-block animate-spin h-5 w-5 border-2 border-indigo-600 border-t-transparent rounded-full\"></div><button type=\"submit\" class=\"px-6 py-3 rounded-lg font-medium transition active:scale-98 bg-indigo-600 hover:bg-indigo-700 text-white w-full\">Sign In</button></div><!-- Alpine client-side global error --><div x-show=\"formErrors.global\" class=\"bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg\"><span x-text=\"formErrors.global\"></span></div><!-- Server-side global error --></form><div class=\"px-6 py-4 flex items-center justify-end gap-3\"><div class=\"text-center text-sm text-gray-600 my-1\"><button type=\"button\" class=\"text-indigo-600 hover:underline\" hx-get=\"/auth/modal/forgot\" hx-target=\"#modal-content\" hx-swap=\"innerHTML\" @click=\"$store.ui.modal.open = true\">Forgot your password ?</button></div></div></div>",
			},
			{
				name:           "Reset-Password Success HTMX",
				method:         "POST",
				path:           "/api/v1/auth/reset-password",
				body:           `{"email":"test@example.com"}`,
				backendStatus:  http.StatusOK,
				expectedStatus: http.StatusOK,
				expectedHeader: "resetSuccess",
				expectedBody:   "Email Sent",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Update backend behavior for this specific sub-test
				// We use a custom handler for the mock server to simulate specific backend errors
				backend.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.backendStatus)
					w.Write([]byte(`{"status":"mocked"}`))
				})

				req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("HX-Request", "true") // The magic header
				req.Host = strings.TrimPrefix(backend.URL, "http://")

				resp, err := app.Test(req, fiber.TestConfig{Timeout: -1})
				assert.NoError(t, err)

				// 1. Verify Status Code
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)

				// 2. Verify HTMX Trigger Header
				assert.Equal(t, tt.expectedHeader, resp.Header.Get("HX-Trigger"))

				// 3. Verify Body Content
				bodyBytes, _ := io.ReadAll(resp.Body)
				bodyStr := string(bodyBytes)

				assert.Contains(t, bodyStr, tt.expectedBody)

				// 4. Verify HTMX Retarget on error
				if tt.backendStatus >= 400 {
					assert.Equal(t, "#form-error", resp.Header.Get("HX-Retarget"))
				}
			})
		}
	})
}
