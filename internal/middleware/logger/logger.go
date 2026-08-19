package logger

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type responseData struct {
	statusCode int
	size       int
}

type responseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.responseData.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func WithLogging(fn http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			statusCode: http.StatusOK,
			size:       0,
		}
		lw := &responseWriter{
			w,
			responseData,
		}

		uri := r.RequestURI
		method := r.Method

		fn.ServeHTTP(lw, r)

		duration := time.Since(start)

		log.Info().
			Str("method", method).
			Str("uri", uri).
			Dur("duration", duration).
			Int("status", lw.responseData.statusCode).
			Int("size", lw.responseData.size).
			Msg("Request completed")
	}

	return http.HandlerFunc(logFn)
}
