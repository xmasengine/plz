package plz_test

import (
	"testing"
)

func TestIntegrationConstant(t *testing.T) {
	io := compileAndRun(t, `CONSTANT answer = 42
DECLARE x WORD
LET x = answer
OUTPUT 0 x
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationLength(t *testing.T) {
	io := compileAndRun(t, `DECLARE arr ARRAY [10] BYTE
DECLARE x BYTE
OUTPUT 0 LENGTH(arr)
OUTPUT 1 LENGTH(x)
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output port 0")
	}
	if io.OutBytes[0][0] != 10 {
		t.Errorf("expected LENGTH(arr) = 10, got %d", io.OutBytes[0][0])
	}
	if len(io.OutBytes[1]) < 1 {
		t.Fatal("expected output port 1")
	}
	if io.OutBytes[1][0] != 1 {
		t.Errorf("expected LENGTH(x) = 1, got %d", io.OutBytes[1][0])
	}
}

func TestIntegrationLengthBoundsCheck(t *testing.T) {
	io := compileAndRun(t, `DECLARE arr ARRAY [5] BYTE
DECLARE idx BYTE
LET idx = 0
WHILE idx < 10 DO
  IF idx < LENGTH(arr) THEN DO
    LET arr[idx] = idx + 1
  END
  LET idx = idx + 1
END
OUTPUT 0 arr[0]
OUTPUT 1 arr[4]
HALT`)
	if len(io.OutBytes[0]) < 1 || io.OutBytes[0][0] != 1 {
		t.Errorf("expected arr[0] = 1, got %d", io.OutBytes[0][0])
	}
	if len(io.OutBytes[1]) < 1 || io.OutBytes[1][0] != 5 {
		t.Errorf("expected arr[4] = 5, got %d", io.OutBytes[1][0])
	}
}

func TestIntegrationLengthData(t *testing.T) {
	io := compileAndRun(t, `
DATA my_data 10, 20, 30
DECLARE n BYTE
LET n = LENGTH(my_data)
OUTPUT 0 n
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected 1 output")
	}
	if io.OutBytes[0][0] != 3 {
		t.Errorf("expected 3, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationLengthDataTile(t *testing.T) {
	src := "DATA tiles TILE `..XX..XX`\n" +
		"DECLARE n BYTE\n" +
		"LET n = LENGTH(tiles)\n" +
		"OUTPUT 0 n\n" +
		"HALT"
	io := compileAndRun(t, src)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected 1 output")
	}
	if io.OutBytes[0][0] != 1 {
		t.Errorf("expected 1, got %d", io.OutBytes[0][0])
	}
}
