package main

import (
	"github.com/xmasengine/plz/pkg/z80/isa"
)

import (
	"fmt"
)

func genISA() {
	fmt.Println("package isa")
	fmt.Println("")
	fmt.Println("const (")
	names := make(map[isa.BitOpcode]string)

	for i := 0; i <= 255; i++ {
		o := isa.BitOpcode(i)
		x, y, z := o.SplitBitOpcode()
		name := ""

		switch x {
		case isa.BitOpcodeKindShift:
			switch y {
			case isa.BitOpcodeBitRLC:
				name = "RLC_"
			case isa.BitOpcodeBitRRC:
				name = "RRC_"
			case isa.BitOpcodeBitRL:
				name = "RL_"
			case isa.BitOpcodeBitRR:
				name = "RR_"
			case isa.BitOpcodeBitSLA:
				name = "SLA_"
			case isa.BitOpcodeBitSRA:
				name = "SRA_"
			case isa.BitOpcodeBitSLL:
				name = "SLL_"
			case isa.BitOpcodeBitSRL:
				name = "SRL_"

			}
		case isa.BitOpcodeKindTest:
			name = fmt.Sprintf("BIT_%d_", y)
		case isa.BitOpcodeKindClear:
			name = fmt.Sprintf("RES_%d_", y)
		case isa.BitOpcodeKindSet:
			name = fmt.Sprintf("SET_%d_", y)
		}

		reg := ""

		switch z {
		case isa.RegisterIndexB:
			reg = "B"
		case isa.RegisterIndexC:
			reg = "C"
		case isa.RegisterIndexD:
			reg = "D"
		case isa.RegisterIndexE:
			reg = "E"
		case isa.RegisterIndexH:
			reg = "H"
		case isa.RegisterIndexL:
			reg = "L"
		case isa.RegisterIndexPtrHL:
			reg = "PtrHL"
		case isa.RegisterIndexA:
			reg = "A"
		}
		name = name + reg
		names[o] = name
		fmt.Printf("\t%s BitOpcode = %d\n", name, o)
	}
	fmt.Println(")")

	fmt.Println("func (o BitOpcode) String() string {")
	fmt.Println("\tswitch b {")
	for i := 0; i <= 255; i++ {
		o := isa.BitOpcode(i)
		name := names[o]
		fmt.Printf("\tcase %s: return \"%s\"\n", name, name)

	}
	fmt.Println("\t}")
	fmt.Println("}")
}

const asmHeader = `
package asm


type ObjKind int

const (
	FuncObj ObjKind = iota
	VarObj
	MacroObj
)


type Obj struct {
	Name string
	Kind ObjKind
	Def String
}

type Asm interface {
	Obj(o Obj)
	Emit(codes ...byte)
	At(offset int)
}

type Operand interface {
	Bytes() []byte
	String() string
}

type RegOperand byte
type BitOperand byte
type DispOperand uint8
type Imm8Operand uint8
type Imm16Operand uint16


type Sym struct {
	Act func (asm Asm, operands...Operand) error
}

type Symtab map[string] Sym


func (s *Symtab) Init() *Symtab") {

`

const asmFooter = `
}
`

func genASM() {
	fmt.Println(asmHeader)
	defer fmt.Println(asmFooter)

	names := make(map[isa.BitOpcode]string)

	for i := 0; i <= 255; i++ {
		o := isa.BitOpcode(i)
		x, y, z := o.SplitBitOpcode()
		name := ""

		switch x {
		case isa.BitOpcodeKindShift:
			switch y {
			case isa.BitOpcodeBitRLC:
				name = "RLC"
			case isa.BitOpcodeBitRRC:
				name = "RRC"
			case isa.BitOpcodeBitRL:
				name = "RL"
			case isa.BitOpcodeBitRR:
				name = "RR"
			case isa.BitOpcodeBitSLA:
				name = "SLA"
			case isa.BitOpcodeBitSRA:
				name = "SRA"
			case isa.BitOpcodeBitSLL:
				name = "SLL"
			case isa.BitOpcodeBitSRL:
				name = "SRL"
			}
		case isa.BitOpcodeKindTest:
			name = fmt.Sprintf("BIT")
		case isa.BitOpcodeKindClear:
			name = fmt.Sprintf("RES")
		case isa.BitOpcodeKindSet:
			name = fmt.Sprintf("SET")
		}

		reg := ""

		switch z {
		case isa.RegisterIndexB:
			reg = "B"
		case isa.RegisterIndexC:
			reg = "C"
		case isa.RegisterIndexD:
			reg = "D"
		case isa.RegisterIndexE:
			reg = "E"
		case isa.RegisterIndexH:
			reg = "H"
		case isa.RegisterIndexL:
			reg = "L"
		case isa.RegisterIndexPtrHL:
			reg = "PtrHL"
		case isa.RegisterIndexA:
			reg = "A"
		}
		name = name + reg
		names[o] = name
		fmt.Printf("\t%s BitOpcode = %d\n", name, o)
	}
	fmt.Println(")")

	fmt.Println("func (o BitOpcode) String() string {")
	fmt.Println("\tswitch b {")
	for i := 0; i <= 255; i++ {
		o := isa.BitOpcode(i)
		name := names[o]
		fmt.Printf("\tcase %s: return \"%s\"\n", name, name)

	}
	fmt.Println("\t}")
	fmt.Println("}")
}
