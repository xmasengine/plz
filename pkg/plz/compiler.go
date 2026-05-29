package plz

import (
	"fmt"

	asm "github.com/xmasengine/plz/pkg/z80asm"
)

// Compile runs the full PL/Z compilation pipeline on the given source file:
// scanning, parsing, semantic checking, code generation, and finally
// assembling the generated Z80 assembly into the specified output format.
//
// The src argument is the path to the PL/Z source file. The out argument is
// the output binary path (without extension). The format argument selects the
// output format: "bin" for a flat binary, or "sms" for a Sega Master System
// ROM image.
func Compile(out, format, src string) error {
	tokens, err := ScanFile(src)
	if err != nil {
		return err
	}

	parser := NewParser(tokens)
	program := Program{}
	err = program.Parse(parser)
	if err != nil {
		return err
	}

	gen, err := NewGenFile(out + ".asm")
	if err != nil {
		return err
	}

	err = program.Gen(gen)
	if err != nil {
		return err
	}

	switch format {
	case "bin":
		err = asm.AssembleFiles(out, []string{gen.FileName()})
	case "sms":
		err = asm.AssembleSMS(out, []string{gen.FileName()})
	default:
		return fmt.Errorf("unknown output format %q", format)
	}
	if err != nil {
		return err
	}
	return nil
}
