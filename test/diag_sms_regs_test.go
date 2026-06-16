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

func compileAndRunSMSFragment(t *testing.T, source string) *sms.VDP {
	t.Helper()
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "test.plz")
	if err := os.WriteFile(srcPath, []byte(source), 0644); err != nil {
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

	binPath := filepath.Join(dir, "test.bin")
	if err := asm.AssembleFiles(binPath, []string{asmPath}); err != nil {
		t.Fatalf("assemble: %v\n%s", err, asmText)
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

func TestDiagSMSRegDump(t *testing.T) {
	v := compileAndRunSMSFilePIR(t, "../include/libplz_test.plz")

	t.Logf("VDP regs:")
	for i := 0; i < 11; i++ {
		t.Logf("  R%d = 0x%02X", i, v.Reg(i))
	}
	t.Logf("displayEnable=%v frameIntEnable=%v",
		v.Reg(1)&0x40 != 0, v.Reg(1)&0x20 != 0)

	if v.FrameReady() {
		fb := v.Framebuffer()
		t.Logf("framebuffer size=%d", len(fb))
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				off := (y*256 + x) * 4
				if off+3 < len(fb) {
					t.Logf("  pixel(%d,%d) = R=%d G=%d B=%d", x, y, fb[off], fb[off+1], fb[off+2])
				}
			}
		}
	} else {
		t.Logf("no frame ready")
	}
}
