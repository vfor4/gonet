package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vfor4/gonet/metrics"
)

var (
	metAddr = flag.String("metAddr", "127.0.0.1:8001", "metrics server address")
	webAddr = flag.String("servAddr", "127.0.0.1:8002", "web server address")
)

func HelloMetricHandler(w http.ResponseWriter, _ *http.Request) {
	metrics.Requests.Add(1)
	defer func(start time.Time) {
		metrics.RequestDurationHistogram.Observe(time.Since(start).Seconds())
		metrics.RequestDurationSumary.Observe(time.Since(start).Seconds())
	}(time.Now())
	_, err := w.Write([]byte("hello"))
	if err != nil {
		metrics.WriteErrors.Add(1)
	}
}

func NewHTTPServer(addr string, mux http.Handler, stateFunc func(net.Conn, http.ConnState)) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s := &http.Server{
		Addr:              addr,
		Handler:           mux,
		IdleTimeout:       time.Minute,
		ReadHeaderTimeout: 30 * time.Second,
		ConnState:         stateFunc,
	}
	go func() {
		log.Fatal(s.Serve(l))
	}()
	return nil
}

func onStateChange(_ net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		metrics.OpenConnections.Add(1)
	case http.StateClosed:
		metrics.OpenConnections.Add(-1)
	}
}

func main() {
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	if err := NewHTTPServer(*metAddr, mux, nil); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Metrics server is listening on %v ...\n", *metAddr)
	if err := NewHTTPServer(*webAddr, http.HandlerFunc(HelloMetricHandler), onStateChange); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("web server is listening on %v ...\n", *webAddr)

	clients := 500
	gets := 100
	wg := new(sync.WaitGroup)

	for _ = range clients {
		wg.Add(1)
		c := &http.Client{
			Transport: http.DefaultTransport.(*http.Transport).Clone(),
		}
		go func() {
			defer wg.Done()
			for _ = range gets {
				resp, err := c.Get(fmt.Sprintf("http://%s", *webAddr))
				if err != nil {
					log.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}()

	}
	wg.Add(1)
	wg.Wait()
}
