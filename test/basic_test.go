package plz_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xmasengine/plz/pkg/plz"
)

func TestIntegrationPIR_OutputLiteral(t *testing.T) {
	testArchs(t, `OUTPUT 0 42
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_LetNumber(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 99
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 99 {
			t.Errorf("expected 99, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_LetAdd(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 10 + 20
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 30 {
			t.Errorf("expected 30, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_LetSub(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 50 - 8
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_LetMul(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 6 * 7
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_LetDiv(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 42 / 6
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 7 {
			t.Errorf("expected 7, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_LetMod(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 47 % 5
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 2 {
			t.Errorf("expected 2, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_IfThen(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 0
IF 1 THEN LET x = 42
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_IfThenElse(t *testing.T) {
	testArchs(t, `DECLARE x WORD
IF 0 THEN LET x = 10 ELSE LET x = 20
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 20 {
			t.Errorf("expected 20, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_WhileLoop(t *testing.T) {
	testArchs(t, `DECLARE cnt WORD
LET cnt = 5
WHILE cnt > 0 DO
  OUTPUT 0 cnt
  LET cnt = cnt - 1
END
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) != 5 {
			t.Fatalf("expected 5 outputs, got %d", len(res.OutBytes[0]))
		}
		for i, b := range res.OutBytes[0] {
			want := byte(5 - i)
			if b != want {
				t.Errorf("output[%d] = %d, want %d", i, b, want)
			}
		}
	})
}

func TestIntegrationPIR_ByteVar(t *testing.T) {
	testArchs(t, `DECLARE bv BYTE
LET bv = 200
OUTPUT 0 bv
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 200 {
			t.Errorf("expected 200, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_ProcedureCall(t *testing.T) {
	testArchs(t, `DECLARE x WORD
PROCEDURE inc(n WORD) WORD
  RETURN n + 1
END
LET x = inc(41)
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_ComparisonEQ(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 10
IF x == 10 THEN OUTPUT 0 1 ELSE OUTPUT 0 0
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_ForLoop(t *testing.T) {
	testArchs(t, `DECLARE cnt WORD
DECLARE sum WORD
LET sum = 0
FOR cnt = 1 TO 5 DO
  LET sum = sum + cnt
END
OUTPUT 0 sum
HALT`, func(t *testing.T, res *RunResult) {
		// Sum 1..5 = 15
		if res.OutBytes[0][0] != 15 {
			t.Errorf("expected 15, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_ForLoopWithVariable(t *testing.T) {
	testArchs(t, `DECLARE cnt WORD
DECLARE cnt2 WORD
FOR cnt = 'A' TO 'E' DO
  LET cnt2 = cnt + 1
  OUTPUT 0 cnt2
END
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) != 5 {
			t.Fatalf("expected 5 outputs, got %d", len(res.OutBytes[0]))
		}
		outs := string(res.OutBytes[0])
		if outs != "BCDEF" {
			t.Errorf("expected BCDEF, got %s", outs)
		}
	})
}

func TestIntegrationPIR_ExpressionChain(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 1 + 2 * 3
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		// 1 + 2 * 3 = 7 (MUL has higher precedence)
		if res.OutBytes[0][0] != 7 {
			t.Errorf("expected 7, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_OutputExpression(t *testing.T) {
	testArchs(t, `OUTPUT 0 3 + 4
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 7 {
			t.Errorf("expected 7, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_ByteWidening(t *testing.T) {
	testArchs(t, `DECLARE bv BYTE
DECLARE w WORD
LET bv = 7
LET w = bv + 1
OUTPUT 0 w
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 8 {
			t.Errorf("expected 8, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_UnaryNeg(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 0 - 5
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		// -5 = 0xFFFB, low byte = 0xFB = 251
		if res.OutBytes[0][0] != 251 {
			t.Errorf("expected 251 (low byte of -5), got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_GoTo(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 10
GOTO skip
LET x = 99
skip: OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 10 {
			t.Errorf("expected 10, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_GroupDo(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 0
DO
  LET x = x + 1
END
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_InputConstPort(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = INPUT(0)
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		// INPUT(0) returns whatever is at port 0 at runtime (unknown).
		// Just verify it completes without error; the result is undefined.
	})
}

func TestIntegrationPIR_InputNonConstPortErrors(t *testing.T) {
	// Non-constant INPUT port must produce a compile-time error on PIR backend.
	src := `DECLARE port WORD
LET port = 0
DECLARE x WORD
LET x = INPUT(port)
OUTPUT 0 x
HALT`
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "test.plz")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	tokens, err := plz.ScanFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	prog := plz.Program{}
	parser := plz.NewParser(tokens)
	if err := prog.Parse(parser); err != nil {
		t.Fatal(err)
	}
	_, err = prog.GenPIR()
	if err == nil {
		t.Fatal("expected error for non-constant INPUT port, got nil")
	}
}

func TestIntegrationPIR_WhileSingle(t *testing.T) {
	testArchs(t, `DECLARE cnt WORD
LET cnt = 3
WHILE cnt > 0 LET cnt = cnt - 1
OUTPUT 0 cnt
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 0 {
			t.Errorf("expected 0, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_ForSingle(t *testing.T) {
	testArchs(t, `DECLARE cnt WORD
DECLARE sum WORD
LET sum = 0
FOR cnt = 1 TO 3 LET sum = sum + cnt
OUTPUT 0 sum
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 6 { // 1+2+3 = 6
			t.Errorf("expected 6, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_ForLoopWithStep(t *testing.T) {
	testArchs(t, `DECLARE cnt WORD
DECLARE sum WORD
LET sum = 0
FOR cnt = 1 TO 5 BY 2 DO
  LET sum = sum + cnt
END
OUTPUT 0 sum
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 9 { // 1 + 3 + 5 = 9
			t.Errorf("expected 9, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_TrueFalse(t *testing.T) {
	testArchs(t, `DECLARE t WORD
DECLARE f WORD
LET t = TRUE
LET f = FALSE
OUTPUT 0 t
OUTPUT 0 f
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 2 {
			t.Fatal("expected 2 outputs")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1 (TRUE), got %d", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 0 {
			t.Errorf("expected 0 (FALSE), got %d", res.OutBytes[0][1])
		}
	})
}

func TestIntegrationPIR_AtBankCompiles(t *testing.T) {
	testArchs(t, `AT BANK 1
DATA _bd 99
OUTPUT 0 42
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_WordCast(t *testing.T) {
	testArchs(t, `DECLARE w WORD
LET w = WORD(0xAB)
OUTPUT WORD 0 w
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 2 {
			t.Fatalf("expected 2 bytes (word output), got %v", res.OutBytes[0])
		}
		if res.OutBytes[0][0] != 0xAB {
			t.Errorf("expected low byte 0xAB, got 0x%02x", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 0 {
			t.Errorf("expected high byte 0x00, got 0x%02x", res.OutBytes[0][1])
		}
	})
}

func TestIntegrationPIR_ConstNoEq(t *testing.T) {
	testArchs(t, `CONSTANT x 42
DECLARE v WORD
LET v = x
OUTPUT 0 v
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_MultiStatementLine(t *testing.T) {
	testArchs(t, `DECLARE x WORD; DECLARE y WORD
LET x = 10; LET y = 20; OUTPUT 0 x; OUTPUT 0 y
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 2 {
			t.Fatal("expected 2 outputs")
		}
		if res.OutBytes[0][0] != 10 {
			t.Errorf("expected 10, got %d", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 20 {
			t.Errorf("expected 20, got %d", res.OutBytes[0][1])
		}
	})
}

func TestIntegrationPIR_ConstExpr(t *testing.T) {
	testArchs(t, `CONSTANT x = 10 + 20
DECLARE v WORD
LET v = x
OUTPUT 0 v
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 30 {
			t.Errorf("expected 30, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_PragmaNoBoundCheck(t *testing.T) {
	// PRAGMA NOBOUNDCHECK only affects the code generator, not the checker.
	// The checker always validates array bounds. Use a valid index but
	// verify the pragma is at least accepted without error.
	testArchs(t, `DECLARE arr ARRAY[4] BYTE
PRAGMA NOBOUNDCHECK
LET arr[0] = 42
OUTPUT 0 arr[0]
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_TileContent(t *testing.T) {
	// TILE data stores a single 8-byte tile. Only index 0 is valid
	// for the checker (len(Tiles)=1), which is the first byte of the tile.
	testArchs(t, "DATA mytile TILE `..XX..XX`\nOUTPUT 0 mytile[0]\nHALT", func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 0x33 {
			t.Errorf("expected byte 0 = 0x33 (..XX..XX = 00110011), got 0x%02x", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_BankStmtCompiles(t *testing.T) {
	testArchs(t, `BANK 1
OUTPUT 0 42
HALT`, func(t *testing.T, res *RunResult) {
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	}, "z80", "nes")
}
