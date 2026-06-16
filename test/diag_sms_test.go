package plz_test

import (
	"context"
	"image"
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

func compileAndRunSMSFilePIR(t *testing.T, path string) *sms.VDP {
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
		t.Fatalf("genpir: %v", err)
	}
	pirProg = pir.Optimize(pirProg)
	asmText := pir.NewZ80Gen(pir.DefaultConfig()).Gen(pirProg)
	asmPath := filepath.Join(dir, base+".asm2")
	if err := os.WriteFile(asmPath, []byte(asmText), 0644); err != nil {
		t.Fatalf("write asm: %v", err)
	}

	t.Logf("PIR assembly written to %s", asmPath)

	binPath := filepath.Join(dir, base+".bin2")
	if err := asm.AssembleFiles(binPath, []string{asmPath}); err != nil {
		t.Fatalf("assemble: %v", err)
	}

	bin, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read bin: %v", err)
	}
	t.Logf("binary %d bytes, OUT count=%d", len(bin), countOUT(bin))

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

func countOUT(bin []byte) int {
	n := 0
	for i := 0; i < len(bin)-1; i++ {
		if bin[i] == 0xD3 && bin[i+1] == 191 {
			n++
		}
	}
	return n
}

func compileAndRunSMSPIR(t *testing.T, src string) *sms.VDP {
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
		t.Fatalf("genpir: %v", err)
	}
	pirProg = pir.Optimize(pirProg)
	asmText := pir.NewZ80Gen(pir.DefaultConfig()).Gen(pirProg)
	asmPath := filepath.Join(dir, "test.asm")
	if err := os.WriteFile(asmPath, []byte(asmText), 0644); err != nil {
		t.Fatalf("write asm: %v", err)
	}

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

func TestDiagSMSLibPlzPIR(t *testing.T) {
	v := compileAndRunSMSFilePIR(t, "../include/libplz_test.plz")
	if !v.FrameReady() {
		t.Fatal("no frame rendered")
	}
	t.Logf("VDP regs:")
	for i := 0; i < 11; i++ {
		t.Logf("  R%d = 0x%02X", i, v.Reg(i))
	}
	t.Logf("displayEnable=%v frameIntEnable=%v",
		v.Reg(1)&0x40 != 0, v.Reg(1)&0x20 != 0)
	fb := v.Framebuffer()
	t.Logf("framebuffer size=%d", len(fb))
	if len(fb) >= 16 {
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				off := (y*256 + x) * 4
				if off+3 < len(fb) {
					t.Logf("  pixel(%d,%d) = R=%d G=%d B=%d", x, y, fb[off], fb[off+1], fb[off+2])
				}
			}
		}
	}
	// Count non-zero VRAM bytes
	vramNonZero := 0
	for i := 0; i < 16384; i++ {
		if v.VRAMAt(uint16(i)) != 0 {
			vramNonZero++
		}
	}
	t.Logf("non-zero VRAM bytes: %d", vramNonZero)
	img := frameImage(v)
	samples := []image.Point{
		{0, 0}, {1, 0}, {2, 0}, {4, 0}, {6, 0},
		{0, 2}, {0, 4}, {0, 6},
		{4, 4}, {2, 6}, {6, 2},
	}
	nonBlack := 0
	for _, p := range samples {
		_, g, _, _ := img.At(p.X, p.Y).RGBA()
		if g != 0 {
			nonBlack++
		}
	}
	if nonBlack == 0 {
		t.Fatal("all sampled pixels are black — tiles not rendering correctly")
	}
	t.Logf("LibPlz (PIR): frame %dx%d, %d/%d sampled pixels non-black",
		img.Bounds().Dx(), img.Bounds().Dy(), nonBlack, len(samples))
}
