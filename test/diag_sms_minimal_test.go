package plz_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/user-none/go-chip-sn76489"
	"github.com/xmasengine/plz/pkg/pir"
	"github.com/xmasengine/plz/pkg/plz"
	"github.com/xmasengine/plz/pkg/sms"
	"github.com/xmasengine/plz/pkg/z80/emu"
	asm "github.com/xmasengine/plz/pkg/z80asm"
)

func compilePLZToSMS(t *testing.T, src string) *sms.VDP {
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
	pirProg, err := prog.GenPIR()
	if err != nil {
		t.Fatalf("genPIR: %v", err)
	}
	pirProg = pir.Optimize(pirProg)
	asmText := pir.NewZ80Gen(pir.DefaultConfig()).Gen(pirProg)
	asmPath := filepath.Join(dir, "test.asm")
	if err := os.WriteFile(asmPath, []byte(asmText), 0644); err != nil {
		t.Fatalf("write asm: %v", err)
	}
	t.Logf("Assembly length %d bytes", len(asmText))
	binPath := filepath.Join(dir, "test.bin")
	if err := asm.AssembleFiles(binPath, []string{asmPath}); err != nil {
		t.Fatalf("assemble: %v", err)
	}
	bin, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read bin: %v", err)
	}
	v := sms.New(false)
	psg := sn76489.New(3579545, 44100, 10000, sn76489.Sega)
	cpu := emu.NewCPU(emu.WithBinary(bin...), emu.WithVDP(v), emu.WithPSG(psg))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := emu.RunSMS(ctx, cpu, v); err != nil {
		t.Fatalf("cpu run: %v", err)
	}
	v.Tick(sms.LinesNTSC * sms.HblankCycles)
	return v
}

func compilePLZFileToSMS(t *testing.T, path string) *sms.VDP {
	t.Helper()
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	tokens, err := plz.ScanFile(path)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	prog := plz.Program{}
	parser := plz.NewParser(tokens)
	if err := prog.Parse(parser); err != nil {
		t.Fatalf("parse: %v", err)
	}
	pirProg, err := prog.GenPIR()
	if err != nil {
		t.Fatalf("genPIR: %v", err)
	}
	pirProg = pir.Optimize(pirProg)
	asmText := pir.NewZ80Gen(pir.DefaultConfig()).Gen(pirProg)
	asmPath := filepath.Join(dir, base+".asm2")
	if err := os.WriteFile(asmPath, []byte(asmText), 0644); err != nil {
		t.Fatalf("write asm: %v", err)
	}
	t.Logf("Assembly (%s) length %d bytes", asmPath, len(asmText))
	binPath := filepath.Join(dir, base+".bin2")
	if err := asm.AssembleFiles(binPath, []string{asmPath}); err != nil {
		t.Fatalf("assemble: %v", err)
	}
	bin, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read bin: %v", err)
	}
	v := sms.New(false)
	psg := sn76489.New(3579545, 44100, 10000, sn76489.Sega)
	cpu := emu.NewCPU(emu.WithBinary(bin...), emu.WithVDP(v), emu.WithPSG(psg))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := emu.RunSMS(ctx, cpu, v); err != nil {
		t.Fatalf("cpu run: %v", err)
	}
	v.Tick(sms.LinesNTSC * sms.HblankCycles)
	return v
}

// Test direct VDP register writes with the corrected OUT_W
func TestDiagSMSDirectWrite(t *testing.T) {
	src := `
start:
  OUTPUT WORD 0xBF 0x8004
  OUTPUT WORD 0xBF 0x8182
`
	v := compilePLZToSMS(t, src)
	t.Logf("Direct write test:")
	for i := 0; i < 11; i++ {
		t.Logf("  R%d = 0x%02X", i, v.Reg(i))
	}
	if v.Reg(0) != 4 || v.Reg(1) != 0x82 {
		t.Fatal("VDP regs not set correctly")
	}
}

// Test via write_vdp_reg procedure from libplz.plz (using INCLUDE)
func TestDiagSMSIncludeWrite(t *testing.T) {
	// Compile the full libplz_test.plz using a fresh compilation
	v := compilePLZFileToSMS(t, "../include/libplz_test.plz")
	t.Logf("Full libplz test:")
	for i := 0; i < 11; i++ {
		t.Logf("  R%d = 0x%02X", i, v.Reg(i))
	}
}
