package main

import "fmt"

func main() {
	timer := make(chan int, 1)
	timer <- 5
	go HandleChan(timer)
}

func HandleChan(timer chan int) {
	select {
	case n := <-timer:
		fmt.Println(n)
	default:
		fmt.Println("nothing fancy")
	}
}
