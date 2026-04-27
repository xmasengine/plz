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

type arguments struct {
	Output string
	Mode   struct {
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

func main() {
	args := arguments{Ctx: context.Background()}
	flag.BoolVar(&args.Mode.Assembler, "A", false, "Switch to assembler mode")
	flag.BoolVar(&args.Mode.Assembler, "E", false, "Switch to emulator mode")
	flag.BoolVar(&args.Mode.Help, "h", false, "Display help")
	flag.DurationVar(&args.Timeout, "t", 0, "Emulation timeout")
	flag.StringVar(&args.Output, "o", "", "output file name")
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
		err := asm.AssembleFile(args.Output, args.Sources)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(2)
		}
	} else if args.Mode.Emulator {
		if len(args.Sources) < 1 {
			fmt.Fprintf(os.Stderr, "error: need one file to emulate")
			os.Exit(2)
		}
		emu.RunFile(args.Ctx, args.Sources[0])
	}

	os.Exit(0)
}
