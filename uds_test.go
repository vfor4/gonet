package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestUnitStreaming(t *testing.T) {
	dir, err := os.MkdirTemp("", "echo_unix")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	}()
	nw := "unix"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	socket := filepath.Join(dir, fmt.Sprintf("%d.sock", os.Getpid()))
	sAddr, err := streamingEchoServer(ctx, nw, socket)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	fmt.Println(socket)
	err = os.Chmod(socket, os.ModeSocket|0666)
	if err != nil {
		t.Fatal(err)
	}

	// client
	err = streamingClient(ctx, sAddr.Network(), sAddr.String())
	if err != nil {
		cancel()
		t.Fatal(err)
	}
}

func streamingClient(ctx context.Context, network, address string) error {
	dialer := net.Dialer{}
	client, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return err
	}

	for i := 0; i < 3; i++ {
		_, err := client.Write([]byte("ping"))
		if err != nil {
			return err
		}
	}
	buf := make([]byte, 1024)
	rn, err := client.Read(buf)
	if err != nil {
		return err
	}
	fmt.Println("heyooo", string(buf[:rn]))
	return nil
}

func streamingEchoServer(ctx context.Context, network, address string) (net.Addr, error) {
	unixPath := &net.UnixAddr{Name: address, Net: "unix"}
	listener, err := net.ListenUnix(network, unixPath)
	listener.SetUnlinkOnClose(false)
	if err != nil {
		return nil, err
	}

	go func() {
		defer func() {
			<-ctx.Done()
			listener.Close()

		}()
		b := make([]byte, 1024)
		for {
			fmt.Println("accepting...")
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				rn, err := conn.Read(b)
				fmt.Println("reading...")
				if err != nil {
					return
				}
				_, err = conn.Write(b[:rn])
				if err != nil {
					return
				}
			}()
		}
	}()
	return listener.Addr(), nil

}
