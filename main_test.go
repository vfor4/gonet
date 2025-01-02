package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func TestDeadLine(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Log(err)
	}
	defer l.Close()
	done := make(chan struct{})
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				t.Log(err)
			}
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(5 * time.Second))
			log.Println("accepting...")
			go func(c net.Conn) {
				defer func() {
					done <- struct{}{}
				}()
				buf := make([]byte, 1024)
				log.Println("reading...")
				_, err = c.Read(buf)
				if !errors.Is(err, os.ErrDeadlineExceeded) {
					t.Log("not deadline exceeded ")
				}
				t.Log(err)
			}(conn)
		}
	}()
	<-done
}

func xTestCancelMultiDialiers(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		conn, err := l.Accept()
		log.Println("accepting...")
		if err != nil {
			log.Println(err)
			conn.Close()
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	res := make(chan int)
	d := func(ctx context.Context, addr string, wg *sync.WaitGroup, id int, res chan int) {
		var d net.Dialer
		dConn, err := d.Dial("tcp", addr)
		log.Println(id, "is dialing...")
		defer wg.Done()
		if err != nil {
			log.Println(err)
		}
		dConn.Close()

		select {
		case <-ctx.Done():
		case res <- id:
		}
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go d(ctx, l.Addr().String(), &wg, i, res)
	}
	response := <-res
	cancel()
	wg.Wait()
	close(res)

	if ctx.Err() != context.Canceled {
		log.Printf("not cancel error %v\n", err)
	}
	log.Println("response:", response)
}

func xTestNetTimeout(t *testing.T) {
	d := net.Dialer{}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	sync := make(chan struct{})
	defer cancel()

	go func(t *testing.T) {
		defer func() { sync <- struct{}{} }()
		c, err := d.DialContext(ctx, "tcp", "10.0.0.1:http")
		if err == nil {
			c.Close()
			t.Log("connection is not timeout")
		}
	}(t)
	<-sync
	t.Fatal(ctx.Err())
}

func xTestNet(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func(done chan struct{}) {
		defer func() { done <- struct{}{} }()

		for {
			conn, err := listener.Accept()
			if err != nil {
				t.Log(err)
				return
			}

			go func(c net.Conn, done chan struct{}) {
				defer func() {
					c.Close()
					done <- struct{}{}
				}()

				buf := make([]byte, 1024)
				for {
					n, err := c.Read(buf)
					if err != nil {
						if err != io.EOF {
							t.Error(err)
						}
						return
					}

					t.Logf("received: %q", buf[:n])
				}
			}(conn, done)
		}
	}(done)

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	conn.Write([]byte("kiss my ass"))

	conn.Close()
	<-done
	listener.Close()
	<-done

}
