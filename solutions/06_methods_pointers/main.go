package main

import "fmt"

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

func (a *Account) Withdraw(amount float64) error {
	if amount > a.Balance {
		return fmt.Errorf("insufficient funds")
	}
	a.Balance -= amount
	return nil
}

func (a Account) BrokenDeposit(amount float64) {
	a.Balance += amount // mutates the copy only
}

func main() {
	acc := Account{Owner: "Jagjit", Balance: 100}
	fmt.Println(acc.Summary())

	acc.Deposit(50)
	fmt.Println(acc.Summary())

	if err := acc.Withdraw(1000); err != nil {
		fmt.Println(err)
	}

	if err := acc.Withdraw(30); err != nil {
		fmt.Println(err)
	}
	fmt.Println(acc.Summary())

	acc.BrokenDeposit(50)
	fmt.Println(acc.Summary()) // unchanged - proves value receiver copies
}
