package plz_test

import "testing"

// TestProcWordReturnOnly tests simplest WORD procedure return (no FOR loop, no local)
func TestProcWordReturnOnly(t *testing.T) {
	testArchs(t, `
DECLARE result WORD

PROCEDURE test(n WORD) WORD
  RETURN n
END

LET result = test(42)
OUTPUT 0 result
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

// TestProcWordLocal tests WORD procedure with local var (no FOR loop)
func TestProcWordLocal(t *testing.T) {
	testArchs(t, `
DECLARE result WORD

PROCEDURE test(n WORD) WORD
  DECLARE x WORD
  LET x = n
  RETURN x
END

LET result = test(42)
OUTPUT 0 result
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

// TestProcWordAdd tests WORD ADD inside procedure (no FOR loop)
func TestProcWordAdd(t *testing.T) {
	testArchs(t, `
DECLARE result WORD

PROCEDURE test(n WORD) WORD
  DECLARE s WORD
  LET s = n
  LET s = s + 1
  RETURN s
END

LET result = test(41)
OUTPUT 0 result
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

// TestProcWordForLoopSimplest tests WORD FOR loop with ADD inside procedure
func TestProcWordForLoopSimplest(t *testing.T) {
	testArchs(t, `
DECLARE result WORD

PROCEDURE test(n WORD) WORD
  DECLARE i WORD
  DECLARE s WORD
  LET s = 0
  FOR i = 1 TO n DO
    LET s = s + 1
  END
  RETURN s
END

LET result = test(5)
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

// TestProcWordForLoopNoBody tests WORD FOR loop with empty body
func TestProcWordForLoopNoBody(t *testing.T) {
	testArchs(t, `
DECLARE result WORD

PROCEDURE test(n WORD) WORD
  DECLARE i WORD
  FOR i = 1 TO n DO
  END
  RETURN i
END

LET result = test(3)
OUTPUT 0 result
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 4 {
			t.Errorf("expected 4, got %d", res.OutBytes[0][0])
		}
	})
}
