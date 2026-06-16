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
