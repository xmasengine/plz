package plz

import (
	"fmt"
	"os"
	"path/filepath"

	pir "github.com/xmasengine/plz/pkg/pir"
	asm "github.com/xmasengine/plz/pkg/z80asm"
)

// Architecture constants for CompileOpt.
const (
	ArchZ80  = "z80"
	Arch6502 = "6502"
	ArchNES  = "nes"
)

// Compile runs the full PL/Z compilation pipeline using the PIR-based
// Z80 generator. Use CompileOpt for other targets.
func Compile(out, format, src string) error {
	return CompileOpt(out, format, src, ArchZ80, false)
}

// CompileOpt runs the PL/Z compilation pipeline with target architecture
// selection and optional legacy generator fallback.
//
// arch selects the target: "z80" (default), "6502", or "nes".
// When legacy is true, the original direct tree-walking generator is used
// instead of the PIR-based pipeline (Z80 only).
func CompileOpt(out, format, src, arch string, legacy bool) error {
	tokens, err := ScanFile(src)
	if err != nil {
		return err
	}

	parser := NewParser(tokens)
	program := Program{IncludedFiles: make(map[string]bool)}
	absSrc, _ := filepath.Abs(src)
	program.IncludedFiles[absSrc] = true
	err = program.Parse(parser)
	if err != nil {
		return err
	}

	// 6502/NES path
	if arch == Arch6502 || arch == ArchNES {
		pirProg, err := program.GenPIR()
		if err != nil {
			return err
		}
		var cfg pir.Gen6502Config
		if arch == ArchNES {
			cfg = pir.NES6502Config()
		} else {
			cfg = pir.Default6502Config()
		}
		pirProg = pir.Optimize(pirProg)
		gen := pir.NewGen6502(cfg)
		asmText := gen.Gen(pirProg)
		cfg.IntHandlerName = gen.IntHandler()
		cfg.NmiHandlerName = gen.NmiHandler()
		bin, err := pir.Assemble6502(cfg, asmText, gen.BankLines())
		if err != nil {
			return err
		}
		return os.WriteFile(out, bin, 0644)
	}

	// Z80 path (default)
	asmPath := out + ".asm"

	if legacy {
		gen, err := NewGenFile(asmPath)
		if err != nil {
			return err
		}
		err = program.Gen(gen)
		if err != nil {
			return err
		}
	} else {
		pirProg, err := program.GenPIR()
		if err != nil {
			return err
		}
		pirProg = pir.Optimize(pirProg)
		asmText := pir.NewZ80Gen(pir.DefaultConfig()).Gen(pirProg)
		err = os.WriteFile(asmPath, []byte(asmText), 0644)
		if err != nil {
			return err
		}
	}

	switch format {
	case "bin":
		err = asm.AssembleFiles(out, []string{asmPath})
	case "sms":
		err = asm.AssembleSMS(out, []string{asmPath})
	default:
		return fmt.Errorf("unknown output format %q", format)
	}
	return err
}
