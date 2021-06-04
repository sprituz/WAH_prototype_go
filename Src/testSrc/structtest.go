package main

import "fmt"

type MakeDate struct {
	Year  int
	Month int
	Day   int
}
type Car struct {
	m MakeDate
}

func main() {

	a := Car{MakeDate{Year: 2020, Month: 1, Day: 30}}
	a.m = MakeDate{2021, 2, 30}

	fmt.Println(a)
}
