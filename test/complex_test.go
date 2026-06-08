package plz_test

import (
	"testing"
)

// TestIntegrationBubbleSort exercises nested FOR loops and array writes with
// computed indices (inline swap since parameters are by-value).
func TestIntegrationBubbleSort(t *testing.T) {
	io := compileAndRun(t, `
DECLARE arr ARRAY [6] BYTE
DECLARE ii WORD
DECLARE jj WORD
DECLARE tmp BYTE

LET arr[0] = 5
LET arr[1] = 3
LET arr[2] = 8
LET arr[3] = 1
LET arr[4] = 9
LET arr[5] = 2

FOR ii = 0 TO 4 DO
  FOR jj = 0 TO 4 - ii DO
    IF arr[jj] > arr[jj+1] THEN DO
      LET tmp = arr[jj]
      LET arr[jj] = arr[jj+1]
      LET arr[jj+1] = tmp
    END
  END
END

OUTPUT 0 arr[0]
OUTPUT 0 arr[1]
OUTPUT 0 arr[2]
OUTPUT 0 arr[3]
OUTPUT 0 arr[4]
OUTPUT 0 arr[5]
HALT`)
	if len(io.OutBytes[0]) != 6 {
		t.Fatalf("expected 6 outputs, got %d: %v", len(io.OutBytes[0]), io.OutBytes[0])
	}
	expected := []byte{1, 2, 3, 5, 8, 9}
	for i, v := range expected {
		if io.OutBytes[0][i] != v {
			t.Errorf("arr[%d] = %d, expected %d", i, io.OutBytes[0][i], v)
		}
	}
}

// TestIntegrationFactorial computes 5! using an iterative factorial procedure
// with a FOR loop, demonstrating multi-call arithmetic.
func TestIntegrationFactorial(t *testing.T) {
	io := compileAndRun(t, `
DECLARE result WORD

PROCEDURE fact(n WORD) WORD
  DECLARE i WORD
  DECLARE r WORD
  LET r = 1
  FOR i = 2 TO n DO
    LET r = r * i
  END
  RETURN r
END

LET result = fact(5)
OUTPUT 0 result
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 120 { // 5! = 120
		t.Errorf("expected 120, got %d", io.OutBytes[0][0])
	}
}

// TestIntegrationArraySum computes the sum of array elements using a procedure
// with global array access.
func TestIntegrationArraySum(t *testing.T) {
	io := compileAndRun(t, `
DECLARE arr ARRAY [5] WORD
DECLARE total WORD
DECLARE len WORD

PROCEDURE sum() WORD
  DECLARE i WORD
  DECLARE s WORD
  LET s = 0
  FOR i = 0 TO len DO
    LET s = s + arr[i]
  END
  RETURN s
END

LET arr[0] = 10
LET arr[1] = 20
LET arr[2] = 30
LET arr[3] = 40
LET arr[4] = 50
LET len = 4

LET total = sum()
OUTPUT 0 total
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	// 10+20+30+40+50 = 150
	if io.OutBytes[0][0] != 150 {
		t.Errorf("expected 150, got %d", io.OutBytes[0][0])
	}
}

// TestIntegrationCallChain tests a chain of procedure calls:
// main → outer → inner, with parameters and return values.
func TestIntegrationCallChain(t *testing.T) {
	io := compileAndRun(t, `
DECLARE result WORD

PROCEDURE add(x WORD, y WORD) WORD
  RETURN x + y
END

PROCEDURE double(x WORD) WORD
  RETURN add(x, x)
END

PROCEDURE compute(n WORD) WORD
  DECLARE a WORD
  DECLARE b WORD
  LET a = double(n)
  LET b = add(a, 5)
  RETURN b
END

LET result = compute(10)
OUTPUT 0 result
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	// compute(10): a=double(10)=20, b=add(20,5)=25
	if io.OutBytes[0][0] != 25 {
		t.Errorf("expected 25, got %d", io.OutBytes[0][0])
	}
}

// TestIntegrationNestedForArray builds a multiplication table in an array
// using nested FOR loops with computed indices.
func TestIntegrationNestedForArray(t *testing.T) {
	io := compileAndRun(t, `
DECLARE table ARRAY [16] BYTE
DECLARE ii WORD
DECLARE jj WORD

FOR ii = 0 TO 3 DO
  FOR jj = 0 TO 3 DO
    LET table[ii*4 + jj] = (ii + 1) * (jj + 1)
  END
END

OUTPUT 0 table[0]  // 1*1 = 1
OUTPUT 0 table[1]  // 1*2 = 2
OUTPUT 0 table[4]  // 2*1 = 2
OUTPUT 0 table[5]  // 2*2 = 4
OUTPUT 0 table[15] // 4*4 = 16
HALT`)
	if len(io.OutBytes[0]) != 5 {
		t.Fatalf("expected 5 outputs, got %d: %v", len(io.OutBytes[0]), io.OutBytes[0])
	}
	// Outputs are: table[0], table[1], table[4], table[5], table[15]
	checks := []struct {
		name string
		got  byte
		want byte
	}{
		{"table[0]", io.OutBytes[0][0], 1},   // ii=0,jj=0: (1)*(1) = 1
		{"table[1]", io.OutBytes[0][1], 2},   // ii=0,jj=1: (1)*(2) = 2
		{"table[4]", io.OutBytes[0][2], 2},   // ii=1,jj=0: (2)*(1) = 2
		{"table[5]", io.OutBytes[0][3], 4},   // ii=1,jj=1: (2)*(2) = 4
		{"table[15]", io.OutBytes[0][4], 16}, // ii=3,jj=3: (4)*(4) = 16
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %d, expected %d", c.name, c.got, c.want)
		}
	}
}

// TestIntegrationProcWithForAndVars tests a procedure with local BYTE/WORD
// variables, a FOR loop, arithmetic, and a CALL inside the loop body.
func TestIntegrationProcWithForAndVars(t *testing.T) {
	io := compileAndRun(t, `
DECLARE result WORD

PROCEDURE accumulate(base WORD, n BYTE) WORD
  DECLARE i BYTE
  DECLARE sum WORD
  LET sum = base
  FOR i = 1 TO n DO
    LET sum = sum + i
  END
  RETURN sum
END

LET result = accumulate(10, 5)
OUTPUT 0 result
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	// 10 + 1 + 2 + 3 + 4 + 5 = 25
	if io.OutBytes[0][0] != 25 {
		t.Errorf("expected 25, got %d", io.OutBytes[0][0])
	}
}

// TestIntegrationFibonacci computes the 10th Fibonacci number iteratively
// using a FOR loop with two accumulator variables.
func TestIntegrationFibonacci(t *testing.T) {
	io := compileAndRun(t, `
DECLARE result WORD

PROCEDURE fib(n WORD) WORD
  DECLARE a WORD
  DECLARE b WORD
  DECLARE i WORD
  DECLARE tmp WORD
  IF n == 0 THEN RETURN 0
  IF n == 1 THEN RETURN 1
  LET a = 0
  LET b = 1
  FOR i = 2 TO n DO
    LET tmp = a + b
    LET a = b
    LET b = tmp
  END
  RETURN b
END

LET result = fib(10)
OUTPUT 0 result
OUTPUT 1 result >> 8
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	// fib(10) = 55
	if io.OutBytes[0][0] != 55 {
		t.Errorf("expected 55 (low byte), got %d", io.OutBytes[0][0])
	}
}

// TestIntegrationArrayCopy copies one array to another using a procedure with
// a FOR loop.
func TestIntegrationArrayCopy(t *testing.T) {
	io := compileAndRun(t, `
DECLARE src ARRAY [5] BYTE
DECLARE dst ARRAY [5] BYTE

PROCEDURE copy(len BYTE)
  DECLARE i BYTE
  FOR i = 0 TO len DO
    LET dst[i] = src[i]
  END
END

LET src[0] = 11
LET src[1] = 22
LET src[2] = 33
LET src[3] = 44
LET src[4] = 55

CALL copy(4)

OUTPUT 0 dst[0]
OUTPUT 0 dst[1]
OUTPUT 0 dst[2]
OUTPUT 0 dst[3]
OUTPUT 0 dst[4]
HALT`)
	if len(io.OutBytes[0]) != 5 {
		t.Fatalf("expected 5 outputs, got %d: %v", len(io.OutBytes[0]), io.OutBytes[0])
	}
	expected := []byte{11, 22, 33, 44, 55}
	for i, v := range expected {
		if io.OutBytes[0][i] != v {
			t.Errorf("dst[%d] = %d, expected %d", i, io.OutBytes[0][i], v)
		}
	}
}
