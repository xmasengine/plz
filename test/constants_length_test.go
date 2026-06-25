package plz_test

import (
	"testing"
)

func TestIntegrationPIR_Constant(t *testing.T) {
	testArchs(t, `CONSTANT answer = 42
DECLARE x WORD
LET x = answer
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

func TestIntegrationPIR_Length(t *testing.T) {
	testArchs(t, `DECLARE arr ARRAY [10] BYTE
DECLARE x BYTE
OUTPUT 0 LENGTH(arr)
OUTPUT 0 LENGTH(x)
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 2 {
			t.Fatal("expected 2 outputs")
		}
		if res.OutBytes[0][0] != 10 {
			t.Errorf("expected LENGTH(arr) = 10, got %d", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 1 {
			t.Errorf("expected LENGTH(x) = 1, got %d", res.OutBytes[0][1])
		}
	})
}

func TestIntegrationPIR_LengthBoundsCheck(t *testing.T) {
	testArchs(t, `DECLARE arr ARRAY [5] BYTE
DECLARE idx BYTE
LET idx = 0
WHILE idx < 10 DO
  IF idx < LENGTH(arr) THEN DO
    LET arr[idx] = idx + 1
  END
  LET idx = idx + 1
END
OUTPUT 0 arr[0]
OUTPUT 0 arr[4]
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 2 {
			t.Fatal("expected 2 outputs")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected arr[0] = 1, got %d", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 5 {
			t.Errorf("expected arr[4] = 5, got %d", res.OutBytes[0][1])
		}
	})
}

func TestIntegrationPIR_LengthData(t *testing.T) {
	testArchs(t, `DATA my_data 10, 20, 30
DECLARE n BYTE
LET n = LENGTH(my_data)
OUTPUT 0 n
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected 1 output")
		}
		if res.OutBytes[0][0] != 3 {
			t.Errorf("expected 3, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationPIR_LengthDataTile(t *testing.T) {
	src := "DATA tiles TILE `..XX..XX`\n" +
		"DECLARE n BYTE\n" +
		"LET n = LENGTH(tiles)\n" +
		"OUTPUT 0 n\n" +
		"HALT"
	testArchs(t, src, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected 1 output")
		}
		if res.OutBytes[0][0] != 1 {
			t.Errorf("expected 1, got %d", res.OutBytes[0][0])
		}
	})
}
