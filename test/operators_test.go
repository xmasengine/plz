package plz_test

import (
	"testing"
)

func TestIntegrationCompareEq(t *testing.T) {
	testArchs(t, `DECLARE va WORD
DECLARE vb WORD
DECLARE result WORD
LET va = 5
LET vb = 5
LET result = va == vb
OUTPUT 0 result
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationCompareGt(t *testing.T) {
	testArchs(t, `DECLARE va WORD
DECLARE vb WORD
DECLARE result WORD
LET va = 10
LET vb = 5
LET result = va > vb
OUTPUT 0 result
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationNot(t *testing.T) {
	testArchs(t, `DECLARE x WORD
DECLARE y WORD
LET x = 0
LET y = !x
OUTPUT 0 y
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationNotEqual(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 5 != 3
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationLt(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 3 < 5
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationGte(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 5 >= 5
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationLte(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 5 <= 5
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationMul(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 6 * 7
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationDiv(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 100 / 7
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 14 {
			t.Errorf("expected 14, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationMod(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 100 % 7
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 2 {
			t.Errorf("expected 2, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationShiftLeft(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 3 << 2
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 12 {
			t.Errorf("expected 12, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationShiftRight(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 48 >> 3
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 6 {
			t.Errorf("expected 6, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationBitAnd(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 7 & 3
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 3 {
			t.Errorf("expected 3, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationBitOr(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 4 | 3
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 7 {
			t.Errorf("expected 7, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationBitXor(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 7 ^ 3
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 4 {
			t.Errorf("expected 4, got %d", res.OutBytes[0][0])
		}
	})
}

// Keyword operator alternatives for comparison operators
func TestIntegrationKeywordEQ(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 5 EQ 5
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1 (5 EQ 5), got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordNE(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 3 NE 5
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1 (3 NE 5), got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordGT(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 10 GT 5
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1 (10 GT 5), got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordLT(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 3 LT 5
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1 (3 LT 5), got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordGE(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 5 GE 5
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1 (5 GE 5), got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordLE(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 5 LE 5
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1 (5 LE 5), got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordNOT(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = NOT 0
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1 (NOT 0), got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordPLUS(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 10 PLUS 3
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 13 {
			t.Errorf("expected 13, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordMINUS(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 10 MINUS 3
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 7 {
			t.Errorf("expected 7, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordTIMES(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 6 TIMES 7
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordDIV(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 100 DIV 7
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 14 {
			t.Errorf("expected 14, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordMOD(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 100 MOD 7
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 2 {
			t.Errorf("expected 2, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordBITAND(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 7 BITAND 3
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 3 {
			t.Errorf("expected 3, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordBITOR(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 4 BITOR 3
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 7 {
			t.Errorf("expected 7, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordXOR(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 7 XOR 3
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 4 {
			t.Errorf("expected 4, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordSHL(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 3 SHL 2
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 12 {
			t.Errorf("expected 12, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationKeywordSHR(t *testing.T) {
	testArchs(t, `DECLARE x WORD
LET x = 48 SHR 3
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 6 {
			t.Errorf("expected 6, got %d", res.OutBytes[0][0])
		}
	})
}
