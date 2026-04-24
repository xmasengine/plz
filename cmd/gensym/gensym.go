package main

import "bufio"
import "fmt"
import "os"
import "strings"
import "slices"

var opmap = map[string]string{
	"(BC)":   "OpPtrBC",
	"(C)":    "OpPortPtrC",
	"(DE)":   "OpPtrDE",
	"(HL)":   "OpPtrHL",
	"(IX)":   "OpPtrIX",
	"(IX+n)": "OpPtrIX, OpOffset",
	"(IY)":   "OpPtrIY",
	"(IY+n)": "OpPtrIY, OpOffset",
	"(N)":    "OpPtrImm8",
	"(NN)":   "OpPtrImm16",
	"(SP)":   "OpPtrSP",
	"0":      "OpInt",
	"1":      "OpInt",
	"10H":    "OpInt",
	"18H":    "OpInt",
	"2":      "OpInt",
	"20H":    "OpInt",
	"28H":    "OpInt",
	"30H":    "OpInt",
	"38H":    "OpInt",
	"8H":     "OpInt",
	"A":      "OpRegA",
	"AF":     "OpRegAF",
	"AF'":    "OpRegAFS",
	"B":      "OpRegB",
	"BC":     "OpRegBC",
	"C":      "OpRegC",
	"D":      "OpRegD",
	"DE":     "OpRegDE",
	"E":      "OpRegE",
	"H":      "OpRegH",
	"HL":     "OpRegHL",
	"I":      "OpRegI",
	"IX":     "OpRegIX",
	"IY":     "OpRegIY",
	"L":      "OpRegL",
	"M":      "OpFlag",
	"N":      "OpFlag",
	"NC":     "OpFlag",
	"NN":     "OpImm16",
	"NZ":     "OpFlag",
	"P":      "OpFlag",
	"PE":     "OpFlag",
	"PO":     "OpFlag",
	"R":      "OpRegR",
	"SP":     "OpRegSP",
	"Z":      "OpFlag",
	"b":      "OpOffset",
	"n":      "OpImm8",
	"r":      "OpReg",
}

const header = `
//line cmd/gensym/gensym.go:63
package asm

type OpOperand int

const (
	OpPtrBC     OpOperand = (1<<17+iota)
	OpPortPtrC
	OpPtrDE
	OpPtrHL
	OpPtrIX
	OpPtrIY
	OpPtrImm8
	OpPtrImm16
	OpPtrSP
	OpRegA
	OpRegAF
	OpRegAFS
	OpRegB
	OpRegBC
	OpRegC
	OpRegD
	OpRegDE
	OpRegE
	OpRegH
	OpRegHL
	OpRegI
	OpRegIX
	OpRegIY
	OpRegL
	OpRegR
	OpRegSP
	OpFlag
	OpImm8
	OpOffset
	OpImm16
	OpReg
	OpInt
	OpString
)


type OpInfo struct {
	Name string
	Size int
	Operands []OpOperand
	OpCode []string
}

type OpCodeFunc func(operands ... byte) []byte

var Ops = []OpInfo {

`

const footer = `
}
`

const MinLine = 42

func main() {
	knownOperands := []string{}
	out := os.Stdout
	fmt.Fprintln(out, header)
	defer fmt.Fprintln(out, footer)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < MinLine || line[0] == '#' {
			continue
		}
		// Mnemonic     Size OP-Code         Clock  SZHPNC  Effect
		mnemonic := line[0:15]
		size := line[15:16]
		opcode := strings.Trim(line[18:34], " \t")
		opcodes := strings.Split(opcode, " ")

		name, operandsString, _ := strings.Cut(mnemonic, " ")
		operands := strings.Split(operandsString, ",")
		fmt.Fprintf(out, `OpInfo { Name: "%s", Size: %s, OpCode: []string{`, name, size)
		for _, opcode := range opcodes {
			top := strings.Trim(opcode, " \t")
			fmt.Fprintf(out, `"0x%s", `, top)
		}

		fmt.Fprintf(out, `}, Operands:[]OpOperand{`)
		for _, operand := range operands {
			top := strings.Trim(operand, " \t")
			if top == "" {
				continue
			}
			fmt.Fprintf(out, `%s, `, opmap[top])
			idx, found := slices.BinarySearch(knownOperands, top)
			if !found {
				knownOperands = slices.Insert(knownOperands, idx, top)
			}
		}
		fmt.Fprintln(out, `} },`)
	}

}
