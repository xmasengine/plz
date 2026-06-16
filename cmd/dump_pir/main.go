package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xmasengine/plz/pkg/pir"
	"github.com/xmasengine/plz/pkg/plz"
)

func main() {
	srcPath := "/home/bjorn/src/plz/include/libplz_test.plz"

	tokens, err := plz.ScanFile(srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scan error: %v\n", err)
		os.Exit(1)
	}

	prog := plz.Program{IncludedFiles: make(map[string]bool)}
	absSrc, _ := filepath.Abs(srcPath)
	prog.IncludedFiles[absSrc] = true
	parser := plz.NewParser(tokens)
	if err := prog.Parse(parser); err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	pirProg, err := prog.GenPIR()
	if err != nil {
		fmt.Fprintf(os.Stderr, "genpir error: %v\n", err)
		os.Exit(1)
	}

	// Optimized
	optProg := pir.Optimize(pirProg)

	fmt.Println("=== PRE-OPTIMIZATION PIR ===")
	fmt.Print(pirProg.String())
	fmt.Println("=== POST-OPTIMIZATION PIR ===")
	fmt.Print(optProg.String())
}
