package handler

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/supabase-community/gotrue-go/types"

	"github.com/clownware/alpine-go-performance-starter/internal/auth"
	"github.com/clownware/alpine-go-performance-starter/internal/view"
	"github.com/clownware/alpine-go-performance-starter/internal/view/pages"
)

// AuthPage renders the combined login/signup page.
func AuthPage(w http.ResponseWriter, r *http.Request) {
	props := pages.AuthPageProps{
		BaseProps: view.NewBaseProps("Login or Sign Up"),
	}
	if err := view.Render(w, r, http.StatusOK, pages.AuthPage(props)); err != nil {
		slog.Error("Failed to render auth page", "error", err)
	}
}

// AuthLoginPost handles the login form submission.
func AuthLoginPost(authClient *auth.AuthClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			log.Printf("[ERROR] Parsing login form: %v", err)
			view.SetHXTrigger(w, `{"showToast":{"level":"error","message":"Failed to process form."}}`)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		if email == "" || password == "" {
			view.SetHXTrigger(w, `{"showToast":{"level":"error","message":"Email and password cannot be empty."}}`)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Use the SignInWithEmailPassword method from the supabase.Client
		// It returns a session object, which we aren't directly using here yet
		// because Supabase handles cookie setting server-side (usually).
		_, err := authClient.Client.SignInWithEmailPassword(email, password)

		if err != nil {
			log.Printf("[ERROR] Supabase login failed for %s: %v", email, err)
			// Provide a generic error for security
			view.SetHXTrigger(w, `{"showToast":{"level":"error","message":"Invalid login credentials."}}`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		log.Printf("[INFO] User login successful for email: %s", email)
		// Supabase client handles setting cookies.
		// Trigger a full page reload or redirect client-side via HTMX header.
		view.SetHXRedirect(w, "/profile") // Redirect to profile page after login
		w.WriteHeader(http.StatusOK)
	}
}

// AuthSignupPost handles the signup form submission.
func AuthSignupPost(authClient *auth.AuthClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			log.Printf("[ERROR] Parsing signup form: %v", err)
			view.SetHXTrigger(w, `{"showToast":{"level":"error","message":"Failed to process form."}}`)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		if email == "" || password == "" {
			view.SetHXTrigger(w, `{"showToast":{"level":"error","message":"Email and password cannot be empty."}}`)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Use the Signup method from the underlying gotrue client via authClient.Client.Auth
		_, err := authClient.Client.Auth.Signup(types.SignupRequest{
			Email:    email,
			Password: password,
			// Data: map[string]interface{}{"custom_field": "value"}, // Optional: Add custom user data
		})

		if err != nil {
			log.Printf("[ERROR] Supabase signup failed: %v", err)
			// Provide a more user-friendly error based on the type of Supabase error if possible
			view.SetHXTrigger(w, `{"showToast":{"level":"error","message":"Signup failed. User might already exist or password is too weak."}}`)
			w.WriteHeader(http.StatusConflict) // Or Bad Request depending on error
			// Optionally render a specific error message
			return
		}

		log.Printf("[INFO] User signup initiated for email: %s", email)
		view.SetHXTrigger(w, `{"showToast":{"level":"success","message":"Signup successful! Please check your email to confirm your account."}}`)
		w.WriteHeader(http.StatusOK)
		// Optionally clear the form or redirect, or just show the toast
		// w.Write([]byte("Signup successful! Check email.")) // Example direct response
	}
}

// AuthLogoutPost handles the logout request.
func AuthLogoutPost(authClient *auth.AuthClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Call Logout on the underlying gotrue client.
		// It implicitly uses the auth token from the request cookies/headers.
		err := authClient.Client.Auth.Logout()
		if err != nil {
			// Log the error, but proceed with logout flow client-side anyway
			log.Printf("[ERROR] Supabase signout failed: %v", err)
		}

		// Clear Supabase cookies explicitly (best practice)
		// Supabase might set these headers itself, but being explicit doesn't hurt.
		http.SetCookie(w, &http.Cookie{
			Name:     "sb-access-token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1, // Expire immediately
			HttpOnly: true,
			Secure:   r.TLS != nil, // Set Secure flag if using HTTPS
			SameSite: http.SameSiteLaxMode,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "sb-refresh-token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
		})

		log.Println("[INFO] User logout processed.")
		view.SetHXTrigger(w, `{"showToast":{"level":"success","message":"You have been logged out."}}`)
		view.SetHXRedirect(w, "/auth/page") // Redirect to login page
		w.WriteHeader(http.StatusOK)
	}
}
