package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

type Methods map[string]http.Handler

func (m Methods) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	if handler, ok := m[method]; ok {
		if handler == nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		handler.ServeHTTP(w, r)
	} else {
		if method != http.MethodOptions {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		} else {
			w.Header().Add("OPTIONS", m.allowMethods())
		}
	}
}

func (m Methods) allowMethods() string {
	t := make([]string, len(m))
	for method := range m {
		t = append(t, method)
	}
	return strings.Join(t, " ")
}

func HomePageMethods(files string) Methods {
	return Methods{
		http.MethodGet: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if pusher, ok := w.(http.Pusher); ok {
				targets := []string{
					"/static/style.css",
					"/static/golang-gopher.svg",
				}
				for _, file := range targets {
					if err := pusher.Push(file, nil); err != nil {
						log.Printf("Failed to push file %v; error - %v", file, err)
					}
				}
				http.ServeFile(w, r, filepath.Join(files, "index.html"))
			} else {
				log.Println("Not a Pusher")
			}
		}),
	}
}
func SecondHomePageMethods(files string) Methods {
	return Methods{
		http.MethodGet: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(files, "index2.html"))
		}),
	}
}

func StrictPrefixMiddleware(prefix string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, ".") {
			http.Error(w, "File not found", http.StatusNotFound)
		}
		next.ServeHTTP(w, r)
	})
}

var (
	files = "./static/"
	cert  = "./trust/example.com+5.pem"
	pkey  = "./trust/example.com+5-key.pem"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", StrictPrefixMiddleware(".", http.FileServer(http.Dir("./static/")))))
	mux.Handle("/", HomePageMethods(files))
	mux.Handle("/2", SecondHomePageMethods(files))

	srv := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           mux,
		IdleTimeout:       time.Minute,
		ReadHeaderTimeout: 30 * time.Second,
	}

	done := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		if <-c == os.Interrupt {
			if err := srv.Shutdown(context.Background()); err != nil {
				log.Printf("%v", err)
			}
			close(done)
			return
		}
	}()

	log.Printf("Serving files from %q over TLS %s\n ", files, srv.Addr)
	err := srv.ListenAndServeTLS(cert, pkey)
	if err != http.ErrServerClosed {
		err = nil
	}
	<-done
}
