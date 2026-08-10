package main

import "fmt"

// ============================================================
// CONCEPT: Methods, Value vs Pointer Receivers
// ============================================================
//
// A method is a function with a "receiver" — attaches it to a type:
//
//   type Account struct {
//       Owner   string
//       Balance float64
//   }
//
//   // VALUE receiver — gets a COPY, cannot mutate the original
//   func (a Account) Summary() string {
//       return fmt.Sprintf("%s: $%.2f", a.Owner, a.Balance)
//   }
//
//   // POINTER receiver — gets the REAL thing via a pointer, CAN mutate
//   func (a *Account) Deposit(amount float64) {
//       a.Balance += amount   // Go auto-dereferences, no need for (*a).Balance
//   }
//
// RULE OF THUMB: if a method needs to mutate state, OR the struct is
// large (avoid copying), use a pointer receiver. Otherwise value receiver
// is fine. Mixing receiver types on the same struct is legal but considered
// bad style — pick one style per type, usually pointer if ANY method needs it.
//
// Calling: Go automatically takes the address for you when needed:
//   acc := Account{Owner: "Jagjit", Balance: 100}
//   acc.Deposit(50)      // Go rewrites this as (&acc).Deposit(50) automatically
//   fmt.Println(acc.Balance) // 150 - the original WAS mutated
//
// This auto-addressing only works on addressable values (local variables,
// not literals/map values) — another reason pointer receivers need care.

type Account struct {
	Owner   string
	Balance float64
}

func (a Account) Summary() string {
	return fmt.Sprintf("%s: $%.2f", a.Owner, a.Balance)
}

func (a *Account) Deposit(amount float64) {
	a.Balance += amount
}

func (a Account) BrokenDeposit(amount float64) {
	a.Balance += amount
}

func (a *Account) Withdraw(amount float64) error {
	if a.Balance < amount {
		return fmt.Errorf("insufficient funds")
	}

	a.Balance -= amount
	return nil
}

func main() {
	// TODO 1: define `type Account struct { Owner string; Balance float64 }`
	// above main (Go requires top-level types outside main)

	// TODO 2: write a VALUE-receiver method `Summary() string` that returns
	// a formatted string like "Jagjit: $100.00"

	// TODO 3: write a POINTER-receiver method `Deposit(amount float64)`
	// that adds to Balance

	// TODO 4: write a POINTER-receiver method `Withdraw(amount float64) error`
	// that returns an error (use fmt.Errorf("insufficient funds")) if
	// amount > Balance, otherwise subtracts and returns nil
	// (don't worry about deeply understanding `error` yet — that's Exercise 8;
	// just get the mechanics working here)

	// TODO 5: in main — create an Account, call Summary() and print it,
	// call Deposit(50), print Summary() again to CONFIRM the mutation stuck,
	// call Withdraw(1000) (should fail — print the error),
	// call Withdraw(30) (should succeed), print final Summary()
	account := Account{Owner: "Jagjit", Balance: 0}
	fmt.Println(account.Summary())
	account.Deposit(50)
	fmt.Println(account.Summary())
	fmt.Println(account.Withdraw(100))
	account.Withdraw(30)
	fmt.Println(account.Summary())
	// TODO 6: THE PROOF — write a throwaway value-receiver method
	// `BrokenDeposit(amount float64)` that does the same thing as Deposit
	// but with a VALUE receiver. Call it, then print Summary() — confirm
	// the balance did NOT change. This is the same lesson as Exercise 1's
	// struct-copy trap, now with methods instead of plain functions.
	account.BrokenDeposit(30)
	fmt.Println(account.Summary())
}

// EXPECTED OUTPUT (yours will vary based on your account name):
// Jagjit: $100.00
// Jagjit: $150.00
// insufficient funds
// Jagjit: $120.00
// Jagjit: $120.00   <- BrokenDeposit had no effect
