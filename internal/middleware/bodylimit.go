package middleware

import (
	"io"
	"net/http"
)

type limitedReader struct {
	io.Reader
	limit int64
	read  int64
}

func (r *limitedReader) Read(p []byte) (n int, err error) {
	if r.read >= r.limit {
		return 0, io.EOF
	}
	n, err = r.Reader.Read(p)
	r.read += int64(n)
	if r.read > r.limit {
		return n, io.EOF
	}
	return n, err
}

func BodyLimit(maxSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				r.Body = http.MaxBytesReader(w, r.Body, maxSize)
			}
			next.ServeHTTP(w, r)
		})
	}
}
