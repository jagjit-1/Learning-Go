package main

import "fmt"

// ============================================================
// CONCEPT: embedding — composition that looks like inheritance
// ============================================================
//
// Declare a field with a type and NO name and you have embedded it:
//
//   type Animal struct { Name string }
//   func (a Animal) Describe() string { return "animal " + a.Name }
//
//   type Dog struct {
//       Animal          // embedded — no field name
//       Breed string
//   }
//
// Dog now gets Animal's fields and methods PROMOTED:
//
//   d := Dog{Animal: Animal{Name: "Rex"}, Breed: "Lab"}
//   d.Name            // promoted field, shorthand for d.Animal.Name
//   d.Describe()      // promoted method, shorthand for d.Animal.Describe()
//   d.Animal.Name     // the long way is always available
//
// This is the closest Go gets to inheritance, and the difference matters:
//
// >>> THERE IS NO VIRTUAL DISPATCH. <<<
//
// Define Describe on Dog and it SHADOWS the promoted one — d.Describe() now
// calls Dog's. But any method on Animal that calls Describe still calls
// ANIMAL's, because Animal has never heard of Dog:
//
//   func (a Animal) Intro() string { return "I am " + a.Describe() }
//
//   d.Intro()   // "I am animal Rex"  — NOT the Dog version
//
// In C# or Java, a virtual Describe would dispatch to the subclass. Go has no
// subclasses. The embedded Animal is just a field, and its methods receive an
// Animal receiver — the outer Dog is not reachable from in there. If you want
// the outer type's behaviour, pass it as an interface instead:
//
//   func Intro(d Describer) string { return "I am " + d.Describe() }
//
// That's the Go answer: embed to reuse implementation, use interfaces for
// polymorphism. They're separate tools.
//
// SHADOWING applies to fields too. An outer field at a shallower depth wins,
// and the inner one is still reachable through the type name.
//
// EMBEDDING AN INTERFACE in a struct is the decorator trick:
//
//   type LoggingWriter struct {
//       io.Writer                    // embedded INTERFACE
//       Records []string
//   }
//
// LoggingWriter satisfies io.Writer immediately, via promotion. Then you
// override only the method you care about and the rest keeps delegating —
// handy for big interfaces where you want to intercept one call. (The catch:
// if the embedded interface is nil, calling a promoted method panics.)
//
// EMBEDDING INTERFACES IN INTERFACES is how the stdlib composes them:
//
//   type ReadWriter interface {
//       Reader
//       Writer
//   }
//
// AND IN JSON, embedded struct fields are FLATTENED into the parent object —
// no nested object appears, which is usually what you want:
//
//   {"name":"Ada","salary":100}     not     {"Person":{"name":"Ada"},...}

// TODO 1: define `type Animal struct { Name string }` with three VALUE-receiver
// methods:
//   Describe() string  -> "animal <Name>"
//   GetName() string   -> the Name
//   Intro() string     -> "I am " + a.Describe()

// TODO 2: define `type Dog struct` embedding Animal plus a `Breed string`
// field, and give Dog its own `Describe() string` returning
// "dog <Name> (<Breed>)". Note you can reach Name directly — it's promoted.

// TODO 3: define three interfaces, using interface embedding for the third:
//   type Namer interface { GetName() string }
//   type Describer interface { Describe() string }
//   type Entity interface { Namer; Describer }
// Then write `func Explain(e Entity) string` returning
// "<GetName()>: <Describe()>". Called with a Dog this DOES get Dog's
// Describe — which is the whole point of using an interface rather than
// relying on embedding.

// TODO 4: define `type LoggingWriter struct` embedding io.Writer plus a
// `Records []string` field, with a pointer-receiver `Write` that appends the
// written text to Records and then delegates to the embedded Writer.

// TODO 5: field shadowing. Define
//   type Base struct { ID int }
//   type Wrapper struct { Base; ID string }
// and write `func IDs(w Wrapper) (string, int)` returning the outer ID and
// the inner one. Work out which name refers to which before you run it.

// TODO 6: define `type Person struct { Name string; Email string }` with json
// tags "name" and "email", and `type Employee struct` embedding Person with a
// `Salary int` tagged "salary". Write `func EmployeeJSON(e Employee) (string, error)`.
// Predict the shape of the JSON before you run it.

func main() {
	// TODO 7: build a Dog and print its promoted Name, then Describe().

	// TODO 8: print dog.Intro() and then dog.Animal.Describe(). Notice they
	// agree with each other and NOT with dog.Describe() — that's the missing
	// virtual dispatch.

	// TODO 9: print Explain(dog).

	// TODO 10: wrap a bytes.Buffer in a LoggingWriter, write two strings
	// through it, then print the buffer contents and len(Records).

	// TODO 11: print IDs of a Wrapper built with both IDs set.

	// TODO 12: print EmployeeJSON for an Employee.

	fmt.Print()
}

// EXPECTED OUTPUT:
// Rex dog Rex (Lab)
// I am animal Rex
// animal Rex
// Rex: dog Rex (Lab)
// hello world 2
// outer 7
// {"name":"Ada","email":"ada@example.com","salary":100}
