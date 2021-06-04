package main

import "fmt"

type Vertex struct {
	X int
	Y int
}
type Test struct {
	a Vertex
	b int
}

func main() {
	a := Vertex{1, 0}
	a = Vertex{2, 2}
	var b = Test{a, 10}
	var c = 50
	fmt.Println(a.X + b.b + c)
}
