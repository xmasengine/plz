package plz

import asm "github.com/xmasengine/plz/pkg/z80asm"

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

	if format == "bin" {
		err = asm.AssembleFiles(out, []string{gen.File.Name()})
	} else if format == "sms" {
		err = asm.AssembleSMS(out, []string{gen.File.Name()})
	}
	if err != nil {
		return err
	}
	return nil
}
