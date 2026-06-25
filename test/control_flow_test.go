package plz_test

import "testing"

func TestIntegrationBreak(t *testing.T) {
	testArchs(t, `DECLARE idx WORD
DECLARE result WORD
LET result = 0
FOR idx = 0 TO 10 DO
  IF idx == 5 THEN BREAK
  LET result = result + 1
END
OUTPUT 0 result
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 5 {
			t.Errorf("expected 5, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationContinue(t *testing.T) {
	testArchs(t, `DECLARE idx WORD
DECLARE result WORD
LET result = 0
FOR idx = 0 TO 10 DO
  IF idx == 5 THEN CONTINUE
  LET result = result + 1
END
OUTPUT 0 result
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 10 {
			t.Errorf("expected 10, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationBreakInTask(t *testing.T) {
	testArchs(t, `TASK demo PRIORITY 0
  DECLARE idx WORD
  DECLARE result WORD
  LET result = 0
  FOR idx = 0 TO 10 DO
    IF idx == 5 THEN BREAK
    LET result = result + 1
  END
  OUTPUT 0 result
END
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 5 {
			t.Errorf("expected 5, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationContinueInTask(t *testing.T) {
	testArchs(t, `TASK demo PRIORITY 0
  DECLARE idx WORD
  DECLARE result WORD
  LET result = 0
  FOR idx = 0 TO 5 DO
    IF idx == 3 THEN CONTINUE
    LET result = result + 1
  END
  OUTPUT 0 result
END
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 5 {
			t.Errorf("expected 5 (0,1,2,4,5 skipping 3), got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationCase(t *testing.T) {
	testArchs(t, `DECLARE cx BYTE
DECLARE result BYTE
FOR cx = 0 TO 3 DO
  CASE cx OF 0 LET result = 10
         OF 1 LET result = 20
         OF 2 LET result = 30
         OF DEFAULT LET result = 99
  END
  OUTPUT 0 result
END
HALT`, func(t *testing.T, res *RunResult) {
		got := res.OutBytes[0]
		if len(got) != 4 {
			t.Fatalf("expected 4 outputs, got %d: %v", len(got), got)
		}
		expect := []byte{10, 20, 30, 99}
		for i, v := range expect {
			if got[i] != v {
				t.Errorf("cx=%d: expected %d, got %d", i, v, got[i])
			}
		}
	})
}

func TestIntegrationCaseConst(t *testing.T) {
	testArchs(t, `CONSTANT RED = 0
CONSTANT GREEN = 1
CONSTANT BLUE = 2
DECLARE cx BYTE
DECLARE result BYTE
FOR cx = 0 TO 2 DO
  CASE cx OF RED   LET result = 10
         OF GREEN LET result = 20
         OF BLUE  LET result = 30
  END
  OUTPUT 0 result
END
HALT`, func(t *testing.T, res *RunResult) {
		got := res.OutBytes[0]
		if len(got) != 3 {
			t.Fatalf("expected 3 outputs, got %d: %v", len(got), got)
		}
		expect := []byte{10, 20, 30}
		for i, v := range expect {
			if got[i] != v {
				t.Errorf("cx=%d: expected %d, got %d", i, v, got[i])
			}
		}
	})
}

func TestIntegrationLogicalAnd(t *testing.T) {
	testArchs(t, `DECLARE va BYTE
DECLARE vb BYTE
LET va = 3
LET vb = 5
OUTPUT 0 va && vb
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationLogicalOr(t *testing.T) {
	testArchs(t, `DECLARE va BYTE
DECLARE vb BYTE
LET va = 0
LET vb = 7
OUTPUT 0 va || vb
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationAndKeyword(t *testing.T) {
	testArchs(t, `DECLARE va BYTE
DECLARE vb BYTE
LET va = 3
LET vb = 5
OUTPUT 0 va AND vb
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationOrKeyword(t *testing.T) {
	testArchs(t, `DECLARE va BYTE
DECLARE vb BYTE
LET va = 0
LET vb = 7
OUTPUT 0 va OR vb
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationMultiDimArray(t *testing.T) {
	testArchs(t, `DECLARE arr ARRAY [2] ARRAY [3] BYTE
LET arr[0][0] = 11
LET arr[0][1] = 12
LET arr[0][2] = 13
LET arr[1][0] = 21
LET arr[1][1] = 22
LET arr[1][2] = 23
OUTPUT 0 arr[1][1]
OUTPUT 0 arr[0][2]
OUTPUT 0 arr[0][0]
OUTPUT 0 arr[1][0]
HALT`, func(t *testing.T, res *RunResult) {
		got := res.OutBytes[0]
		if len(got) < 4 {
			t.Fatalf("expected 4 outputs, got %d: %v", len(got), got)
		}
		expect := []byte{22, 13, 11, 21}
		for i, v := range expect {
			if got[i] != v {
				t.Errorf("output %d: expected %d, got %d", i, v, got[i])
			}
		}
	})
}

func TestIntegrationSaveLoad(t *testing.T) {
	testArchs(t, `DECLARE src ARRAY [4] BYTE
LET src[0] = 10
LET src[1] = 20
LET src[2] = 30
LET src[3] = 40
DECLARE dst ARRAY [4] BYTE
SAVE AT 0x4000 src
LET src[0] = 0
LET src[1] = 0
LET src[2] = 0
LET src[3] = 0
LOAD AT 0x4000 dst
OUTPUT 0 dst[0]
OUTPUT 0 dst[1]
OUTPUT 0 dst[2]
OUTPUT 0 dst[3]
HALT`, func(t *testing.T, res *RunResult) {
		got := res.OutBytes[0]
		if len(got) < 4 {
			t.Fatalf("expected 4 outputs, got %d: %v", len(got), got)
		}
		expect := []byte{10, 20, 30, 40}
		for i, v := range expect {
			if got[i] != v {
				t.Errorf("output %d: expected %d, got %d", i, v, got[i])
			}
		}
	}, "z80", "6502")
}

func TestIntegrationTask(t *testing.T) {
	testArchs(t, `TASK demo PRIORITY 0
  OUTPUT 0 42
END`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationMultiTask(t *testing.T) {
	testArchs(t, `TASK ta PRIORITY 0
  OUTPUT 0 1
END
TASK tb PRIORITY 1
  OUTPUT 0 2
END`, func(t *testing.T, res *RunResult) {
		got := res.OutBytes[0]
		if len(got) < 2 {
			t.Fatalf("expected 2 outputs, got %d: %v", len(got), got)
		}
		if got[0] != 1 {
			t.Errorf("expected task ta output 1, got %d", got[0])
		}
		if got[1] != 2 {
			t.Errorf("expected task tb output 2, got %d", got[1])
		}
	})
}

func TestIntegrationHaltInTask(t *testing.T) {
	testArchs(t, `TASK ta PRIORITY 0
  OUTPUT 0 1
END
TASK tb PRIORITY 1
  OUTPUT 0 2
  HALT
  OUTPUT 0 3
END`, func(t *testing.T, res *RunResult) {
		got := res.OutBytes[0]
		if len(got) < 2 {
			t.Fatalf("expected 2 outputs, got %d: %v", len(got), got)
		}
		if got[0] != 1 {
			t.Errorf("expected 1, got %d", got[0])
		}
		if got[1] != 2 {
			t.Errorf("expected 2, got %d", got[1])
		}
		// HALT in tb should prevent output 3 — verify
		if len(got) > 2 {
			t.Errorf("expected only 2 outputs, got %d: %v", len(got), got)
		}
	})
}

func TestIntegrationSuspendResume(t *testing.T) {
	testArchs(t, `TASK worker PRIORITY 1
  OUTPUT 0 2
  OUTPUT 0 3
END
TASK main PRIORITY 0
  SUSPEND worker
  OUTPUT 0 1
  RESUME worker
END`, func(t *testing.T, res *RunResult) {
		got := res.OutBytes[0]
		if len(got) < 3 {
			t.Fatalf("expected 3 outputs, got %d: %v", len(got), got)
		}
		if got[0] != 1 {
			t.Errorf("expected output 1, got %d", got[0])
		}
		if got[1] != 2 {
			t.Errorf("expected output 2, got %d", got[1])
		}
		if got[2] != 3 {
			t.Errorf("expected output 3, got %d", got[2])
		}
	}, "z80", "6502")
}

func TestIntegrationSaveLoadPIR(t *testing.T) {
	testArchs(t, `DECLARE src ARRAY [4] BYTE
LET src[0] = 10
LET src[1] = 20
LET src[2] = 30
LET src[3] = 40
SAVE AT 0xB000 src
LET src[0] = 99
LET src[1] = 99
LET src[2] = 99
LET src[3] = 99
LOAD AT 0xB000 src
OUTPUT 0 src[0]
OUTPUT 0 src[1]
OUTPUT 0 src[2]
OUTPUT 0 src[3]
HALT`, func(t *testing.T, res *RunResult) {
		got := res.OutBytes[0]
		if len(got) < 4 {
			t.Fatalf("expected 4 outputs, got %d: %v", len(got), got)
		}
		if got[0] != 10 {
			t.Errorf("expected 10, got %d", got[0])
		}
		if got[1] != 20 {
			t.Errorf("expected 20, got %d", got[1])
		}
		if got[2] != 30 {
			t.Errorf("expected 30, got %d", got[2])
		}
		if got[3] != 40 {
			t.Errorf("expected 40, got %d", got[3])
		}
	})
}

func TestIntegrationDeclareInit(t *testing.T) {
	testArchs(t, `DECLARE x BYTE = 42
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

func TestIntegrationDeclareInitWord(t *testing.T) {
	testArchs(t, `DECLARE w WORD = 0xFEED
OUTPUT WORD 0 w
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 2 {
			t.Fatalf("expected 2 outputs, got %d: %v", len(res.OutBytes[0]), res.OutBytes[0])
		}
		if res.OutBytes[0][0] != 0xED {
			t.Errorf("expected lo byte 0xED, got 0x%02x", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 0xFE {
			t.Errorf("expected hi byte 0xFE, got 0x%02x", res.OutBytes[0][1])
		}
	})
}

func TestIntegrationDeclareInitWordByte(t *testing.T) {
	testArchs(t, `DECLARE w WORD = 0xFEED
DECLARE a BYTE
LET a = BYTE(w)
OUTPUT 0 a
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 0xED {
			t.Errorf("expected 0xED, got 0x%02x", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationDeclareInitLocal(t *testing.T) {
	testArchs(t, `DECLARE x BYTE = 10
DECLARE y BYTE = 20
PROCEDURE add() BYTE
  DECLARE s BYTE = 0
  LET s = x + y
  RETURN s
END
OUTPUT 0 add()
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 30 {
			t.Errorf("expected 30, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_ElseIf(t *testing.T) {
	testArchs(t, `DECLARE x BYTE
LET x = 1
IF x == 0 THEN OUTPUT 0 10 ELSE IF x == 1 THEN OUTPUT 0 20
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 2 {
			t.Fatal("expected 2 outputs")
		}
		if res.OutBytes[0][0] != 20 {
			t.Errorf("expected ELSE IF branch to output 20, got %d", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 1 {
			t.Errorf("expected x=1 unchanged, got %d", res.OutBytes[0][1])
		}
	})
}

func TestIntegrationPIR_ElseIfElse(t *testing.T) {
	testArchs(t, `DECLARE x BYTE
LET x = 2
IF x == 0 THEN OUTPUT 0 10 ELSE IF x == 1 THEN OUTPUT 0 20 ELSE OUTPUT 0 30
OUTPUT 0 x
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 2 {
			t.Fatal("expected 2 outputs")
		}
		if res.OutBytes[0][0] != 30 {
			t.Errorf("expected ELSE branch to output 30, got %d", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 2 {
			t.Errorf("expected x=2 unchanged, got %d", res.OutBytes[0][1])
		}
	})
}

func TestIntegrationPIR_CaseDo(t *testing.T) {
	testArchs(t, `CASE 1 DO
  OF 0 OUTPUT 0 10
  OF 1 OUTPUT 0 20
  OF 2 OUTPUT 0 30
END
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 20 {
			t.Errorf("expected case 1 to output 20, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_NestedWhile(t *testing.T) {
	testArchs(t, `DECLARE x BYTE
DECLARE y BYTE
DECLARE n BYTE
WHILE x < 3 DO
  WHILE y < 3 DO
    LET n = x * 3 + y + 1
    OUTPUT 0 n
    LET y = y + 1
  END
  LET y = 0
  LET x = x + 1
END
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 9 {
			t.Fatalf("expected 9 outputs, got %d: %v", len(res.OutBytes[0]), res.OutBytes[0])
		}
		expected := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
		for i, e := range expected {
			if res.OutBytes[0][i] != e {
				t.Errorf("output[%d]: expected %d, got %d", i, e, res.OutBytes[0][i])
			}
		}
	})
}
