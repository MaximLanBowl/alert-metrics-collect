package middleware

import (
	"bytes"
	"crypto/hmac"
	"io"
	"net/http"

	"github.com/MaximLanBowl/alert-metrics-collect/internal/crypto"
)

func HashSH256(key string) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			gotHash := r.Header.Get("HashSHA256")
			if gotHash == "" {
				http.Error(w, "Missing Hash SHA256 header", http.StatusBadRequest)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error reading request body", http.StatusBadRequest)
				return
			}
			r.Body.Close()

			r.Body = io.NopCloser(bytes.NewBuffer(body))

			expected := crypto.CalcHash(body, key)
			if !hmac.Equal([]byte(expected), []byte(gotHash)) {
				http.Error(w, "Hash not equal", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
