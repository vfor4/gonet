package main

func main() {
	a := 5
	s := make([]int, a, 5)
	s[0] = 5
	update(s)
}
func update(s []int) {
	s[0] = 5
	s = append(s, 7)
}
