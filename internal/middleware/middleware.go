package middleware

import (
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// RequestID generates a unique request ID for each HTTP request.
var RequestID = chimiddleware.RequestID

// Logger logs the start and end of each request with the elapsed processing time.
var Logger = chimiddleware.Logger

// Recoverer recovers from panics and returns a 500 error.
var Recoverer = chimiddleware.Recoverer

// Timeout enforces a maximum duration for each request.
var Timeout = chimiddleware.Timeout

// Compress applies gzip/deflate response compression at the given level.
var Compress = chimiddleware.Compress
