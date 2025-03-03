package main

import (
	"flag"

	"github.com/vfor4/gonet/housework"
	"google.golang.org/grpc"
)

var addr, certFn, keyFn string

func init() {
	flag.StringVar(&addr, "address", "localhost:443", "listening address")
	flag.StringVar(&certFn, "cert", "cert.pem", "certificate")
	flag.StringVar(&keyFn, "key", "key.pem", "private key")
}

func main() {
	flag.Parse()

	server := grpc.NewServer()
	rosie := new(housework.Rosie)
	server.RegisterService(housework.File_housework_proto.Services().ByName(""), nil)
}
