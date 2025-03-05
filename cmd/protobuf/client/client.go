package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vfor4/gonet/housework"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var addr, certfn string

func init() {
	flag.StringVar(&addr, "addr", "localhost:8443", "address")
	flag.StringVar(&certfn, "ca-cert", "cert.pem", "certificate")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			`Usage: %s [flags] [add chore, ...|complete #]
			    add         add comma-separated chores
			    complete    complete designated chore
			Flags:
			`, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	caCert, err := os.ReadFile(certfn)
	if err != nil {
		log.Fatal(err)
	}

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(caCert); !ok {
		log.Fatal("Failed to add cert from PEM")
	}

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(
		&tls.Config{
			RootCAs:          certPool,
			MinVersion:       tls.VersionTLS12,
			CurvePreferences: []tls.CurveID{tls.CurveP256},
		},
	)))
	if err != nil {
		log.Fatal(err)
	}
	rosie := housework.NewRobotMaidClient(conn)
	ctx := context.Background()
	switch strings.ToLower(flag.Arg(0)) {
	case "add":
		err = add(ctx, strings.Join(flag.Args()[1:], " "), rosie)
	case "complete":
		err = complete(ctx, flag.Arg(1), rosie)
	}

	if err != nil {
		log.Fatal(err)
	}

	err = list(ctx, rosie)
	if err != nil {
		log.Fatal(err)
	}
}

func list(ctx context.Context, client housework.RobotMaidClient) error {
	chores, err := client.List(ctx, new(housework.Empty))
	if err != nil {
		return err
	}
	if len(chores.Chores) == 0 {
		fmt.Println("There are no things to do, RobotMaid")
		return nil
	}
	for i, chore := range chores.Chores {
		c := " "
		if chore.Complete {
			c = "X"
		}
		fmt.Printf("%d\t[%s]\t%s\n", i+1, c, chore.Description)
	}
	return nil
}

func add(ctx context.Context, choresString string, client housework.RobotMaidClient) error {
	chores := new(housework.Chores)
	for _, chore := range strings.Split(choresString, ",") {
		chores.Chores = append(chores.Chores, &housework.Chore{Description: chore})
	}
	if len(chores.Chores) > 0 {
		_, err := client.Add(ctx, chores)
		if err != nil {
			return err
		}
	}
	return nil
}

func complete(ctx context.Context, chore string, client housework.RobotMaidClient) error {
	choreNum, err := strconv.Atoi(chore)
	if err != nil {
		return err
	}
	req := &housework.CompleteRequest{ChoreNumber: int32(choreNum)}
	_, err = client.Complete(ctx, req)
	if err != nil {
		return err
	}
	return nil
}
