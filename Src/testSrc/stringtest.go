package main

import "fmt"

var test = "!!!"

func main() {
	str := "hello"
	world := "world"
	golang := "golang" + "wow"

	helloworld := str + world + test + golang

	fmt.Println(helloworld)
}
