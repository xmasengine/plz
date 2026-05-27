package plz_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xmasengine/plz/pkg/plz"
	"github.com/xmasengine/plz/pkg/z80/emu"
	asm "github.com/xmasengine/plz/pkg/z80asm"
)

// compileAndRun compiles a PL/Z source string, assembles it, loads it into
// the Z80 emulator, runs it, and returns the ByteIO output.
func compileAndRun(t *testing.T, src string) *emu.ByteIO {
	t.Helper()
	dir := t.TempDir()

	// Write source
	srcPath := filepath.Join(dir, "test.plz")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Parse
	tokens, err := plz.ScanFile(srcPath)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	prog := plz.Program{}
	parser := plz.NewParser(tokens)
	if err := prog.Parse(parser); err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Generate assembly
	asmPath := filepath.Join(dir, "test.asm")
	gen, err := plz.NewGenFile(asmPath)
	if err != nil {
		t.Fatalf("gen file: %v", err)
	}
	if err := prog.Gen(gen); err != nil {
		t.Fatalf("gen: %v", err)
	}
	gen.Close()

	// Assemble
	binPath := filepath.Join(dir, "test.bin")
	if err := asm.AssembleFiles(binPath, []string{asmPath}); err != nil {
		t.Fatalf("assemble: %v", err)
	}

	// Read binary
	bin, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read bin: %v", err)
	}

	// Load into emulator
	io := &emu.ByteIO{}
	cpu := emu.NewCPU(emu.WithBinary(bin...))
	cpu.IO = io

	// Run with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cpu.Run(ctx); err != nil {
		t.Fatalf("cpu run: %v", err)
	}

	return io
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
IF x THEN OUTPUT 0 1 ELSE OUTPUT 0 2
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	// x=0 is falsy, should take ELSE branch
	if io.OutBytes[0][0] != 2 {
		t.Errorf("expected 2, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationIfThen(t *testing.T) {
	io := compileAndRun(t, `DECLARE x WORD
LET x = 1
IF x THEN OUTPUT 0 99
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	// x=1 is truthy, should take THEN branch
	if io.OutBytes[0][0] != 99 {
		t.Errorf("expected 99, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationWhileLoop(t *testing.T) {
	io := compileAndRun(t, `DECLARE count WORD
LET count = 3
WHILE count DO LET count = count - 1; OUTPUT 0 count END
HALT`)
	// Should output: 2, 1, 0 (after decrementing from 3)
	if len(io.OutBytes[0]) != 3 {
		t.Fatalf("expected 3 outputs, got %d: %v", len(io.OutBytes[0]), io.OutBytes[0])
	}
	if io.OutBytes[0][0] != 2 {
		t.Errorf("expected 2, got %d", io.OutBytes[0][0])
	}
	if io.OutBytes[0][1] != 1 {
		t.Errorf("expected 1, got %d", io.OutBytes[0][1])
	}
	if io.OutBytes[0][2] != 0 {
		t.Errorf("expected 0, got %d", io.OutBytes[0][2])
	}
}

func TestIntegrationForLoop(t *testing.T) {
	io := compileAndRun(t, `DECLARE cnt WORD
FOR cnt = 1 TO 3 DO OUTPUT 0 cnt END
HALT`)
	if len(io.OutBytes[0]) != 3 {
		t.Fatalf("expected 3 outputs, got %d: %v", len(io.OutBytes[0]), io.OutBytes[0])
	}
	if io.OutBytes[0][0] != 1 {
		t.Errorf("expected 1, got %d", io.OutBytes[0][0])
	}
	if io.OutBytes[0][1] != 2 {
		t.Errorf("expected 2, got %d", io.OutBytes[0][1])
	}
	if io.OutBytes[0][2] != 3 {
		t.Errorf("expected 3, got %d", io.OutBytes[0][2])
	}
}

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
	// 5 == 5 should be 1 (true)
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
	// 10 > 5 should be 1 (true)
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
	// !0 should be 1 (true)
	if io.OutBytes[0][0] != 1 {
		t.Errorf("expected 1, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationArrayRead(t *testing.T) {
	io := compileAndRun(t, `DECLARE arr WORD
DECLARE tmp WORD
LET arr[0] = 42
LET tmp = arr[0]
OUTPUT 0 tmp
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
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
	// Direct expression in OUTPUT (no intermediate variable)
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

func TestIntegrationRecord(t *testing.T) {
	io := compileAndRun(t, `DECLARE s RECORD x WORD, y BYTE END
LET s.x = 300
LET s.y = 42
OUTPUT 0 s.x
OUTPUT 0 s.y
HALT`)
	if len(io.OutBytes[0]) < 2 {
		t.Fatal("expected 2 outputs")
	}
	if io.OutBytes[0][0] != 44 { // 300 = 0x012C, low byte = 0x2C = 44
		t.Errorf("expected 44 for s.x low byte, got %d", io.OutBytes[0][0])
	}
	if io.OutBytes[0][1] != 42 {
		t.Errorf("expected 42 for s.y, got %d", io.OutBytes[0][1])
	}
}

func TestIntegrationProcCall(t *testing.T) {
	io := compileAndRun(t, `DECLARE result WORD
PROCEDURE add (x WORD, y WORD) WORD
  RETURN x + y
END
CALL add(2, 3)
LET result = add(2, 3)
OUTPUT 0 result
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 5 {
		t.Errorf("expected 5, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationArrayDeclared(t *testing.T) {
	io := compileAndRun(t, `DECLARE arr ARRAY [10] WORD
LET arr[0] = 42
LET arr[1] = 99
OUTPUT 0 arr[0]
OUTPUT 0 arr[1]
HALT`)
	if len(io.OutBytes[0]) < 2 {
		t.Fatal("expected 2 outputs")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
	if io.OutBytes[0][1] != 99 {
		t.Errorf("expected 99, got %d", io.OutBytes[0][1])
	}
}

func TestIntegrationProcLocalDeclare(t *testing.T) {
	io := compileAndRun(t, `DECLARE result WORD
PROCEDURE double (x WORD) WORD
  DECLARE t WORD
  LET t = x + x
  RETURN t
END
LET result = double(21)
OUTPUT 0 result
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationProcByteParam(t *testing.T) {
	io := compileAndRun(t, `DECLARE result WORD
PROCEDURE double (x BYTE) WORD
  RETURN x + x
END
LET result = double(21)
OUTPUT 0 result
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationProcByteReturn(t *testing.T) {
	io := compileAndRun(t, `DECLARE result WORD
PROCEDURE getByte BYTE
  RETURN 42
END
LET result = getByte()
OUTPUT 0 result
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationProcRecordParam(t *testing.T) {
	io := compileAndRun(t, `DECLARE s RECORD x BYTE, y WORD END
DECLARE val WORD
PROCEDURE getX (rv RECORD x BYTE, y WORD END) WORD
  RETURN rv.x
END
LET s.x = 42
LET s.y = 100
LET val = getX(s)
OUTPUT 0 val
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationByteArray(t *testing.T) {
	io := compileAndRun(t, `DECLARE arr BYTE
DECLARE idx WORD
LET arr[0] = 10
LET arr[1] = 20
LET idx = 0
LET idx = arr[idx]
OUTPUT 0 idx
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 10 {
		t.Errorf("expected 10, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationDefineRecordAlias(t *testing.T) {
	io := compileAndRun(t, `DEFINE Point RECORD x WORD, y BYTE END
DECLARE pt TYPE Point
DECLARE val WORD
LET pt.x = 300
LET pt.y = 42
LET val = pt.x
OUTPUT 0 val
OUTPUT 0 pt.y
HALT`)
	if len(io.OutBytes[0]) < 2 {
		t.Fatal("expected 2 outputs")
	}
	if io.OutBytes[0][0] != 44 { // 300 = 0x012C, low byte = 0x2C = 44
		t.Errorf("expected 44 for p.x low byte, got %d", io.OutBytes[0][0])
	}
	if io.OutBytes[0][1] != 42 {
		t.Errorf("expected 42 for p.y, got %d", io.OutBytes[0][1])
	}
}

func TestIntegrationDefineProcAlias(t *testing.T) {
	io := compileAndRun(t, `DEFINE Point RECORD x WORD, y BYTE END
DECLARE pt TYPE Point
DECLARE val WORD
PROCEDURE getY (pp TYPE Point) WORD
  RETURN pp.y
END
LET pt.x = 100
LET pt.y = 77
LET val = getY(pt)
OUTPUT 0 val
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 77 {
		t.Errorf("expected 77, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationDefineByteAlias(t *testing.T) {
	io := compileAndRun(t, `DEFINE MyByte BYTE
DECLARE bv TYPE MyByte
DECLARE val WORD
LET bv = 255
LET val = bv
OUTPUT 0 val
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 255 {
		t.Errorf("expected 255, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationArrayOfRecordsFieldRead(t *testing.T) {
	io := compileAndRun(t, `DECLARE arr ARRAY [4] RECORD x WORD, y BYTE END
DECLARE idx WORD
DECLARE val WORD
LET arr[0].x = 100
LET arr[0].y = 50
LET arr[1].x = 200
LET arr[1].y = 77
LET idx = 1
LET val = arr[idx].x
OUTPUT 0 val
LET val = arr[idx].y
OUTPUT 0 val
LET val = arr[0].x
OUTPUT 0 val
LET val = arr[0].y
OUTPUT 0 val
HALT`)
	if len(io.OutBytes[0]) < 4 {
		t.Fatal("expected 4 outputs")
	}
	if io.OutBytes[0][0] != 200 {
		t.Errorf("expected 200 for arr[1].x, got %d", io.OutBytes[0][0])
	}
	if io.OutBytes[0][1] != 77 {
		t.Errorf("expected 77 for arr[1].y, got %d", io.OutBytes[0][1])
	}
	if io.OutBytes[0][2] != 100 {
		t.Errorf("expected 100 for arr[0].x, got %d", io.OutBytes[0][2])
	}
	if io.OutBytes[0][3] != 50 {
		t.Errorf("expected 50 for arr[0].y, got %d", io.OutBytes[0][3])
	}
}

func TestIntegrationRecordWithArrayField(t *testing.T) {
	io := compileAndRun(t, `DECLARE s RECORD x WORD, arr ARRAY [4] BYTE END
DECLARE idx WORD
DECLARE val WORD
LET s.x = 999
LET s.arr[0] = 10
LET s.arr[1] = 20
LET s.arr[2] = 30
LET s.arr[3] = 40
LET idx = 2
LET val = s.arr[idx]
OUTPUT 0 val
LET val = s.x
OUTPUT 0 val
HALT`)
	if len(io.OutBytes[0]) < 2 {
		t.Fatal("expected 2 outputs")
	}
	if io.OutBytes[0][0] != 30 {
		t.Errorf("expected 30 for s.arr[2], got %d", io.OutBytes[0][0])
	}
	// 999 = 0x03E7, low byte = 0xE7 = 231
	if io.OutBytes[0][1] != 231 {
		t.Errorf("expected 231 for s.x low byte, got %d", io.OutBytes[0][1])
	}
}
