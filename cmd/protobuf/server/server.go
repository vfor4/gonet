package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/vfor4/gonet/housework"
	"google.golang.org/grpc"
)

var addr, certFn, keyFn string

func init() {
	flag.StringVar(&addr, "address", "localhost:8443", "listening address")
	flag.StringVar(&certFn, "cert", "cert.pem", "certificate")
	flag.StringVar(&keyFn, "key", "key.pem", "private key")
}

type Rosie struct {
	mu     sync.Mutex
	chores []*housework.Chore
	housework.UnimplementedRobotMaidServer
}

func (r *Rosie) Add(_ context.Context, chores *housework.Chores) (*housework.Response, error) {

	r.mu.Lock()
	r.chores = append(r.chores, chores.Chores...)
	r.mu.Unlock()
	return &housework.Response{Message: "ok"}, nil
}

func (r *Rosie) Complete(_ context.Context, req *housework.CompleteRequest) (*housework.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.chores == nil && req.ChoreNumber < 1 || int(req.ChoreNumber) > len(r.chores) {
		return nil, fmt.Errorf("chore %d not found", req.ChoreNumber)
	}
	r.chores[req.ChoreNumber-1].Complete = true
	return &housework.Response{Message: "ok"}, nil
}

func (r *Rosie) List(_ context.Context, _ *housework.Empty) (*housework.Chores, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.chores == nil {
		r.chores = make([]*housework.Chore, 0)
	}
	return &housework.Chores{Chores: r.chores}, nil
}

func main() {
	flag.Parse()

	server := grpc.NewServer()
	rosie := new(Rosie)
	housework.RegisterRobotMaidServer(server, rosie)

	cert, err := tls.LoadX509KeyPair(certFn, keyFn)
	if err != nil {
		log.Fatal(err)
	}

	listen, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Listening for TLS connections on %s ...", addr)
	err = server.Serve(tls.NewListener(listen, &tls.Config{
		NextProtos:               []string{"h2"},
		Certificates:             []tls.Certificate{cert},
		CurvePreferences:         []tls.CurveID{tls.CurveP256},
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
	}))
	if err != nil {
		log.Fatal(err)
	}
}
