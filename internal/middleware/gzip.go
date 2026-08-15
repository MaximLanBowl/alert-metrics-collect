package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipWriter struct {
	http.ResponseWriter
	gwr           *gzip.Writer
	validCompress bool
	headerWritten bool
}

func newGzipWriter(w http.ResponseWriter) *gzipWriter {
	return &gzipWriter{
		ResponseWriter: w,
		gwr:            gzip.NewWriter(w),
	}
}

func (g *gzipWriter) Header() http.Header {
	return g.ResponseWriter.Header()
}

func (g *gzipWriter) WriteHeader(statusCode int) {
	if g.headerWritten {
		return
	}
	g.headerWritten = true

	ct := g.ResponseWriter.Header().Get("Content-Type")
	compressible := strings.Contains(ct, "text/html") || strings.Contains(ct, "application/json")

	if statusCode < 300 && compressible {
		g.Header().Set("Content-Encoding", "gzip")
		g.validCompress = true
	}

	g.ResponseWriter.WriteHeader(statusCode)
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	if !g.headerWritten {
		g.WriteHeader(http.StatusOK)
	}

	if g.validCompress {
		return g.gwr.Write(b)
	}

	return g.ResponseWriter.Write(b)
}

func (g *gzipWriter) Close() error {
	if g.validCompress {
		return g.gwr.Close()
	}

	return nil
}

type gzipReader struct {
	io.ReadCloser
	gr *gzip.Reader
}

func newGzipReader(rc io.ReadCloser) (*gzipReader, error) {
	gr, err := gzip.NewReader(rc)
	if err != nil {
		return nil, err
	}

	return &gzipReader{
		ReadCloser: rc,
		gr:         gr,
	}, nil
}

func (r *gzipReader) Read(b []byte) (int, error) {
	return r.gr.Read(b)
}

func (r *gzipReader) Close() error {
	if err := r.gr.Close(); err != nil {
		return err
	}

	return r.ReadCloser.Close()
}

func Compressor(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			rc, err := newGzipReader(r.Body)
			if err != nil {
				http.Error(w, "failed to create gzip reader", http.StatusInternalServerError)
				return
			}
			defer rc.Close()
			r.Body = rc
		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			gw := newGzipWriter(w)
			defer gw.Close()

			w = gw
		}

		h.ServeHTTP(w, r)
	})
}
