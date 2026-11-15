package main

import (
	"strings"
	"testing"
)

// The statement footer is the attribution claim: present by default,
// gone when FABI_NO_CREDIT is set, and always a single line naming
// the tool and its repository.
func TestStatementCreditDefaultOn(t *testing.T) {
	t.Setenv("FABI_NO_CREDIT", "")
	c := statementCredit()
	if !strings.Contains(c, "fabi") ||
		!strings.Contains(c, "https://github.com/shizukutanaka/fabi") {
		t.Errorf("credit = %q, want one line naming fabi + repo URL", c)
	}
	if strings.Contains(c, "\n") {
		t.Errorf("credit must be one line, got %q", c)
	}
}

func TestStatementCreditOptOut(t *testing.T) {
	t.Setenv("FABI_NO_CREDIT", "1")
	if c := statementCredit(); c != "" {
		t.Errorf("credit = %q, want empty when FABI_NO_CREDIT is set", c)
	}
}
