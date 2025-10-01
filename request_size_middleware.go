package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/giygas/medicaments-api/config"
	"github.com/giygas/medicaments-api/logging"
)

// requestSizeMiddleware limits the size of incoming requests to prevent memory exhaustion attacks
func requestSizeMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check Content-Length header if present
			if contentLength := r.Header.Get("Content-Length"); contentLength != "" {
				if length, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
					if length > cfg.MaxRequestBody {
						logging.Warn("Request body too large",
							"content_length", length,
							"max_allowed", cfg.MaxRequestBody,
							"remote_addr", r.RemoteAddr,
							"user_agent", r.UserAgent())

						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusRequestEntityTooLarge)

						errorResponse := map[string]string{
							"error": fmt.Sprintf("Request body too large. Maximum allowed size is %d bytes", cfg.MaxRequestBody),
						}

						respondWithJSON(w, http.StatusRequestEntityTooLarge, errorResponse)
						return
					}
				}
			}

			// Check header size (rough estimate)
			headerSize := int64(0)
			for key, values := range r.Header {
				headerSize += int64(len(key))
				for _, value := range values {
					headerSize += int64(len(value))
				}
			}

			if headerSize > cfg.MaxHeaderSize {
				logging.Warn("Request headers too large",
					"header_size", headerSize,
					"max_allowed", cfg.MaxHeaderSize,
					"remote_addr", r.RemoteAddr,
					"user_agent", r.UserAgent())

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestHeaderFieldsTooLarge)

				errorResponse := map[string]string{
					"error": fmt.Sprintf("Request headers too large. Maximum allowed size is %d bytes", cfg.MaxHeaderSize),
				}

				respondWithJSON(w, http.StatusRequestHeaderFieldsTooLarge, errorResponse)
				return
			}

			// If all checks pass, proceed with the request
			next.ServeHTTP(w, r)
		})
	}
}
