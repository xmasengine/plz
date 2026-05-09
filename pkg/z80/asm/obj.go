package asm

type EntryKind int

const (
	FuncObj EntryKind = iota
	VarObj
	LabelObj
	MacroObj
)

type Entry struct {
	Kind EntryKind `json:"kind"`
	Name string    `json:"name"`
	Def  string    `json:"def,omitempty"`
	Src  string    `json:"src,omitempty"`
	Loc  int       `json:"loc,omitempty"`
}

type Obj struct {
	Name    string  `json:"name,omitempty"`
	Src     string  `json:"src,omitempty"`
	Bank    int     `json:"bank,omitempty"`
	Entries []Entry `json:"entries"`
}

type Asm interface {
	Entry(o Entry)
	Emit(codes ...byte)
	At(offset int)
}

type Operand interface {
	Bytes() []byte
	String() string
}

/* Possible operands:

Real operands, which require additional bytes to be emitted:

* (IX+d) d signed 2 complements offset, can be used for every (HL).
* (IY+d) d signed 2 complements offset, can be used for every (HL).
* signed 2 complements relative displacement for JMP instructions.
* Imm8: immediate 8 bits value, widely used.
* Imm16: immediate 16 bits value, sometimes used.
* (Imm16): pointer to a 16 bits value, sometimes used.
* Port: immediate 8 bits port number, unsigned, used for I/O.

Pseudo operands which are in fact part of the instruction:

* 8 bit register B,C,D,E,A: widely used.
* 16 bits register pairs BC DE HL AF: widely used.
* (HL) pointer, widely used.
* (IX+d) pointer, can be used for every (HL). The offset is a real operand.
* (IY+d) pointer, can be used for every (HL). The offset is a real operand.
* (BC), (DE) pointers, sometimes used.
* 0,1,2,3,4,5,6,7: bit to use for bit operations
* 0x0, 0x8, 0x16, ... address to use for RST reset location.
* Optionally: flag to use for RET or JMP

We consider these pseudo-operands as separate from the opcode for convienience
and readability. For example:

	LD A 10  // is slightly more readable than
	LDA 10   // or
	LD_A 10  //
	BIT 1 A  // is slightly more readable than
	BIT_1_A  //


*/

type RegOperand byte
type BitOperand byte
type DispOperand uint8
type ResetOperand uint8
type Imm8Operand uint8
type Imm16Operand uint16

type Sym struct {
	Act func(asm Asm, operands ...Operand) error
}

type Symtab map[string]Sym

type Oper interface {
	op()
}

type OperandRegPair byte

func (o OperandRegPair) op() {}

const (
	OperandBC OperandRegPair = iota + 1
	OperandDE
	OperandHL
	OperandIX
	OperandIY
)

type OperandRegister byte

func (o OperandRegister) op() {}

const (
	OperandB OperandRegister = iota
	OperandC
	OperandD
	OperandE
	OperandH
	OperandL
	OperandPtrHL
	OperandA
)

type OperandSpecialRegister byte

func (o OperandSpecialRegister) op() {}

const (
	OperandR OperandSpecialRegister = iota
	OperandI
	OperandAFS
	OperandAF
	OperandF
	OperandSP
)

type OperandFlag byte

func (o OperandFlag) op() {}

const (
	OperandFlagC OperandFlag = iota
	OperandFlagNC
	OperandFlagZ
	OperandFlagNZ
	OperandFlagPO
	OperandFlagPE
	OperandFlagM
)

type OperandPtr struct {
	To Oper
}

func (o OperandPtr) op() {}

var (
	OperandPtrBC    = OperandPtr{To: OperandBC}
	OperandPtrDE    = OperandPtr{To: OperandDE}
	OperandPtrSP    = OperandPtr{To: OperandSP}
	OperandPtrImm8  = OperandPtr{To: OperandImm8(0)}
	OperandPtrImm16 = OperandPtr{To: OperandImm16(0)}
	OperandPtrC     = OperandPtr{To: OperandC}
)

type OperandImm8 uint8

func (o OperandImm8) op() {}

type OperandOffset int8

func (o OperandOffset) op() {}

type OperandImm16 uint16

func (o OperandImm16) op() {}

type OperandPort byte

func (o OperandPort) op() {}

type OperandInt int

func (o OperandInt) op() {}

type OperandString string

func (o OperandString) op() {}
