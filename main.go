package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"
)

var (
	count    = flag.Int("count", 3, "how many times to ping?")
	interval = flag.Duration("interval", 3*time.Second, "The interval between pings")
	timeout  = flag.Duration("timeout", 2*time.Second, "The timeout")
)

func init() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] host:port\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}
}
func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Println("host:port is required")
		flag.Usage()
		os.Exit(1)
	}
	if *count <= 0 {
		fmt.Println("Ctrl + C to stop")
	}

	var try int
	for (*count <= 0) || (try <= *count) {
		try++
		fmt.Printf("try %v", try)
		start := time.Now()
		_, err := net.DialTimeout("tcp", flag.Arg(0), *timeout)
		end := time.Since(start)
		if err, ok := err.(net.Error); !ok || !err.Temporary() {
			fmt.Printf("error: %v, after: %v", err, end)
			os.Exit(0)
		}
		time.Sleep(*interval)
	}
}
