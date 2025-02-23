package main

import (
	"database/sql"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"path"
	"sort"
	"strings"
	"testing"
	"time"
)

func StrictMiddleware(prefix string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, v := range strings.Split(path.Clean(r.URL.Path), "/") {
			if strings.HasPrefix(v, prefix) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func TestFileServer(t *testing.T) {
	handler := http.StripPrefix("/static", StrictMiddleware(".", http.FileServer(http.Dir("./files/"))))
	tests := []struct {
		path string
		code int
	}{
		{"http://test.com/static/file1.txt", http.StatusOK},
		{"http://test.com/static/file2.txt", http.StatusOK},
		{"http://test.com/static/.confidential.fbi", http.StatusNotFound},
	}

	for _, v := range tests {
		r := httptest.NewRequest(http.MethodGet, v.path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Result().StatusCode != http.StatusOK {
			t.Logf("Not ok, status - %v; filename %v", w.Result().StatusCode, v.path)
		}
	}
}

func xTestMiddlewares(t *testing.T) {
	handler := http.TimeoutHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log("handling...")
		time.Sleep(time.Minute)
	}), 2*time.Second, "time out")

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://url.com", nil)
	handler.ServeHTTP(w, r)

	if http.StatusServiceUnavailable != w.Result().StatusCode {
		t.Fatalf("Expected 503 but got %v\n", w.Result().StatusCode)
	}
	b, err := io.ReadAll(w.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", b)
}

type Methods map[string]http.Handler

type Handlers struct {
	db     *sql.DB
	logger *log.Logger
}

func (h Handlers) handler1() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h.db
		if err != nil {
			return
		}
	}
}
func (h Handlers) handler2() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.logger.Print("See you")
	}
}

func (m Methods) allowedMethods() string {
	allows := make([]string, 0)
	for k := range m {
		allows = append(allows, k)
	}
	sort.Strings(allows)
	return strings.Join(allows, " ")
}

func (m Methods) ServerHttp() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler, ok := m[r.Method]; ok {
			if handler == nil {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			} else {
				handler.ServeHTTP(w, r)
			}
			return
		}
		if r.Method == http.MethodOptions {
			w.Header().Add("Allow", m.allowedMethods())
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func DefaultMethods() Methods {
	return Methods{
		http.MethodGet: http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("Hellow world"))
			}),
		http.MethodPost: http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				fmt.Fprintf(w, "Hello %s\n", html.EscapeString(string(b)))
			},
		),
	}

}

func xTestPitfallWriteBodyFirst(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Hello client"))
	})
	r := httptest.NewRequest(http.MethodGet, "http://test.com", nil)
	w := httptest.NewRecorder()

	handler(w, r)
	t.Log(w.Result().Status)
}

func xTestHttpServer(t *testing.T) {
	t.Log("morning")
	d := DefaultMethods()
	srv := &http.Server{
		Addr:              "127.0.0.1:",
		Handler:           http.TimeoutHandler(d.ServerHttp(), time.Minute, "time out"),
		IdleTimeout:       5 * time.Minute,
		ReadHeaderTimeout: time.Minute,
	}
	l, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		err := srv.Serve(l)
		if err != http.ErrServerClosed {
			t.Error(err)
		}
	}()
}
