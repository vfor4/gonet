package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

const DEFAULT_INTERVAL_PING_TIME = 5 * time.Second

func xTestReadHugeData(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal("failed to listen")
	}
	rb := make([]byte, 1<<24)
	_, err = rand.Read(rb)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		defer conn.Close()
		_, err = conn.Write(rb)
		if err != nil {
			fmt.Println(err)
			return
		}
	}()

	dConn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 1024)
	for {
		n, err := dConn.Read(b)
		if err != nil {
			if err != io.EOF {
				t.Fatal(err)
			}
			return
		}
		t.Logf("read %d bytes", n) // buf[:n] is the data read from conn
	}
}

func xTestAdvancePinger(t *testing.T) {
	done := make(chan struct{})
	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal("failed to listen")
	}

	begin := time.Now()
	go func() {
		defer close(done)
		conn, err := l.Accept()
		if err != nil {
			fmt.Println(err)
		}
		err = conn.SetDeadline(time.Now().Add(5 * time.Second))
		ctx, cancel := context.WithCancel(context.Background())
		resetTimer := make(chan time.Duration, 1)
		resetTimer <- time.Second
		defer func() {
			cancel()
			conn.Close()
		}()
		go Pinger(ctx, conn, resetTimer)
		for {
			b := make([]byte, 1024)
			n, err := conn.Read(b)
			if err != nil {
				fmt.Println("Listener: failed to read", err)
			}
			fmt.Printf("Listener received: %s\n", b[:n])
			err = conn.SetDeadline(time.Now().Add(5 * time.Second))
		}
	}()

	dConn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal("Failed to dial", err)
	}
	for i := 0; i < 4; i++ {
		b := make([]byte, 1024)
		n, err := dConn.Read(b)
		if err != nil {
			fmt.Println("Dial failed to read", err)
		}
		fmt.Printf("Dial received: %s\n", b[:n])
	}
	_, err = dConn.Write([]byte("pong"))
	if err != nil {
		t.Fatal("Dial failed to pong", err)
	}
	for i := 0; i < 4; i++ {
		b := make([]byte, 1024)
		n, err := dConn.Read(b)
		if err != nil {
			fmt.Println("Dial failed to read", err)
			return
		}
		fmt.Printf("Dial received: %s\n", b[:n])
	}
	<-done
	end := time.Since(begin).Truncate(time.Second)
	t.Logf("Done [%s]", end)
}

func Pinger(ctx context.Context, w io.Writer, resetTime <-chan time.Duration) {
	var interval time.Duration
	select {
	case <-ctx.Done():
	case interval = <-resetTime:
	default:
	}
	if interval <= 0 {
		interval = DEFAULT_INTERVAL_PING_TIME
	}
	fmt.Printf("Started pinger with interval %v\n", interval)
	timer := time.NewTimer(interval)
	defer func() {
		if !timer.Stop() {
			<-timer.C
		}
	}()
	startedTime := time.Now()
	for {
		select {
		case <-timer.C:
			fmt.Printf("tick (%v)\n", time.Since(startedTime).Round(1000*time.Millisecond))
			if _, err := w.Write([]byte("ping")); err != nil {
				fmt.Printf("Failed to write ping command%v", err)
				return
			}
		case <-ctx.Done():
			fmt.Println("done")
			return
		case newInterval := <-resetTime:
			if !timer.Stop() {
				<-timer.C
			}
			if newInterval > 0 {
				interval = newInterval
			}
		}
		timer.Reset(interval)
	}
}

func xTestBufferedChan(t *testing.T) {
	t.Log("hi")
	c := make(chan int)
	var wg sync.WaitGroup
	go func() {
		fmt.Println("v:")
		wg.Add(1)
		defer wg.Done()
		v := <-c
		fmt.Println("v:", v)
	}()
	wg.Wait()
}

func xTestNonBlockingPinger(t *testing.T) {
	t.Log("hi")
	ctx, cancel := context.WithCancel(context.Background())
	r, w := io.Pipe()
	resetTimer := make(chan time.Duration, 1)
	resetTimer <- 1 * time.Second

	go func() {
		Pinger(ctx, w, resetTimer)
	}()

	readPing := func(r io.Reader, d time.Duration) {
		if d >= 0 {
			resetTimer <- d
			fmt.Printf("reseted timer to %v\n", d)
		}

		b := make([]byte, 1024)
		n, err := r.Read(b)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("readPing: %q\n", b[:n])
	}

	for _, v := range []int{0, 2000} {
		readPing(r, time.Duration(v)*time.Millisecond)
	}
	cancel()
}

func xTestBlockingPinger(t *testing.T) {
	slidingTimes := [5]int{1, 1, 2, 3, 4}
	timer := time.NewTimer(time.Duration(slidingTimes[0]) * time.Second)
	for _, v := range slidingTimes {
		now := time.Now()
		timer.Reset(time.Duration(v) * time.Second)
		<-timer.C
		fmt.Printf("Tick (%v)\n", time.Since(now).Round(time.Second))
	}
	t.Log("x")
}

func xTestDeadLine(t *testing.T) {
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
