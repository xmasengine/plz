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

// compileAndRun compiles a PL/Z source string, assembles it, loads it into
// the Z80 emulator, runs it, and returns the ByteIO output.
func compileAndRun(t *testing.T, src string) *emu.ByteIO {
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

	io := &emu.ByteIO{}
	cpu := emu.NewCPU(emu.WithBinary(bin...))
	cpu.IO = io

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

// compileAndRunSMSFile compiles a PL/Z source file (in its own directory so
// INCLUDE resolves), assembles it, runs it with the SMS VDP emulator, and
// returns the VDP for framebuffer inspection.
func compileAndRunSMSFile(t *testing.T, path string) *sms.VDP {
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

	asmPath := filepath.Join(dir, base+".asm")
	gen, err := plz.NewGenFile(asmPath)
	if err != nil {
		t.Fatalf("gen file: %v", err)
	}
	if err := prog.Gen(gen); err != nil {
		t.Fatalf("gen: %v", err)
	}
	gen.Close()

	binPath := filepath.Join(dir, base+".bin")
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

func compileAndRunPIR(t *testing.T, src string) *emu.ByteIO {
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

	io := &emu.ByteIO{}
	cpu := emu.NewCPU(emu.WithBinary(bin...))
	cpu.IO = io

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cpu.Run(ctx); err != nil {
		t.Fatalf("cpu run: %v", err)
	}

	return io
}

func frameImage(v *sms.VDP) *image.RGBA {
	fb := v.Framebuffer()
	h := len(fb) / (sms.ScreenWidth * 4)
	r := image.Rect(0, 0, sms.ScreenWidth, h)
	return &image.RGBA{Pix: fb, Stride: sms.ScreenWidth * 4, Rect: r}
}
