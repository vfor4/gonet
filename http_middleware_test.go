package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func shouldIReturnHandler(next http.Handler) http.Handler {
	log.Println("check")
	return next
}

func TestReturnMiddleware(t *testing.T) {
	handler := shouldIReturnHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("bigo"))
	}))

	r := httptest.NewRequest(http.MethodGet, "http://test.com", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	b, _ := io.ReadAll(w.Body)
	log.Printf("%s", string(b))
}
