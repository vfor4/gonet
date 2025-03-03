package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/http2"
)

func TestTLS(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			u := "https://" + r.Host + r.RequestURI
			http.Redirect(w, r, u, http.StatusPermanentRedirect)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer s.Close()

	r, err := s.Client().Get(s.URL)
	if err != nil {
		t.Fatal(err)
	}

	if r.StatusCode != http.StatusOK {
		t.Fatal(r.Status)
	}

	tp := &http.Transport{
		TLSClientConfig: &tls.Config{
			CurvePreferences: []tls.CurveID{tls.CurveP256},
			MinVersion:       tls.VersionTLS12,
		},
	}
	_ = http2.ConfigureTransport(tp)

	client2 := &http.Client{Transport: tp}
	tp.TLSClientConfig.InsecureSkipVerify = true
	r2, err := client2.Get(s.URL)
	log.Println(r2.StatusCode)
}
