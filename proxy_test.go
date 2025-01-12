package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
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

func TestWriteAndRead(t *testing.T) {
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
		fmt.Println(p.String())
	}()

	dConn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		fmt.Printf("Dial error %v\n", err)
		return
	}
	p := String("hello")
	_, err = p.WriteTo(dConn)
	if err != nil {
		fmt.Println(err)
		return
	}
	<-done
}

func testProxy(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		conn, err := l.Accept()
		defer func() {
			conn.Close()
			done <- struct{}{}
		}()
		if err != nil {
			fmt.Printf("Listener error %v\n", err)
		}
		for {
			b := make([]byte, 1024)
			n, err := conn.Read(b)
			if err != nil {
				fmt.Printf("Listener read - error %v\n", err)
				return
			}
			if string(b[:n]) == "ping" {
				_, err = conn.Write([]byte("pong"))
				if err != nil {
					fmt.Printf("Listener write - error %v\n", err)
				}
			}
		}
	}()
}

func xProxy(ac net.Conn, bc net.Conn) error {
	_, err := io.Copy(ac, bc)
	if err != nil {
		return err
	}
	_, err = io.Copy(bc, ac)
	if err != nil {
		return err
	}
	return nil
}
