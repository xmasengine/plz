package plz

import (
	"strings"
	"testing"
)

func checkTest(t *testing.T, src string) error {
	t.Helper()
	tokens, err := Scan(strings.NewReader(src))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	prog := Program{}
	parser := NewParser(tokens)
	if err := prog.Parse(parser); err != nil {
		t.Fatalf("parse: %v", err)
	}
	c := NewChecker()
	return prog.Check(c)
}

func TestCheckUndeclaredVariable(t *testing.T) {
	err := checkTest(t, "LET x = y")
	if err == nil {
		t.Fatal("expected error for undeclared variable y")
	}
	if !strings.Contains(err.Error(), "undeclared") {
		t.Errorf("expected 'undeclared' error, got: %v", err)
	}
}

func TestCheckDeclaredVariable(t *testing.T) {
	err := checkTest(t, "DECLARE y WORD\nLET x = y")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
