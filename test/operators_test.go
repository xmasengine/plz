package plz_test

import (
	"testing"
)

func TestIntegrationCompareEq(t *testing.T) {
	io := compileAndRun(t, `DECLARE va WORD
DECLARE vb WORD
DECLARE result WORD
LET va = 5
LET vb = 5
LET result = va == vb
OUTPUT 0 result
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 1 {
		t.Errorf("expected 1, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationCompareGt(t *testing.T) {
	io := compileAndRun(t, `DECLARE va WORD
DECLARE vb WORD
DECLARE result WORD
LET va = 10
LET vb = 5
LET result = va > vb
OUTPUT 0 result
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 1 {
		t.Errorf("expected 1, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationNot(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
DECLARE y WORD
LET x = 0
LET y = !x
OUTPUT 0 y
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 1 {
		t.Errorf("expected 1, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationNotEqual(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 5 != 3
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 1 {
		t.Errorf("expected 1, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationLt(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 3 < 5
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 1 {
		t.Errorf("expected 1, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationGte(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 5 >= 5
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 1 {
		t.Errorf("expected 1, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationLte(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 5 <= 5
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 1 {
		t.Errorf("expected 1, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationMul(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 6 * 7
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationDiv(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 100 / 7
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 14 {
		t.Errorf("expected 14, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationMod(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 100 % 7
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 2 {
		t.Errorf("expected 2, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationShiftLeft(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 3 << 2
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 12 {
		t.Errorf("expected 12, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationShiftRight(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 48 >> 3
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 6 {
		t.Errorf("expected 6, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationBitAnd(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 7 & 3
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 3 {
		t.Errorf("expected 3, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationBitOr(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 4 | 3
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 7 {
		t.Errorf("expected 7, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationBitXor(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 7 ^ 3
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 4 {
		t.Errorf("expected 4, got %d", io.OutBytes[0][0])
	}
}
