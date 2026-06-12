package plz_test

import (
	"context"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/beevik/go6502/cpu"
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

// RunResult captures output from a multi-platform test run.
type RunResult struct {
	OutBytes map[int][]byte // port → values (populated from arch-specific source)
	Mem      *cpu.FlatMemory
}

// compilePIR compiles PLZ source to a PIR program (shared frontend).
func compilePIR(t *testing.T, src string) *pir.Program {
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
	return pirProg
}

// readOutBytes reads the sequential output buffer at OutputBase and parses
// it into a map[port][]byte matching the Z80 ByteIO structure.
// Format: each value written by OUT_B occupies 1 byte. Since the port
// number is not stored, all bytes are assigned to port 0.
func readOutBytes(mem *cpu.FlatMemory, outputBase uint16, maxBytes int) map[int][]byte {
	out := map[int][]byte{}
	for i := 0; i < maxBytes; i++ {
		b := mem.LoadByte(outputBase + uint16(i))
		if b == 0 && i > 0 && mem.LoadByte(outputBase+uint16(i)-1) == 0 {
			break // stop at first run of zeros (probably uninitialized)
		}
		out[0] = append(out[0], b)
	}
	return out
}

func ioOutToMap(arr [255][]byte) map[int][]byte {
	m := make(map[int][]byte, 255)
	for i, v := range arr {
		if len(v) > 0 {
			m[i] = v
		}
	}
	return m
}

// compileAndRunArch compiles PLZ → PIR, runs on the given arch, and
// returns results. For z80 the OutBytes come from port I/O; for 6502/nes
// they come from the memory-mapped output buffer.
func compileAndRunArch(t *testing.T, src string, arch string) *RunResult {
	t.Helper()
	pirProg := compilePIR(t, src)

	switch arch {
	case "z80":
		asmText := pir.NewZ80Gen(pir.DefaultConfig()).Gen(pirProg)
		asmPath := filepath.Join(t.TempDir(), "test.asm")
		if err := os.WriteFile(asmPath, []byte(asmText), 0644); err != nil {
			t.Fatalf("write asm: %v", err)
		}
		binPath := filepath.Join(t.TempDir(), "test.bin")
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
			t.Fatalf("z80 run: %v", err)
		}
		return &RunResult{OutBytes: ioOutToMap(io.OutBytes)}

	case "6502":
		cfg := pir.Default6502Config()
		asmText := pir.NewGen6502(cfg).Gen(pirProg)
		mem := run6502asm(t, asmText, cfg.Origin)
		out := readOutBytes(mem, cfg.OutputBase, 1024)
		return &RunResult{OutBytes: out, Mem: mem}

	case "nes":
		cfg := pir.NES6502Config()
		asmText := pir.NewGen6502(cfg).Gen(pirProg)
		rom, err := pir.Assemble6502(cfg, asmText)
		if err != nil {
			t.Fatalf("Assemble6502: %v\n%s", err, asmText)
		}
		prgData := rom[16:]
		prgSize := int(rom[4]) * 0x4000
		mem := cpu.NewFlatMemory()
		for i, b := range prgData[:prgSize] {
			mem.StoreByte(uint16(0x8000+i), b)
		}
		resetVec := uint16(mem.LoadByte(0xFFFC)) | uint16(mem.LoadByte(0xFFFD))<<8
		emu := cpu.NewCPU(cpu.NMOS, mem)
		emu.Reg.PC = resetVec
		bd := &brkDone6502{}
		emu.AttachBrkHandler(bd)
		const maxSteps = 500000
		for i := 0; i < maxSteps && !bd.done; i++ {
			emu.Step()
		}
		if !bd.done {
			t.Fatalf("NES program did not complete after %d steps; PC=$%04X", maxSteps, emu.Reg.PC)
		}
		out := readOutBytes(mem, cfg.OutputBase, 1024)
		return &RunResult{OutBytes: out, Mem: mem}

	default:
		t.Fatalf("unknown arch %q", arch)
		return nil
	}
}

// testArchs runs the given test function on all available architectures.
// Optional archs argument specifies which architectures to test; defaults to all.
func testArchs(t *testing.T, src string, fn func(t *testing.T, res *RunResult), archs ...string) {
	if len(archs) == 0 {
		archs = []string{"z80", "6502", "nes"}
	}
	for _, arch := range archs {
		t.Run(arch, func(t *testing.T) {
			res := compileAndRunArch(t, src, arch)
			fn(t, res)
		})
	}
}
