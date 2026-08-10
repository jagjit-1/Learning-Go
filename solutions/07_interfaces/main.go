package main

import (
	"fmt"
	"math"
)

type Shape interface {
	Area() float64
	Perimeter() float64
}

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64      { return math.Pi * c.Radius * c.Radius }
func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.Radius }

type Rectangle struct {
	Width, Height float64
}

func (r Rectangle) Area() float64      { return r.Width * r.Height }
func (r Rectangle) Perimeter() float64 { return 2 * (r.Width + r.Height) }

type Triangle struct {
	Base, Height, SideA, SideB, SideC float64
}

func (t Triangle) Area() float64      { return 0.5 * t.Base * t.Height }
func (t Triangle) Perimeter() float64 { return t.SideA + t.SideB + t.SideC }

var _ Shape = Circle{}
var _ Shape = Rectangle{}
var _ Shape = Triangle{}

func Describe(s Shape) {
	fmt.Printf("Area: %.2f, Perimeter: %.2f\n", s.Area(), s.Perimeter())
}

func TotalArea(shapes []Shape) float64 {
	total := 0.0
	for _, s := range shapes {
		total += s.Area()
	}
	return total
}

func WhatShape(s Shape) {
	switch v := s.(type) {
	case Circle:
		fmt.Println("It's a circle with radius", v.Radius)
	case Rectangle:
		fmt.Println("It's a rectangle")
	case Triangle:
		fmt.Println("It's a triangle")
	default:
		fmt.Println("Unknown shape")
	}
}

func main() {
	c := Circle{Radius: 5}
	r := Rectangle{Width: 6, Height: 4}
	t := Triangle{Base: 6, Height: 4, SideA: 5, SideB: 5, SideC: 6}

	Describe(c)
	Describe(r)
	Describe(t)

	shapes := []Shape{c, r, t}
	for _, s := range shapes {
		Describe(s)
	}

	fmt.Printf("Total area: %.2f\n", TotalArea(shapes))

	for _, s := range shapes {
		WhatShape(s)
	}
}
