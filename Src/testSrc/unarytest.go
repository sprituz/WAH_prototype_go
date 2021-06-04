package main

import "fmt"

func gcd(a int, b int) int {
	c := a
	d := b

	if c == 0 {
		return d
	}
	for d != 0 {
		if c > d {
			c = c - d
		} else {
			d = d - c
		}
	}
	return c
}
func swap(a int, b int) (int, int) {
	c := b
	d := a

	return c, d
}
func main() {
	a := 42
	b := 56
	c, d := swap(a, b)
	fmt.Println(c + d)
}
