package main

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

func init() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] host:port\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func TestPing(t *testing.T) {
}
