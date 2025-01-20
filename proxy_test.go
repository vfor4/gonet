package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	BinaryType uint8 = iota + 1
	StringType

	MaxPayloadSize uint32 = 10 << 20
)

var ErrMaxPayLoadSize = errors.New("maximum payload size exceeded")

type Payload interface {
	fmt.Stringer
	io.ReaderFrom
	io.WriterTo
	Bytes() []byte
}

type String string

func (s String) String() string {
	return string(s)
}

func (s String) Bytes() []byte {
	return []byte(s)
}

func (s *String) ReadFrom(r io.Reader) (int64, error) {
	var l uint32
	err := binary.Read(r, binary.BigEndian, &l)
	if err != nil {
		return 1, err
	}
	if l > MaxPayloadSize {
		return 5, fmt.Errorf("%v exceeded MaxPayloadSize (10mb)", l)
	}
	payload := make([]byte, l)
	n, err := r.Read(payload)
	if err != nil {
		return int64(n), fmt.Errorf("Failed to read payload, bytes is read: %v", n)
	}
	*s = String(payload)
	n = +5
	return int64(n), nil
}

func (s String) WriteTo(w io.Writer) (int64, error) {
	err := binary.Write(w, binary.BigEndian, StringType)
	if err != nil {
		return 0, err
	}
	err = binary.Write(w, binary.BigEndian, uint32(len(s)))
	if err != nil {
		return 1, err
	}
	n, err := w.Write([]byte(s))
	n = +5
	if err != nil {
		return int64(n), err
	}
	return int64(0), nil
}

func decode(r io.Reader) (Payload, error) {
	var t uint8
	err := binary.Read(r, binary.BigEndian, &t)
	if err != nil {
		return nil, err
	}
	var payload Payload
	switch t {
	case StringType:
		payload = new(String)
	default:
		panic(fmt.Sprintf("Not supported type %s(%v)", string(t), t))
	}
	_, err = payload.ReadFrom(r)
	return payload, nil
}

func xTestProxy(t *testing.T) {
	var wg sync.WaitGroup
	proxyAddr := "127.0.0.1:38027"
	serverAddr := "127.0.0.1:33293"
	listenerReady := make(chan struct{})
	proxyReady := make(chan struct{})
	go StartListener(serverAddr, listenerReady, &wg)
	<-listenerReady
	go StartProxy(proxyAddr, serverAddr, &wg, proxyReady)
	<-proxyReady
	go StartClient(proxyAddr, &wg)
	wg.Wait()
}

func StartClient(addr string, wg *sync.WaitGroup) error {
	wg.Add(1)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(3*time.Second))
	d := net.Dialer{}
	go func() {
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			fmt.Printf("@4, %v\n", err)
			return
		}
		defer func() {
			cancel()
			wg.Done()
		}()
		p := String("ping")
		_, err = p.WriteTo(conn)
		if err != nil {
			fmt.Printf("@5, %v\n", err)
			return
		}
		go func() {
			for {
				p, err := decode(conn)
				if err != nil {
					fmt.Printf("@2, %v", err)
					return
				}
				switch p.String() {
				case "pong":
					fmt.Println("shut down client")
					return
					// po := String("end")
					// _, err := po.WriteTo(conn)
					// if err != nil {
					// 	fmt.Printf("@8, %v\n", err)
					// 	return
					// }
					// fmt.Println("shut down client")
					// return
				// case "ackend":
				// 	fmt.Println("shut down client")
				// 	return
				default:
					fmt.Println(p.String())
					panic("client shouldn't be here'")
				}
			}
		}()
	}()
	return nil
}

func StartListener(addr string, ready chan struct{}, wg *sync.WaitGroup) error {
	wg.Add(1)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	go func() {
		conn, err := l.Accept()
		if err != nil {
			fmt.Printf("@1: %v\n", err)
			return
		}
		defer func() {
			conn.Close()
			wg.Done()
		}()
		for {
			p, err := decode(conn)
			if err != nil {
				fmt.Printf("@2, %v", err)
				return
			}
			switch {
			case p.String() == "ping":
				fmt.Println("pong")
				po := String("pong")
				_, err := po.WriteTo(conn)
				if err != nil {
					fmt.Printf("@8, %v\n", err)
					return
				}
				fmt.Println("shut down listener")
				return
			case strings.Contains(p.String(), "proxy"):
				fmt.Println(p.String(), "hi proxy")
				return
			// case "end":
			// 	fmt.Println("ackend")
			// 	po := String("ackend")
			// 	_, err := po.WriteTo(conn)
			// 	if err != nil {
			// 		fmt.Printf("@9, %v\n", err)
			// 		return
			// 	}
			// 	fmt.Println("shut down listener")
			// 	return
			default:
				panic("listener shouldn't be here'")
			}

		}
	}()
	ready <- struct{}{}
	return nil
}

func xTestWriteAndRead(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}
	_, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	defer cancel()
	go func() {
		conn, err := l.Accept()
		defer func() {
			conn.Close()
			done <- struct{}{}
		}()
		if err != nil {
			fmt.Printf("Listener error %v\n", err)
		}
		p, err := decode(conn)
		if err != nil {
			fmt.Printf("Listener read error, %v", err)
			return
		}
		if p.String() == "ping" {
			po := String("pong")
			_, err := po.WriteTo(conn)
			if err != nil {
				fmt.Printf("Failed to send pong, %v\n", err)
				return
			}
		}
	}()

	dConn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		fmt.Printf("Dial error %v\n", err)
		return
	}
	p := String("ping")
	_, err = p.WriteTo(dConn)
	if err != nil {
		fmt.Println(err)
		return
	}
	po, err := decode(dConn)
	if err != nil {
		fmt.Printf("Dial error to read pong %v\n", err)
		return
	}
	fmt.Println(po.String())
	<-done
}

func StartProxy(addr, serverAddr string, wg *sync.WaitGroup, ready chan struct{}) {
	wg.Add(1)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("@1, %v\n", err)
	}
	go func() {
		from, err := l.Accept()
		if err != nil {
			fmt.Printf("@2 error %v\n", err)
			return
		}
		defer func() {
			from.Close()
			wg.Done()
		}()
		to, err := net.Dial("tcp", serverAddr)
		if err != nil {
			fmt.Printf("@3 error %v\n", err)
			return
		}
		defer func() {
			to.Close()
			wg.Done()
		}()
		r, w := io.Pipe()
		p := String("proxy here")
		_, err = p.WriteTo(w)
		if err != nil {
			fmt.Println(err)
			return
		}
		io.Copy(to, r)
	}()
	ready <- struct{}{}
}

type Monitor struct {
	*log.Logger
}

func (m *Monitor) Write(b []byte) (int, error) {
	return len(b), m.Output(5, string(b))
}

func TestLogger(t *testing.T) {
	m := &Monitor{Logger: log.New(os.Stdout, "monitor", 0)}
	m.Write([]byte("hello world"))
}
