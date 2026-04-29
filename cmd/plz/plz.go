// pls is the central compiler binary that can be used to run the
// compiler, assembler and linker
// Modes are selected with upper case flags, parameters with lower case,
// with the exception of -h which shows help.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"
)

import (
	"github.com/xmasengine/plz/pkg/z80/asm"
	"github.com/xmasengine/plz/pkg/z80/emu"
)

type InputFileValue struct {
	File *os.File
}

func (f *InputFileValue) Set(name string) (err error) {
	if f.File != nil {
		if f.File != os.Stdin {
			f.File.Close()
		}
		f.File = nil
	}
	if name == "-" {
		f.File = os.Stdin
		return nil
	}
	if name != "" {
		f.File, err = os.Open(name)
		return err
	}
	return nil
}

func (f *InputFileValue) String() string {
	if f.File == nil {
		return ""
	}
	if f.File == os.Stdin {
		return "-"
	}
	return f.File.Name()
}

type OutputFileValue struct {
	File *os.File
}

func (f *OutputFileValue) Set(name string) (err error) {
	if f.File != nil {
		if f.File != os.Stdout {
			f.File.Close()
		}
		f.File = nil
	}
	if name == "-" {
		f.File = os.Stdout
		return nil
	}
	if name != "" {
		f.File, err = os.Create(name)
		return err
	}
	return nil
}

func (f *OutputFileValue) String() string {
	if f.File == nil {
		return ""
	}
	if f.File == os.Stdout {
		return "-"
	}
	return f.File.Name()
}

type arguments struct {
	Output     OutputFileValue
	Input      InputFileValue
	OutputPort int
	InputPort  int
	Mode       struct {
		Assembler bool
		Compiler  bool
		Emulator  bool
		Linker    bool
		Help      bool
	}
	Sources []string
	Timeout time.Duration
	Ctx     context.Context
}

const defaultInputPort = 60
const defaultOutputPort = 61

func main() {
	args := arguments{Ctx: context.Background()}
	flag.BoolVar(&args.Mode.Assembler, "A", false, "Switch to assembler mode")
	flag.BoolVar(&args.Mode.Emulator, "E", false, "Switch to emulator mode")
	flag.BoolVar(&args.Mode.Help, "h", false, "Display help")
	flag.DurationVar(&args.Timeout, "t", 0, "Emulation timeout")
	flag.Var(&args.Output, "o", "output file name")
	flag.IntVar(&args.OutputPort, "p", 0, "output port for emulation")
	flag.Var(&args.Input, "i", "input file name")
	flag.IntVar(&args.InputPort, "q", 0, "input port for emulation")
	flag.Parse()
	if args.Mode.Help {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if args.Timeout > 0 {
		ctx, cancel := context.WithTimeout(args.Ctx, args.Timeout)
		defer cancel()
		args.Ctx = ctx
	}

	args.Sources = flag.Args()
	if args.Mode.Assembler {
		err := asm.AssembleWriter(args.Output.File, args.Sources)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(2)
		}
	} else if args.Mode.Emulator {
		if len(args.Sources) < 1 {
			fmt.Fprintf(os.Stderr, "error: need one file to emulate")
			os.Exit(2)
		}
		opts := []emu.CPUOption{emu.WithReaderWriterIO}
		if args.Input.File != nil {
			opts = append(opts, emu.WithReader(byte(args.InputPort), args.Input.File))
			defer args.Input.File.Close()
		}
		if args.Output.File != nil {
			opts = append(opts, emu.WithWriter(byte(args.OutputPort), args.Output.File))
			defer args.Input.File.Close()
		}

		emu.RunFile(args.Ctx, args.Sources[0], opts...)
	}

	os.Exit(0)
}
