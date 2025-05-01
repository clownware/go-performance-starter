package middleware

import (
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// RequestID generates a unique request ID for each HTTP request.
var RequestID = chimiddleware.RequestID

// RealIP sets the RemoteAddr to the client's real IP, using X-Forwarded-For.
var RealIP = chimiddleware.RealIP

// Logger logs the start and end of each request with the elapsed processing time.
var Logger = chimiddleware.Logger

// Recoverer recovers from panics and returns a 500 error.
var Recoverer = chimiddleware.Recoverer

// Timeout enforces a maximum duration for each request.
var Timeout = chimiddleware.Timeout
