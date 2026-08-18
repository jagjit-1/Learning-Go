package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type Animal struct {
	Name string
}

func (a Animal) Describe() string { return "animal " + a.Name }
func (a Animal) GetName() string  { return a.Name }

// Intro calls Describe on an ANIMAL receiver. Even when the value is really a
// Dog, this reaches Animal.Describe — there is no dispatch back to the outer
// type, because Animal cannot see it.
func (a Animal) Intro() string { return "I am " + a.Describe() }

type Dog struct {
	Animal
	Breed string
}

// Shadows the promoted Animal.Describe for calls made ON a Dog.
func (d Dog) Describe() string {
	return "dog " + d.Name + " (" + d.Breed + ")"
}

type Namer interface {
	GetName() string
}

type Describer interface {
	Describe() string
}

type Entity interface {
	Namer
	Describer
}

// Through an interface, the dynamic type's method is used — so a Dog here
// really does get Dog.Describe.
func Explain(e Entity) string {
	return e.GetName() + ": " + e.Describe()
}

type LoggingWriter struct {
	io.Writer
	Records []string
}

func (l *LoggingWriter) Write(p []byte) (int, error) {
	l.Records = append(l.Records, string(p))
	return l.Writer.Write(p)
}

type Base struct {
	ID int
}

type Wrapper struct {
	Base
	ID string // shallower, so plain w.ID is this one
}

func IDs(w Wrapper) (string, int) {
	return w.ID, w.Base.ID
}

type Person struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Employee struct {
	Person
	Salary int `json:"salary"`
}

func EmployeeJSON(e Employee) (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func main() {
	dog := Dog{Animal: Animal{Name: "Rex"}, Breed: "Lab"}

	fmt.Println(dog.Name, dog.Describe())
	fmt.Println(dog.Intro())
	fmt.Println(dog.Animal.Describe())
	fmt.Println(Explain(dog))

	var buf bytes.Buffer
	lw := &LoggingWriter{Writer: &buf}
	fmt.Fprint(lw, "hello ")
	fmt.Fprint(lw, "world")
	fmt.Println(buf.String(), len(lw.Records))

	outer, inner := IDs(Wrapper{Base: Base{ID: 7}, ID: "outer"})
	fmt.Println(outer, inner)

	s, _ := EmployeeJSON(Employee{
		Person: Person{Name: "Ada", Email: "ada@example.com"},
		Salary: 100,
	})
	fmt.Println(s)
}
