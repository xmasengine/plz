// pls is the central compiler binary that can be used to run the
// compiler, assembler and linker
// Modes are selected with upper case flags, parameters with lower case,
// with the exception of -h which shows help.
package main

import "flag"
import "os"

type arguments struct {
	Output string
	Mode   struct {
		Assembler bool
		Compiler  bool
		Linker    bool
		Help      bool
	}
	Sources []string
}

func main() {
	args := arguments{}
	flag.BoolVar(&args.Mode.Assembler, "A", false, "Switch to assembler mode")
	flag.BoolVar(&args.Mode.Help, "h", false, "Display help")
	flag.StringVar(&args.Output, "o", "", "output file name")
	flag.Parse()
	if args.Mode.Help {
		flag.PrintDefaults()
		os.Exit(1)
	}
	if args.Mode.Assembler {

	}

	os.Exit(0)
}
