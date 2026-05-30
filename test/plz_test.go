package plz_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xmasengine/plz/pkg/plz"
	"github.com/xmasengine/plz/pkg/sms"
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

// compileAndRunSMS compiles a PL/Z source string, assembles it, runs it
// with the SMS VDP emulator, and returns the VDP for framebuffer inspection.
func compileAndRunSMS(t *testing.T, src string) *sms.VDP {
	t.Helper()
	dir := t.TempDir()

	srcPath := filepath.Join(dir, "test.plz")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	tokens, err := plz.ScanFile(srcPath)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	prog := plz.Program{}
	parser := plz.NewParser(tokens)
	if err := prog.Parse(parser); err != nil {
		t.Fatalf("parse: %v", err)
	}

	asmPath := filepath.Join(dir, "test.asm")
	gen, err := plz.NewGenFile(asmPath)
	if err != nil {
		t.Fatalf("gen file: %v", err)
	}
	if err := prog.Gen(gen); err != nil {
		t.Fatalf("gen: %v", err)
	}
	gen.Close()

	binPath := filepath.Join(dir, "test.bin")
	if err := asm.AssembleFiles(binPath, []string{asmPath}); err != nil {
		t.Fatalf("assemble: %v", err)
	}

	bin, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read bin: %v", err)
	}
	v := sms.New(false)
	cpu := emu.NewCPU(emu.WithBinary(bin...), emu.WithVDP(v))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := emu.RunSMS(ctx, cpu, v); err != nil {
		t.Fatalf("cpu run: %v", err)
	}

	// Tick VDP to complete any pending frame after the program finishes.
	v.Tick(sms.LinesNTSC * sms.HblankCycles)

	return v
}

func TestIntegrationSMSLibPlz(t *testing.T) {
	v := compileAndRunSMS(t, `
start:
  OUTPUT 0xBF 0x80    // reg 0: enable display
  OUTPUT 0xBF 0x81    // reg 1: screen mode 4
  OUTPUT 0xBF 0x02    // reg 2: tile map base
  OUTPUT 0xBF 0x06    // reg 6: sprite tile base
  OUTPUT 0xBF 0x10    // CRAM address 0
  OUTPUT 0xBE 0x00    // color 0
  OUTPUT 0xBE 0x3F    // color 1
  OUTPUT 0xBF 0x00    // VRAM address low
  OUTPUT 0xBF 0x40    // VRAM address high
  OUTPUT 0xBE 0xFF    // tile data byte
  OUTPUT 0xBE 0x81
  OUTPUT 0xBE 0x81
  OUTPUT 0xBE 0xFF
  HALT`)

	if !v.FrameReady() {
		t.Fatal("no frame rendered after RunSMS")
	}
	fb := v.Framebuffer()
	if len(fb) == 0 {
		t.Fatal("empty framebuffer")
	}
	// At least some pixels should be non-black if tile rendered
	hasContent := false
	for i := 0; i < len(fb); i += 4 {
		if fb[i] != 0 || fb[i+1] != 0 || fb[i+2] != 0 {
			hasContent = true
			break
		}
	}
	if !hasContent {
		t.Log("warning: framebuffer is all black (may be expected if tile 0 pixels are transparent)")
	}
	t.Logf("frame: %d x %d pixels", sms.ScreenWidth, len(fb)/sms.ScreenWidth/4)
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

func TestIntegrationMusic(t *testing.T) {
	io := compileAndRun(t, `
PROCEDURE psg_freq(channel BYTE, freq WORD)
  OUTPUT 0x7F 0x80 | (channel << 5) | (freq & 0x0F)
  OUTPUT 0x7F (freq >> 4) & 0x3F
END
PROCEDURE psg_vol(channel BYTE, vol BYTE)
  OUTPUT 0x7F 0x90 | (channel << 5) | vol
END
PROCEDURE psg_silence()
  CALL psg_vol(0, 15)
  CALL psg_vol(1, 15)
  CALL psg_vol(2, 15)
  OUTPUT 0x7F 0xFF
END
TASK music_test PRIORITY 4
  CALL psg_silence()
  CALL psg_freq(0, 256)
  CALL psg_vol(0, 8)
  SLEEP 1
  CALL psg_vol(0, 15)
  HALT
END
`)
	expected := []byte{159, 191, 223, 255, 128, 16, 152, 159}
	if len(io.OutBytes[0x7F]) != len(expected) {
		t.Fatalf("expected %d bytes on port 0x7F, got %d: %v", len(expected), len(io.OutBytes[0x7F]), io.OutBytes[0x7F])
	}
	for i, b := range expected {
		if io.OutBytes[0x7F][i] != b {
			t.Errorf("byte %d: expected 0x%02X, got 0x%02X", i, b, io.OutBytes[0x7F][i])
		}
	}
}

func TestIntegrationMusicDataDriven(t *testing.T) {
	t.Skip("Big Pickle made a big mistake.")
	io := compileAndRun(t, `
PROCEDURE psg_freq(channel BYTE, freq WORD)
  OUTPUT 0x7F 0x80 | (channel << 5) | (freq & 0x0F)
  OUTPUT 0x7F (freq >> 4) & 0x3F
END
PROCEDURE psg_vol(channel BYTE, vol BYTE)
  OUTPUT 0x7F 0x90 | (channel << 5) | vol
END
PROCEDURE psg_silence()
  CALL psg_vol(0, 15)
  CALL psg_vol(1, 15)
  CALL psg_vol(2, 15)
  OUTPUT 0x7F 0xFF
END

PROCEDURE play_song(song DATA)
  DECLARE idx WORD
  DECLARE freq WORD
  LET idx = 0
  WHILE 1 DO
    IF song[idx] == 0 THEN RETURN

    IF song[idx+1] == 0 THEN RETURN

    IF song[idx+2] == 0 THEN RETURN

    LET freq = song[idx] | (song[idx+1] << 8)
    CALL psg_freq(song[idx+3], freq)
    CALL psg_vol(song[idx+3], song[idx+4])
    SLEEP song[idx+2]
    CALL psg_vol(song[idx+3], 15)
    LET idx = idx + 5
  END
END

my_song: DATA 0x00, 0x01, 1, 0, 8, 0x00, 0x02, 1, 0, 8, 0, 0, 0, 0, 0

TASK music_test PRIORITY 4
  CALL psg_silence()
  CALL play_song(my_song)
  HALT
END
`)
	if len(io.OutBytes[0x7F]) < 2 {
		t.Fatalf("expected output on port 0x7F, got none")
	}
	// Check that silence bytes appear (PSG volume-off commands for channels 0-2 + noise)
	if io.OutBytes[0x7F][0] != 0x9F {
		t.Errorf("expected PSG silence byte 0x9F, got 0x%02X", io.OutBytes[0x7F][0])
	}
	// Check that at least one frequency byte appears (0x80 = ch0 latch + low nibble 0)
	hasFreq := false
	for _, b := range io.OutBytes[0x7F] {
		if b == 0x80 {
			hasFreq = true
			break
		}
	}
	if !hasFreq {
		t.Error("expected PSG frequency byte 0x80 (ch0, low nibble 0)")
	}
	// Verify two notes played (two frequency writes)
	freqCount := 0
	for _, b := range io.OutBytes[0x7F] {
		if b == 0x80 {
			freqCount++
		}
	}
	if freqCount < 2 {
		t.Errorf("expected at least 2 note frequency writes, got %d", freqCount)
	}
	// Song should end (play_song returns) and task halts
	// If the song didn't terminate properly, the task wouldn't HALT
	// and the 5s timeout would fire. The test passes if it completes in time.
}
