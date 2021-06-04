package main

import "fmt"

func main() {
	a := 4
	b := a

	if b%2 == 0 {
		fmt.Println("b 는 짝수입니다.")
	} else if a == 100 {
		fmt.Println("b 는 홀수입니다.")
	}
}
