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

func TestCheckArrayBoundsValid(t *testing.T) {
	err := checkTest(t, "DECLARE arr ARRAY [5] BYTE\nLET x = arr[3]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckArrayBoundsValidWrite(t *testing.T) {
	err := checkTest(t, "DECLARE arr ARRAY [5] BYTE\nLET arr[0] = 42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckArrayBoundsOutOfRange(t *testing.T) {
	err := checkTest(t, "DECLARE arr ARRAY [5] BYTE\nLET x = arr[7]")
	if err == nil {
		t.Fatal("expected out-of-bounds error")
	}
	if !strings.Contains(err.Error(), "out of bounds") {
		t.Errorf("expected 'out of bounds' error, got: %v", err)
	}
}

func TestCheckArrayBoundsWriteOutOfRange(t *testing.T) {
	err := checkTest(t, "DECLARE arr ARRAY [5] BYTE\nLET arr[10] = 99")
	if err == nil {
		t.Fatal("expected out-of-bounds error")
	}
	if !strings.Contains(err.Error(), "out of bounds") {
		t.Errorf("expected 'out of bounds' error, got: %v", err)
	}
}

func TestCheckArrayBoundsNegativeIndex(t *testing.T) {
	err := checkTest(t, "DECLARE arr ARRAY [5] BYTE\nLET x = arr[-1]")
	if err == nil {
		t.Fatal("expected out-of-bounds error")
	}
	if !strings.Contains(err.Error(), "out of bounds") {
		t.Errorf("expected 'out of bounds' error, got: %v", err)
	}
}

func TestCheckArrayBoundsVariableIndex(t *testing.T) {
	err := checkTest(t, "DECLARE arr ARRAY [5] BYTE\nDECLARE i WORD\nLET arr[i] = 42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckArrayBoundsConstantExpr(t *testing.T) {
	err := checkTest(t, "DECLARE arr ARRAY [5] BYTE\nLET x = arr[2+2]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckArrayBoundsConstantExprOutOfRange(t *testing.T) {
	err := checkTest(t, "DECLARE arr ARRAY [5] BYTE\nLET x = arr[2+4]")
	if err == nil {
		t.Fatal("expected out-of-bounds error")
	}
	if !strings.Contains(err.Error(), "out of bounds") {
		t.Errorf("expected 'out of bounds' error, got: %v", err)
	}
}

func TestCheckPragmaBoundcheck(t *testing.T) {
	err := checkTest(t, "PRAGMA BOUNDCHECK")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckPragmaUnknown(t *testing.T) {
	err := checkTest(t, "PRAGMA FOO")
	if err == nil {
		t.Fatal("expected error for unknown pragma")
	}
	if !strings.Contains(err.Error(), "unrecognized pragma") {
		t.Errorf("expected 'unrecognized pragma' error, got: %v", err)
	}
}
