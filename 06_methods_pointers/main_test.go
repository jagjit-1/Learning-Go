package main

// ============================================================
// CHECKER for 06_methods_pointers — run with:  go test
// ============================================================
// The interesting checks here are about RECEIVER KIND: Deposit/Withdraw
// must be pointer receivers (they mutate), Summary/BrokenDeposit must be
// value receivers. That's verified both by behaviour and by reflection.

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		done <- buf.String()
	}()

	func() {
		defer func() {
			os.Stdout = old
			w.Close()
			if rec := recover(); rec != nil {
				t.Fatalf("main() panicked: %v", rec)
			}
		}()
		fn()
	}()

	return <-done
}

// --- TODO 1: the struct ---------------------------------------------
var _ = Account{Owner: "n", Balance: 1.5}

// --- TODO 2: Summary -------------------------------------------------

func TestSummaryFormat(t *testing.T) {
	cases := []struct {
		acc  Account
		want string
	}{
		{Account{Owner: "Jagjit", Balance: 100}, "Jagjit: $100.00"},
		{Account{Owner: "Ada", Balance: 0}, "Ada: $0.00"},
		{Account{Owner: "Grace", Balance: 1234.5}, "Grace: $1234.50"},
		{Account{Owner: "Alan", Balance: 99.999}, "Alan: $100.00"},
	}
	for _, c := range cases {
		if got := c.acc.Summary(); got != c.want {
			t.Errorf("TODO 2: Account{%q, %v}.Summary() = %q, want %q\n"+
				"  hint: fmt.Sprintf(\"%%s: $%%.2f\", ...)",
				c.acc.Owner, c.acc.Balance, got, c.want)
		}
	}
}

// --- TODO 3: Deposit mutates ----------------------------------------

func TestDepositMutates(t *testing.T) {
	acc := Account{Owner: "Jagjit", Balance: 100}
	acc.Deposit(50)
	if acc.Balance != 150 {
		t.Fatalf("TODO 3: after Deposit(50) on a balance of 100, Balance = %v, want 150\n"+
			"  hint: Deposit needs a POINTER receiver — func (a *Account) Deposit(...)",
			acc.Balance)
	}
	acc.Deposit(0.5)
	if acc.Balance != 150.5 {
		t.Errorf("TODO 3: after a further Deposit(0.5), Balance = %v, want 150.5", acc.Balance)
	}
}

// --- TODO 4: Withdraw ------------------------------------------------

func TestWithdrawSucceeds(t *testing.T) {
	acc := Account{Owner: "Jagjit", Balance: 150}
	if err := acc.Withdraw(30); err != nil {
		t.Fatalf("TODO 4: Withdraw(30) from a balance of 150 returned an error: %v", err)
	}
	if acc.Balance != 120 {
		t.Errorf("TODO 4: after Withdraw(30) from 150, Balance = %v, want 120", acc.Balance)
	}
}

func TestWithdrawExactBalanceIsAllowed(t *testing.T) {
	acc := Account{Owner: "Jagjit", Balance: 100}
	if err := acc.Withdraw(100); err != nil {
		t.Fatalf("TODO 4: withdrawing the exact balance should be allowed "+
			"(the check is amount > Balance, not >=), got error: %v", err)
	}
	if acc.Balance != 0 {
		t.Errorf("TODO 4: after withdrawing the full balance, Balance = %v, want 0", acc.Balance)
	}
}

func TestWithdrawTooMuchFails(t *testing.T) {
	acc := Account{Owner: "Jagjit", Balance: 150}
	err := acc.Withdraw(1000)
	if err == nil {
		t.Fatal("TODO 4: Withdraw(1000) from a balance of 150 should return a non-nil error")
	}
	if err.Error() == "" {
		t.Error("TODO 4: the error should carry a message, e.g. \"insufficient funds\"")
	}
	if acc.Balance != 150 {
		t.Errorf("TODO 4: a failed withdrawal must not change the balance — "+
			"got %v, want 150", acc.Balance)
	}
}

// --- TODO 6: BrokenDeposit does NOT mutate --------------------------

func TestBrokenDepositDoesNotMutate(t *testing.T) {
	acc := Account{Owner: "Jagjit", Balance: 120}
	acc.BrokenDeposit(50)
	if acc.Balance != 120 {
		t.Errorf("TODO 6: BrokenDeposit must use a VALUE receiver, so it mutates a "+
			"copy and leaves the original alone. Balance = %v, want it unchanged at 120.",
			acc.Balance)
	}
}

// --- Receiver kinds, checked directly -------------------------------
// A method with a value receiver is in the method set of BOTH Account and
// *Account. A method with a pointer receiver is only in *Account's.

func TestReceiverKinds(t *testing.T) {
	valueType := reflect.TypeOf(Account{})

	pointerReceiver := []string{"Deposit", "Withdraw"}
	for _, name := range pointerReceiver {
		if _, ok := valueType.MethodByName(name); ok {
			t.Errorf("%s should have a POINTER receiver — func (a *Account) %s(...).\n"+
				"  With a value receiver it can only ever mutate a copy.", name, name)
		}
	}

	valueReceiver := []string{"Summary", "BrokenDeposit"}
	for _, name := range valueReceiver {
		if _, ok := valueType.MethodByName(name); !ok {
			t.Errorf("%s should have a VALUE receiver — func (a Account) %s(...)", name, name)
		}
	}
}

// --- main()'s narration ----------------------------------------------

func TestOutput(t *testing.T) {
	out := captureStdout(t, main)
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — see TODO 5")
	}

	money := regexp.MustCompile(`(?m)^.+: \$\d+\.\d{2}$`)
	summaries := money.FindAllString(out, -1)
	if len(summaries) < 4 {
		t.Errorf("TODO 5/6: expected at least 4 Summary() lines (initial, after "+
			"Deposit, after the successful Withdraw, after BrokenDeposit) — found %d",
			len(summaries))
	}

	if len(summaries) >= 2 && summaries[len(summaries)-1] != summaries[len(summaries)-2] {
		t.Errorf("TODO 6: the last two summaries should be identical, because "+
			"BrokenDeposit changes nothing.\n  got %q then %q",
			summaries[len(summaries)-2], summaries[len(summaries)-1])
	}

	if !strings.Contains(strings.ToLower(out), "insufficient") {
		t.Error("TODO 5: expected the failed Withdraw(1000) to print its error " +
			"(\"insufficient funds\")")
	}
}
