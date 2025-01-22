package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestUDPInterLoper(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sl, err := echoServerUDP(ctx, "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	cl, err := net.Dial("udp", sl.String())
	if err != nil {
		t.Fatal(err)
	}
	n, err := cl.Write([]byte("ping"))
	if err != nil {
		t.Fatal(err)
	}
	il, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	im := []byte("interrupt")
	n, err = il.WriteTo(im, cl.LocalAddr())
	if err != nil {
		t.Fatal(err)
	}
	crv := make([]byte, n)
	n, err = cl.Read(crv)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(crv[:n]))
	cl.SetReadDeadline(time.Now().Add(5 * time.Second))
	pb := make([]byte, 1024)
	n, err = cl.Read(pb)
	if err != nil {
		t.Fatal(err)
	}
}

func xTestUDPEchoServer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	originServer, err := echoServerUDP(ctx, "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	client, err := net.ListenPacket("udp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	b := []byte("ping")
	n, err := client.WriteTo(b, originServer)
	if err != nil {
		t.Fatal(err)
	}
	bf := make([]byte, n)
	n, serverAdd, err := client.ReadFrom(bf)
	if err != nil {
		t.Fatal(err)
	}
	if originServer.Network() != serverAdd.Network() {
		t.Fatal("expected server address is not the same")
	}
	fmt.Printf("message is %s\n", bf[:n])
}

func echoServerUDP(ctx context.Context, addr string) (net.Addr, error) {
	l, err := net.ListenPacket("udp", addr)
	if err != nil {
		return nil, err
	}
	go func() {
		go func() {
			<-ctx.Done()
			l.Close()
		}()
		b := make([]byte, 1024)
		for {
			n, sender, err := l.ReadFrom(b)
			if err != nil {
				return
			}
			n, err = l.WriteTo(b[:n], sender)
			if err != nil {
				return
			}
		}
	}()
	return l.LocalAddr(), nil
}
