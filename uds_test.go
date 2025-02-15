package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestUnixDatagram(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	dir, err := os.MkdirTemp("", "unix_data_echo")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.RemoveAll(dir)
	}()
	lsocket := filepath.Join(dir, fmt.Sprintf("l%d.sock", os.Getpid()))
	listenerAddr, err := unixDatagramEchoServer(ctx, "unixgram", lsocket)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chmod(lsocket, os.ModeSocket|0622)
	if err != nil {
		t.Fatal(err)
	}

	defer cancel()

	ssocket := filepath.Join(dir, fmt.Sprintf("s%d.sock", os.Getpid()))
	conn, err := net.ListenPacket("unixgram", ssocket)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chmod(ssocket, os.ModeSocket|0662)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		_, err = conn.WriteTo([]byte("ping"), listenerAddr)
		if err != nil {
			t.Log("Failed to write at i:", i+1)
		}
	}
	buf := make([]byte, 1024)
	for i := 0; i < 3; i++ {
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			t.Log("Failed to read")
		}
		t.Log(string(buf[:n]))
	}
}

func unixDatagramEchoServer(ctx context.Context, network, address string) (net.Addr, error) {
	conn, err := net.ListenPacket(network, address)
	if err != nil {
		return nil, err
	}
	go func() {
		defer func() {
			<-ctx.Done()
			conn.Close()

		}()
		b := make([]byte, 1024)
		for {
			if err != nil {
				return
			}
			go func() {
				rn, readAddr, err := conn.ReadFrom(b)
				fmt.Println("reading...")
				if err != nil {
					return
				}
				_, err = conn.WriteTo(b[:rn], readAddr)
				if err != nil {
					return
				}
			}()
		}
	}()
	return conn.LocalAddr(), nil
}

func xTestUnitStreaming(t *testing.T) {
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
	listener.SetUnlinkOnClose(true)
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
