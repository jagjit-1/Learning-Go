package main

import (
	"fmt"
	"math"
)

// ============================================================
// CONCEPT: Interfaces — implicit satisfaction
// ============================================================
//
// An interface is a set of method signatures. A type satisfies it
// automatically just by having those methods — NO "implements" keyword,
// NO declaration anywhere linking type to interface.
//
//   type Shape interface {
//       Area() float64
//       Perimeter() float64
//   }
//
//   type Circle struct { Radius float64 }
//   func (c Circle) Area() float64      { return math.Pi * c.Radius * c.Radius }
//   func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.Radius }
//
// Circle now satisfies Shape. Nowhere did we write "Circle implements Shape".
//
// A COMPILE-TIME CHECK you can add (fails to build if the type stops
// satisfying the interface — useful as a safety net):
//   var _ Shape = Circle{}
//
// A function can accept the INTERFACE instead of a concrete type,
// meaning it works with anything satisfying it:
//   func Describe(s Shape) {
//       fmt.Printf("Area: %.2f, Perimeter: %.2f\n", s.Area(), s.Perimeter())
//   }
//
// TYPE SWITCH — when you have an interface value and need to know the
// concrete underlying type (used sparingly — needing this a lot is
// often a design smell, but it's essential to know):
//   func WhatShape(s Shape) {
//       switch v := s.(type) {
//       case Circle:
//           fmt.Println("It's a circle with radius", v.Radius)
//       case Rectangle:
//           fmt.Println("It's a rectangle")
//       default:
//           fmt.Println("Unknown shape")
//       }
//   }

type Shape interface {
	Area() float64
	Perimeter() float64
}

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

func (c Circle) Perimeter() float64 {
	return 2 * math.Pi * c.Radius
}

func Describe(s Shape) {
	fmt.Printf("Area: %.2f, Perimeter: %.2f\n", s.Area(), s.Perimeter())
}

func TotalArea(shapes []Shape) float64 {
	var total float64
	for _, shape := range shapes {
		total += shape.Area()
	}
	return total
}

type Rectangle struct {
	Width  float64
	Height float64
}

type Triangle struct {
	SideA  float64
	SideB  float64
	SideC  float64
	Base   float64
	Height float64
}

func (r Rectangle) Area() float64 {
	return r.Height * r.Width
}

func (r Rectangle) Perimeter() float64 {
	return 2 * (r.Height + r.Width)
}

func (t Triangle) Area() float64 {
	return 0.5 * t.Base * t.Height
}

func (t Triangle) Perimeter() float64 {
	return t.SideA + t.SideB + t.SideC
}

func WhatShape(s Shape) {
	switch typ := s.(type) {
	case Circle:
		fmt.Println("It's a circle with radius", typ.Radius)
	case Rectangle:
		fmt.Println("It's a rectangle")
	case Triangle:
		fmt.Println("It's a triangle")
	default:
		fmt.Println("Unknown shape")
	}
}

func main() {
	// TODO 1: define `type Shape interface { Area() float64; Perimeter() float64 }`

	// TODO 2: define `Circle struct { Radius float64 }` with Area/Perimeter methods
	// (use math.Pi)

	// TODO 3: define `Rectangle struct { Width, Height float64 }` with
	// Area/Perimeter methods

	// TODO 4: define `Triangle struct { Base, Height, SideA, SideB, SideC float64 }`
	// Area = 0.5 * Base * Height, Perimeter = SideA + SideB + SideC

	// TODO 5: add compile-time assertions for all three: `var _ Shape = Circle{}` etc.
	var _ Shape = Circle{}
	var _ Shape = Triangle{}
	var _ Shape = Rectangle{}
	// TODO 6: write `func Describe(s Shape)` as shown in the concept block, using it
	// to print area/perimeter for one of each shape
	Describe(Circle{Radius: 10})
	Describe(Triangle{SideA: 5, SideB: 5, SideC: 5, Base: 5, Height: 5})
	Describe(Rectangle{Width: 5, Height: 5})
	// TODO 7: create a `shapes := []Shape{...}` slice containing one of each
	// concrete type, range over it calling Describe on each
	shapes := []Shape{Circle{Radius: 10}, Triangle{SideA: 5, SideB: 5, SideC: 5, Base: 5, Height: 5}, Rectangle{Width: 5, Height: 5}}
	for _, shape := range shapes {
		Describe(shape)
	}
	// TODO 8: write `func TotalArea(shapes []Shape) float64` that sums Area()
	// across the whole slice — print the total
	total := TotalArea(shapes)
	fmt.Printf("Total area: %.2f\n", total)
	// TODO 9: write the WhatShape type-switch function from the concept block
	// (extend it for Rectangle and Triangle too) and call it on each shape
	// in your slice
	a := Circle{Radius: 5}
	b := Rectangle{Width: 5, Height: 5}
	c := Triangle{Base: 5, Height: 5}

	WhatShape(a)
	WhatShape(b)
	WhatShape(c)
}

// EXPECTED OUTPUT (values depend on your dimensions):
// Area: 78.54, Perimeter: 31.42
// Area: 78.54, Perimeter: 31.42
// Area: 24.00, Perimeter: 20.00
// Area: 6.00, Perimeter: 12.00
// Total area: 108.54
// It's a circle with radius 5
// It's a rectangle
// It's a triangle
