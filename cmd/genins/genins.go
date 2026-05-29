package main

import "bufio"
import "fmt"
import "os"
import "strings"
import "slices"

var opmap = map[string]string{
	"(BC)":   "KindPtrBC",
	"(C)":    "KindPortPtrC",
	"(DE)":   "KindPtrDE",
	"(HL)":   "KindPtrHL",
	"(IX)":   "KindPtrIX",
	"(IX+n)": "KindPtrIX, KindOffset",
	"(IY)":   "KindPtrIY",
	"(IY+n)": "KindPtrIY, KindOffset",
	"(N)":    "KindPtrImm8",
	"(NN)":   "KindPtrImm16",
	"(SP)":   "KindPtrSP",
	"0":      "KindInt",
	"1":      "KindInt",
	"10H":    "KindInt",
	"18H":    "KindInt",
	"2":      "KindInt",
	"20H":    "KindInt",
	"28H":    "KindInt",
	"30H":    "KindInt",
	"38H":    "KindInt",
	"8H":     "KindInt",
	"A":      "KindRegA",
	"AF":     "KindRegAF",
	"AF'":    "KindRegAFS",
	"B":      "KindRegB",
	"BC":     "KindRegBC",
	"C":      "KindRegC",
	"D":      "KindRegD",
	"DE":     "KindRegDE",
	"E":      "KindRegE",
	"H":      "KindRegH",
	"HL":     "KindRegHL",
	"I":      "KindRegI",
	"IX":     "KindRegIX",
	"IY":     "KindRegIY",
	"L":      "KindRegL",
	"CA":     "KindFlag",
	"MI":     "KindFlag",
	"N":      "KindImm8",
	"NC":     "KindFlag",
	"NN":     "KindImm16",
	"NZ":     "KindFlag",
	"PL":     "KindFlag",
	"PE":     "KindFlag",
	"PO":     "KindFlag",
	"R":      "KindRegR",
	"SP":     "KindRegSP",
	"Z":      "KindFlag",
	"b":      "KindOffset",
	"n":      "KindImm8",
	"r":      "KindReg",
}

const header = `
//line cmd/genins/genins.go:64
package asm

type OperandKind int

const (
	KindPtrBC     OperandKind = (1+iota)
	KindPortPtrC
	KindPtrDE
	KindPtrHL
	KindPtrIX
	KindPtrIY
	KindPtrImm8
	KindPtrImm16
	KindPtrSP
	KindRegA
	KindRegAF
	KindRegAFS
	KindRegB
	KindRegBC
	KindRegC
	KindRegD
	KindRegDE
	KindRegE
	KindRegH
	KindRegHL
	KindRegI
	KindRegIX
	KindRegIY
	KindRegL
	KindRegR
	KindRegSP
	KindFlag
	KindImm8
	KindOffset
	KindImm16
	KindReg
	KindInt
	KindString
)


type OpInfo struct {
	Name string
	Size int
	Operands []OperandKind
	OpCode []string
}

type OpCodeFunc func(asm Asm) error

func Ins(name string, operands...Operands) {

`

const footer = `
}
`

const MinLine = 42

func main() {
	knownOperands := []string{}
	out := os.Stdout
	fmt.Fprint(out, header)
	defer fmt.Fprint(out, footer)

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

		fmt.Fprintf(out, `}, Operands:[]OperandKind{`)
		sep := ""
		for _, operand := range operands {
			top := strings.Trim(operand, " \t")
			if top == "" {
				continue
			}
			op := opmap[top]
			if op != "" {
				fmt.Fprintf(out, `%s%s`, sep, op)
				sep = ", "
				idx, found := slices.BinarySearch(knownOperands, top)
				if !found {
					knownOperands = slices.Insert(knownOperands, idx, top)
				}
			}
		}
		fmt.Fprintln(out, `} },`)
	}

}
