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
	plz "github.com/xmasengine/plz/pkg/plz"
	"github.com/xmasengine/plz/pkg/z80/emu"
	asm "github.com/xmasengine/plz/pkg/z80asm"
)

type arguments struct {
	Output     string
	Input      string
	Format     string
	OutputPort int
	InputPort  int
	Mode       struct {
		Assembler bool
		Compiler  bool
		Emulator  bool
		Linker    bool
		Run       bool
		Help      bool
	}
	IncludeDirs string
	Sources     []string
	Timeout     time.Duration
	Ctx         context.Context
}

func (args arguments) emuOpts() []emu.CPUOption {
	opts := []emu.CPUOption{emu.WithReaderWriterIO}
	if args.Input != "" {
		input, err := os.Open(args.Input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot open input %s: %s", args.Input, err)
			os.Exit(2)
		}
		defer input.Close()
		opts = append(opts, emu.WithReader(byte(args.InputPort), input))
	}
	if args.Output != "" {
		var output *os.File
		if args.Output == "-" {
			output = os.Stdout
		} else {
			output, err := os.Create(args.Output)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: cannot open output %s: %s", args.Output, err)
				os.Exit(2)
			}
			defer output.Close()
		}

		opts = append(opts, emu.WithWriter(byte(args.OutputPort), output))
	}
	return opts
}

const defaultInputPort = 60
const defaultOutputPort = 61

func ExitIfError(form string, err error, args ...any) {
	if err != nil {
		args = append(args, err)
		fmt.Fprintf(os.Stderr, "error: "+form+" %s\n", args...)
		os.Exit(2)
	}
}

func main() {
	args := arguments{Ctx: context.Background()}
	flag.BoolVar(&args.Mode.Assembler, "A", false, "Switch to assembler mode")
	flag.BoolVar(&args.Mode.Emulator, "E", false, "Switch to emulator mode")
	flag.BoolVar(&args.Mode.Compiler, "C", true, "Switch to compiler mode")
	flag.BoolVar(&args.Mode.Run, "R", false, "Switch to run mode, compile and emulate.")
	flag.BoolVar(&args.Mode.Help, "h", false, "Display help")
	flag.DurationVar(&args.Timeout, "t", 0, "Emulation timeout")
	flag.StringVar(&args.Output, "o", "", "output file name")
	flag.IntVar(&args.OutputPort, "p", 0, "output port for emulation")
	flag.StringVar(&args.Input, "i", "", "input file name")
	flag.StringVar(&args.Format, "f", "bin", "output file format")
	flag.StringVar(&args.IncludeDirs, "d", "", "inclucde directories")
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

	if args.Mode.Run {
		args.Mode.Compiler = false
	}

	args.Sources = flag.Args()
	if args.Mode.Assembler {
		var err error
		if args.Output == "" {
			fmt.Fprintln(os.Stderr, "error: please specify output file with -o")
			os.Exit(2)
		}
		if args.Format == "bin" {
			err = asm.AssembleFiles(args.Output, args.Sources)
		} else if args.Format == "sms" {
			err = asm.AssembleSMS(args.Output, args.Sources)
		} else {
			err = fmt.Errorf("unknown output format: %s", args.Format)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(2)
		}
	} else if args.Mode.Emulator {
		if len(args.Sources) < 1 {
			fmt.Fprintf(os.Stderr, "error: need one file to emulate")
			os.Exit(2)
		}
		opts := args.emuOpts()
		err := emu.RunFile(args.Ctx, args.Sources[0], opts...)
		ExitIfError("emulator", err)
	} else if args.Mode.Compiler {
		if args.Output == "" {
			fmt.Fprintln(os.Stderr, "error: please specify output file with -o")
			os.Exit(2)
		}
		if len(args.Sources) < 1 {
			fmt.Fprintln(os.Stderr, "error: please specify one source file")
			os.Exit(2)
		}
		src := args.Sources[0]
		err := plz.Compile(args.Output, args.Format, src)
		ExitIfError("compiler", err)
	} else if args.Mode.Run {
		output, err := os.CreateTemp("", "plz_*."+args.Format)
		ExitIfError("run", err)
		defer os.Remove(output.Name())

		if len(args.Sources) < 1 {
			fmt.Fprintln(os.Stderr, "error: please specify one source file")
			os.Exit(2)
		}
		src := args.Sources[0]
		err = plz.Compile(output.Name(), args.Format, src)
		ExitIfError("run", err)

		opts := args.emuOpts()
		err = emu.RunFile(args.Ctx, output.Name(), opts...)
		ExitIfError("run", err)

	}
	os.Exit(0)
}
