package plz_test

import (
	"testing"
)

func TestIntegrationPIR_OutputLiteral(t *testing.T) {
	io := compileAndRunPIR(t, `OUTPUT 0 42
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationPIR_LetNumber(t *testing.T) {
	io := compileAndRunPIR(t, `DECLARE x WORD
LET x = 99
OUTPUT 0 x
HALT`)
	if io.OutBytes[0][0] != 99 {
		t.Errorf("expected 99, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationPIR_LetAdd(t *testing.T) {
	io := compileAndRunPIR(t, `DECLARE x WORD
LET x = 10 + 20
OUTPUT 0 x
HALT`)
	if io.OutBytes[0][0] != 30 {
		t.Errorf("expected 30, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationPIR_LetSub(t *testing.T) {
	io := compileAndRunPIR(t, `DECLARE x WORD
LET x = 50 - 8
OUTPUT 0 x
HALT`)
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationPIR_LetMul(t *testing.T) {
	io := compileAndRunPIR(t, `DECLARE x WORD
LET x = 6 * 7
OUTPUT 0 x
HALT`)
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationPIR_LetDiv(t *testing.T) {
	io := compileAndRunPIR(t, `DECLARE x WORD
LET x = 42 / 6
OUTPUT 0 x
HALT`)
	if io.OutBytes[0][0] != 7 {
		t.Errorf("expected 7, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationPIR_LetMod(t *testing.T) {
	io := compileAndRunPIR(t, `DECLARE x WORD
LET x = 47 % 5
OUTPUT 0 x
HALT`)
	if io.OutBytes[0][0] != 2 {
		t.Errorf("expected 2, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationPIR_IfThen(t *testing.T) {
	io := compileAndRunPIR(t, `DECLARE x WORD
LET x = 0
IF 1 THEN LET x = 42
OUTPUT 0 x
HALT`)
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationPIR_IfThenElse(t *testing.T) {
	io := compileAndRunPIR(t, `DECLARE x WORD
IF 0 THEN LET x = 10 ELSE LET x = 20
OUTPUT 0 x
HALT`)
	if io.OutBytes[0][0] != 20 {
		t.Errorf("expected 20, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationPIR_WhileLoop(t *testing.T) {
	io := compileAndRunPIR(t, `DECLARE cnt WORD
LET cnt = 5
WHILE cnt > 0 DO
  OUTPUT 0 cnt
  LET cnt = cnt - 1
END
HALT`)
	if len(io.OutBytes[0]) != 5 {
		t.Fatalf("expected 5 outputs, got %d", len(io.OutBytes[0]))
	}
	for i, b := range io.OutBytes[0] {
		want := byte(5 - i)
		if b != want {
			t.Errorf("output[%d] = %d, want %d", i, b, want)
		}
	}
}

func TestIntegrationPIR_ByteVar(t *testing.T) {
	io := compileAndRunPIR(t, `DECLARE bv BYTE
LET bv = 200
OUTPUT 0 bv
HALT`)
	if io.OutBytes[0][0] != 200 {
		t.Errorf("expected 200, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationPIR_ProcedureCall(t *testing.T) {
	io := compileAndRunPIR(t, `DECLARE x WORD
PROCEDURE inc(n WORD) WORD
  RETURN n + 1
END
LET x = inc(41)
OUTPUT 0 x
HALT`)
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationPIR_ComparisonEQ(t *testing.T) {
	io := compileAndRunPIR(t, `DECLARE x WORD
LET x = 10
IF x == 10 THEN OUTPUT 0 1 ELSE OUTPUT 0 0
HALT`)
	if io.OutBytes[0][0] != 1 {
		t.Errorf("expected 1, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationPIR_ForLoop(t *testing.T) {
	io := compileAndRunPIR(t, `DECLARE cnt WORD
DECLARE sum WORD
LET sum = 0
FOR cnt = 1 TO 5 DO
  LET sum = sum + cnt
END
OUTPUT 0 sum
HALT`)
	// Sum 1..5 = 15
	if io.OutBytes[0][0] != 15 {
		t.Errorf("expected 15, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationOutputLiteral(t *testing.T) {
	io := compileAndRun(t, `OUTPUT 0 42
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationOutputChar(t *testing.T) {
	io := compileAndRun(t, `OUTPUT 7 'H'
OUTPUT 7 'i'
HALT`)
	if len(io.OutBytes[7]) < 2 {
		t.Fatal("expected 2 bytes of output")
	}
	if io.OutBytes[7][0] != 'H' {
		t.Errorf("expected 'H', got %d", io.OutBytes[7][0])
	}
	if io.OutBytes[7][1] != 'i' {
		t.Errorf("expected 'i', got %d", io.OutBytes[7][1])
	}
}

func TestIntegrationLetAdd(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 2 + 3
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 5 {
		t.Errorf("expected 5, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationLetSub(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 10 - 3
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 7 {
		t.Errorf("expected 7, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationIfThenElse(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 0
IF 1 THEN LET x = 10 ELSE LET x = 99
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 10 {
		t.Errorf("expected 10, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationIfThen(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 0
IF 0 THEN LET x = 10
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 0 {
		t.Errorf("expected 0, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationWhileLoop(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 0
WHILE x < 3 DO
  OUTPUT 0 x
  LET x = x + 1
END
HALT`)
	if len(io.OutBytes[0]) != 3 {
		t.Fatalf("expected 3 outputs, got %d: %v", len(io.OutBytes[0]), io.OutBytes[0])
	}
	for i := 0; i < 3; i++ {
		if io.OutBytes[0][i] != byte(i) {
			t.Errorf("output %d: expected %d, got %d", i, i, io.OutBytes[0][i])
		}
	}
}

func TestIntegrationForLoop(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
FOR x = 0 TO 2 DO
  OUTPUT 0 x
END
HALT`)
	if len(io.OutBytes[0]) != 3 {
		t.Fatalf("expected 3 outputs, got %d: %v", len(io.OutBytes[0]), io.OutBytes[0])
	}
	for i := 0; i < 3; i++ {
		if io.OutBytes[0][i] != byte(i) {
			t.Errorf("output %d: expected %d, got %d", i, i, io.OutBytes[0][i])
		}
	}
}

func TestIntegrationForLoopWithVariable(t *testing.T) {
	io := compileAndRun(t, `DECLARE cnt WORD
	DECLARE cnt2 WORD
FOR cnt = 'A' TO 'E' DO
	LET cnt2 = cnt + 1
	OUTPUT 0 cnt2 END
HALT`)
	if len(io.OutBytes[0]) != 5 {
		t.Fatalf("expected 5 outputs, got %d: %v", len(io.OutBytes[0]), io.OutBytes[0])
	}
	outs := string(io.OutBytes[0][0:5])
	if outs != "BCDEF" {
		t.Errorf("expected BCDEF, got %s", outs)
	}
}

func TestIntegrationExpressionChain(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 1 + 2 * 3
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	// 1 + 2 * 3 = 7 (MUL has higher precedence)
	if io.OutBytes[0][0] != 7 {
		t.Errorf("expected 7, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationOutputExpression(t *testing.T) {
	io := compileAndRun(t, `OUTPUT 0 3 + 4
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 7 {
		t.Errorf("expected 7, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationByteVar(t *testing.T) {
	io := compileAndRun(t, `DECLARE bv BYTE
LET bv = 99
OUTPUT 0 bv
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 99 {
		t.Errorf("expected 99, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationByteWidening(t *testing.T) {
	io := compileAndRun(t, `DECLARE bv BYTE
DECLARE w WORD
LET bv = 7
LET w = bv + 1
OUTPUT 0 w
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 8 {
		t.Errorf("expected 8, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationUnaryNeg(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 0 - 5
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	// -5 = 0xFFFB, low byte = 0xFB = 251
	if io.OutBytes[0][0] != 251 {
		t.Errorf("expected 251 (low byte of -5), got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationGoTo(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 10
GOTO skip
LET x = 99
skip: OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 10 {
		t.Errorf("expected 10, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationGroupDo(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 0
DO
  LET x = x + 1
END
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 1 {
		t.Errorf("expected 1, got %d", io.OutBytes[0][0])
	}
}
