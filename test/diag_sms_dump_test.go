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

// TestDiagSMSLibPlzDebug compiles the full libplz_test but dumps
// assembly and checks assembler output.
func TestDiagSMSLibPlzDebug(t *testing.T) {
	path := "../include/libplz_test.plz"
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
		t.Fatalf("genpir: %v", err)
	}
	pirProg = pir.Optimize(pirProg)
	asmText := pir.NewZ80Gen(pir.DefaultConfig()).Gen(pirProg)
	asmPath := filepath.Join(dir, base+".asm")
	if err := os.WriteFile(asmPath, []byte(asmText), 0644); err != nil {
		t.Fatalf("write asm: %v", err)
	}
	t.Logf("Assembly written to %s", asmPath)

	binPath := filepath.Join(dir, base+".bin")
	if err := asm.AssembleFiles(binPath, []string{asmPath}); err != nil {
		t.Fatalf("assemble: %v\n%s", err, asmText)
	}
	t.Logf("Binary written to %s", binPath)

	bin, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read bin: %v", err)
	}
	t.Logf("Binary size: %d bytes", len(bin))

	// Check if the binary has out (191) instructions
	count191 := 0
	count190 := 0
	for i := 0; i < len(bin)-1; i++ {
		// OUT (imm8), A = 0xD3 followed by port byte
		if bin[i] == 0xD3 {
			port := bin[i+1]
			if port == 191 || port == 0xBF {
				count191++
			} else if port == 190 || port == 0xBE {
				count190++
			}
		}
	}
	t.Logf("OUT (191) instructions: %d", count191)
	t.Logf("OUT (190) instructions: %d", count190)

	// Now run and check VDP state
	v := sms.New(false)
	psg := sn76489.New(3579545, 44100, 10000, sn76489.Sega)
	cpu := emu.NewCPU(emu.WithBinary(bin...), emu.WithVDP(v), emu.WithPSG(psg))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := emu.RunSMS(ctx, cpu, v); err != nil {
		t.Fatalf("cpu run: %v", err)
	}
	v.Tick(sms.LinesNTSC * sms.HblankCycles)

	t.Logf("VDP regs after run:")
	for i := 0; i < 11; i++ {
		t.Logf("  R%d = 0x%02X", i, v.Reg(i))
	}
}
