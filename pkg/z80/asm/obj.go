package asm

type OperandKind int

const (
	KindNone OperandKind = iota
	KindImm8
	KindOff8
	KindImm16
	KindPtrImm8
	KindPtrImm16
	KindReg8
	KindReg16
	KindPtrReg8 // C can be contain pointer used for ports.
	KindPtrReg16
	KindRegSpc
	KindFla // Flags
	KindInt // Constant ints like for bit or imm, or for macros.
	KindStr // For macros or other assembler instructions.
)

type Operand interface {
	OperandKind() OperandKind
}

type OperandImm8 uint8

func (OperandImm8) OperandKind() OperandKind {
	return KindImm8
}

var _ Operand = OperandImm8(0)

type OperandImm16 uint16

func (OperandImm16) OperandKind() OperandKind {
	return KindImm16
}

var _ Operand = OperandImm16(0)

type OperandOff8 uint8

func (OperandOff8) OperandKind() OperandKind {
	return KindOff8
}

var _ Operand = OperandOff8(0)

type OperandPtrImm8 uint8

func (OperandPtrImm8) OperandKind() OperandKind {
	return KindPtrImm8
}

var _ Operand = OperandPtrImm8(0)

type OperandPtrImm16 uint16

func (OperandPtrImm16) OperandKind() OperandKind {
	return KindImm16
}

var _ Operand = OperandPtrImm16(0)

type OperandReg8 int

const (
	OperandRegB OperandReg8 = 0
	OperandRegC OperandReg8 = 1
	OperandRegD OperandReg8 = 2
	OperandRegE OperandReg8 = 3
	OperandRegH OperandReg8 = 4
	OperandRegL OperandReg8 = 5
	OperandRegA OperandReg8 = 7 // Skip PtrHL as that is a pointer to reg 16.
)

func (OperandReg8) OperandKind() OperandKind {
	return KindReg8
}

var _ Operand = OperandReg8(0)

type OperandReg16 int

const (
	OperandRegBC OperandReg16 = 0
	OperandRegDE OperandReg16 = 1
	OperandRegHL OperandReg16 = 2
	OperandRegAF OperandReg16 = 3
	OperandRegSP OperandReg16 = 4
	OperandRegIX OperandReg16 = 5
	OperandRegIY OperandReg16 = 6
)

func (OperandReg16) OperandKind() OperandKind {
	return KindReg16
}

var _ Operand = OperandReg16(0)

type OperandRegSpc int

const (
	OperandRegI   OperandRegSpc = 0 // Interrupt register.
	OperandRegR   OperandRegSpc = 1 // Memory refresh register can be read as random seed.
	OperandRegAFS OperandRegSpc = 2 // AF shadow register pair.
)

func (OperandRegSpc) OperandKind() OperandKind {
	return KindRegSpc
}

type OperandPtrReg8 int

const (
	OperandPtrRegC OperandPtrReg8 = 1 // Only posible for C which can be used as a pointer to a port.
)

func (OperandPtrReg8) OperandKind() OperandKind {
	return KindPtrReg8
}

var _ Operand = OperandPtrReg8(0)

type OperandPtrReg16 int

const (
	OperandPtrRegBC OperandPtrReg16 = 0
	OperandPtrRegDE OperandPtrReg16 = 1
	OperandPtrRegHL OperandPtrReg16 = 2
	OperandPtrRegAF OperandPtrReg16 = 3
	OperandPtrRegSP OperandPtrReg16 = 4
	OperandPtrRegIX OperandPtrReg16 = 5
	OperandPtrRegIY OperandPtrReg16 = 6
)

func (OperandPtrReg16) OperandKind() OperandKind {
	return KindPtrReg16
}

var _ Operand = OperandPtrReg16(0)

type OperandFla int

const (
	OperandFlaCarry      OperandFla = 7
	OperandFlaMinus      OperandFla = 6
	OperandFlaNoCarry    OperandFla = 5
	OperandFlaNoZero     OperandFla = 4
	OperandFlaParityEven OperandFla = 3
	OperandFlaParityOdd  OperandFla = 2
	OperandFlaParityZero OperandFla = 1
)

func (OperandFla) OperandKind() OperandKind {
	return KindFla
}

var _ Operand = OperandFla(0)

type OperandInt int

func (OperandInt) OperandKind() OperandKind {
	return KindInt
}

var _ Operand = OperandInt(0)

type OperandString string

func (OperandString) OperandKind() OperandKind {
	return KindString
}

var _ Operand = OperandString(0)

/*

type Operand i

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
*/

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

/*
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
*/
