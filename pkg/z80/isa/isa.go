package isa

import "strconv"

const (
	OpGroupLD = 0b00
)

type Reg byte

const (
	OpRegB Reg = iota
	OpRegC
	OpRegD
	OpRegE
	OpRegH
	OpRegL
	OpRegIndirectHL
	OpRegA
)

type RegPair byte

const (
	OpRegPairBC RegPair = iota
	OpRegPairDE
	OpRegPairHL
	OpRegPairSP
)

type RegPair2 byte

const (
	OpRegPair2BC RegPair2 = iota
	OpRegPair2DE
	OpRegPair2HL
	OpRegPair2AF
)

type ALU byte

const (
	OpALUAdd ALU = iota
	OpALUADC
	OpALUSUB
	OpALUSBC
	OpALUAND
	OpALUXOR
	OpALUOR
	OpALUCP
)

type ROT byte

const (
	OpROTRLC ROT = iota
	OpROTRRC
	OpROTRL
	OpROTRR
	OpROTSLA
	OpROTSRA
	OpROTSLL
	OpROTSRL
)

const (
	OpRegIndirectBC = iota
	OpRegIndirectDE
)

type Displacement int8

type ImmediateByte uint8

type ImmediateWord uint16

type Flag uint8

const (
	FlagCarry     Flag = 1 << 0
	FlagSubstract Flag = 1 << 1
	FlagParity    Flag = 1 << 2
	FlagHalfCarry Flag = 1 << 4
	FlagZero      Flag = 1 << 6
	FlagSign      Flag = 1 << 7
	FlagOverflow       = FlagParity
	FlagNegative       = FlagSubstract
)

func (f *Flag) SetFlag(bit Flag) Flag {
	*f |= bit
	return *f
}

func (f *Flag) ClearFlag(bit Flag) Flag {
	*f &= ^bit
	return *f
}

func (f *Flag) SetBit(bit uint8) Flag {
	return f.SetFlag(1 << bit)
}

func (f *Flag) ClearBit(bit uint8) Flag {
	return f.ClearFlag(1 << bit)
}

func (f Flag) IsFlag(bit Flag) bool {
	return (f & bit) == bit
}

type Opcode uint8

const (
	NOP Opcode = iota
	LD_BC_Imm16
	LD_PtrBC_A
	INC_BC
	INC_B
	DEC_B
	LD_B_Imm8
	RLCA

	EX_AF_xAF
	ADD_HL_BC
	LD_A_PtrBC
	DEC_BC
	INC_C
	DEC_C
	LD_C_Imm8
	RRCA

	DJNZ_Disp
	LD_DE_Imm16
	LD_PtrDE_A
	INC_DE
	INC_D
	DEC_D
	LD_D_Imm8
	RLA

	JR_Disp
	ADD_HL_DE
	LD_A_PtrDE
	DEC_DE
	INC_E
	DEC_E
	LD_E_Imm8
	RRA

	JRNZ_Disp
	LD_HL_Imm16
	LD_PtrImm16_HL
	INC_HL
	INC_H
	DEC_H
	LD_H_Imm8
	DAA

	JRZ_Disp
	ADD_HL_HL
	LD_HL_PtrImm16
	DEC_HL
	INC_L
	DEC_L
	LD_L_Imm8
	CPL

	JRNC_Disp
	LD_SP_Imm16
	LD_PtrImm16_A
	INC_SP
	INC_PtrHL
	DEC_PtrHL
	LD_PtrHL_Imm8
	SCF

	JRC_Disp
	ADD_HL_SP
	LD_A_PtrImm16
	DEC_SP
	INC_A
	DEC_A
	LD_A_Imm8
	CCF

	LD_B_B
	LD_B_C
	LD_B_D
	LD_B_E
	LD_B_H
	LD_B_L
	LD_B_PtrHL
	LD_B_A

	LD_C_B
	LD_C_C
	LD_C_D
	LD_C_E
	LD_C_H
	LD_C_L
	LD_C_PtrHL
	LD_C_A

	LD_D_B
	LD_D_C
	LD_D_D
	LD_D_E
	LD_D_H
	LD_D_L
	LD_D_PtrHL
	LD_D_A

	LD_E_B
	LD_E_C
	LD_E_D
	LD_E_E
	LD_E_H
	LD_E_L
	LD_E_PtrHL
	LD_E_A

	LD_H_B
	LD_H_C
	LD_H_D
	LD_H_E
	LD_H_H
	LD_H_L
	LD_H_PtrHL
	LD_H_A

	LD_L_B
	LD_L_C
	LD_L_D
	LD_L_E
	LD_L_H
	LD_L_L
	LD_L_PtrHL
	LD_L_A

	LD_PtrHL_B
	LD_PtrHL_C
	LD_PtrHL_D
	LD_PtrHL_E
	LD_PtrHL_H
	LD_PtrHL_L
	HALT // Exception
	LD_PtrHL_A

	LD_A_B
	LD_A_C
	LD_A_D
	LD_A_E
	LD_A_H
	LD_A_L
	LD_A_PtrHL
	LD_A_A

	// Add
	ADD_A_B
	ADD_A_C
	ADD_A_D
	ADD_A_E
	ADD_A_H
	ADD_A_L
	ADD_A_PtrHL
	ADD_A_A

	// Add with carry
	ADC_A_B
	ADC_A_C
	ADC_A_D
	ADC_A_E
	ADC_A_H
	ADC_A_L
	ADC_A_PtrHL
	ADC_A_A

	// Subtract
	SUB_A_B
	SUB_A_C
	SUB_A_D
	SUB_A_E
	SUB_A_H
	SUB_A_L
	SUB_A_PtrHL
	SUB_A_A

	// Subtract with carry
	SBC_A_B
	SBC_A_C
	SBC_A_D
	SBC_A_E
	SBC_A_H
	SBC_A_L
	SBC_A_PtrHL
	SBC_A_A

	// BINARY AND
	AND_A_B
	AND_A_C
	AND_A_D
	AND_A_E
	AND_A_H
	AND_A_L
	AND_A_PtrHL
	AND_A_A

	// Binary XOR
	XOR_A_B
	XOR_A_C
	XOR_A_D
	XOR_A_E
	XOR_A_H
	XOR_A_L
	XOR_A_PtrHL
	XOR_A_A

	// Binary OR
	OR_A_B
	OR_A_C
	OR_A_D
	OR_A_E
	OR_A_H
	OR_A_L
	OR_A_PtrHL
	OR_A_A

	// Compare: subtract and set flags but do not set A
	CP_A_B
	CP_A_C
	CP_A_D
	CP_A_E
	CP_A_H
	CP_A_L
	CP_A_PtrHL
	CP_A_A

	RETNZ
	POP_BC
	JPNZ_Imm16
	JP_Imm16
	CALLNZ_Imm16
	PUSH_BC
	ADD_A_Imm8
	RST_0x00

	RETZ
	RET
	JPZ_Imm16
	CB_Prefix
	CALLZ_Imm16
	CALL_Imm16
	ADC_A_Imm8
	RST_0x08

	RETNC
	POP_DE
	JPNC_Imm16
	OUT_Port_A
	CALLNC_Imm16
	PUSH_DE
	SUB_A_Imm8
	RST_0x10

	RETC
	EXX
	JPC_Imm16
	IN_A_Port
	CALLC_Imm16
	DD_Prefix
	SBC_A_Imm8
	RST_0x18

	RETPO
	POP_HL
	JPPO_Imm16
	EX_PtrSP_HL
	CALLPO_Imm16
	PUSH_HL
	AND_A_Imm8
	RST_0x20

	RETPE
	JP_PtrHL
	JPPE_Imm16
	EX_DE_HL
	CALLPE_Imm16
	ED_Prefix
	XOR_A_Imm8
	RST_0x28

	RETP // Return is sign flag is not set that is, positive
	POP_AF
	JPP_Imm16
	DI // Disable interrupts
	CALLP_Imm16
	PUSH_AF
	OR_A_Imm8
	RST_0x30

	RETM
	LD_SP_HL
	JPM_Imm16
	EI // Enable interrupts
	CALLM_Imm16
	FD_Prefix
	CP_A_Imm8
	RST_0x38
)

func (o Opcode) Wait() uint8 {
	return 4
}

func (o Opcode) String() string {
	switch o {
	case NOP:
		return "NOP"
	case LD_BC_Imm16:
		return "LD_BC_Imm16"
	case LD_PtrBC_A:
		return "LD_PtrBC_A"
	case INC_BC:
		return "INC_BC"
	case INC_B:
		return "INC_B"
	case DEC_B:
		return "DEC_B"
	case LD_B_Imm8:
		return "LD_B_Imm8"
	case RLCA:
		return "RLCA"

	case EX_AF_xAF:
		return "EX_AF_xAF"
	case ADD_HL_BC:
		return "ADD_HL_BC"
	case LD_A_PtrBC:
		return "LD_A_PtrBC"
	case DEC_BC:
		return "DEC_BC"
	case INC_C:
		return "INC_C"
	case DEC_C:
		return "DEC_C"
	case LD_C_Imm8:
		return "LD_C_Imm8"
	case RRCA:
		return "RRCA"

	case DJNZ_Disp:
		return "DJNZ_Disp"
	case LD_DE_Imm16:
		return "LD_DE_Imm16"
	case LD_PtrDE_A:
		return "LD_PtrDE_A"
	case INC_DE:
		return "INC_DE"
	case INC_D:
		return "INC_D"
	case DEC_D:
		return "DEC_D"
	case LD_D_Imm8:
		return "LD_D_Imm8"
	case RLA:
		return "RLA"

	case JR_Disp:
		return "JR_Disp"
	case ADD_HL_DE:
		return "ADD_HL_DE"
	case LD_A_PtrDE:
		return "LD_A_PtrDE"
	case DEC_DE:
		return "DEC_DE"
	case INC_E:
		return "INC_E"
	case DEC_E:
		return "DEC_E"
	case LD_E_Imm8:
		return "LD_E_Imm8"
	case RRA:
		return "RRA"

	case JRNZ_Disp:
		return "JRNZ_Disp"
	case LD_HL_Imm16:
		return "LD_HL_Imm16"
	case LD_PtrImm16_HL:
		return "LD_PtrImm16_HL"
	case INC_HL:
		return "INC_HL"
	case INC_H:
		return "INC_H"
	case DEC_H:
		return "DEC_H"
	case LD_H_Imm8:
		return "LD_H_Imm8"
	case DAA:
		return "DAA"

	case JRZ_Disp:
		return "JRZ_Disp"
	case ADD_HL_HL:
		return "ADD_HL_HL"
	case LD_HL_PtrImm16:
		return "LD_HL_PtrImm16"
	case DEC_HL:
		return "DEC_HL"
	case INC_L:
		return "INC_L"
	case DEC_L:
		return "DEC_L"
	case LD_L_Imm8:
		return "LD_L_Imm8"
	case CPL:
		return "CPL"

	case JRNC_Disp:
		return "JRNC_Disp"
	case LD_SP_Imm16:
		return "LD_SP_Imm16"
	case LD_PtrImm16_A:
		return "LD_PtrImm16_A"
	case INC_SP:
		return "INC_SP"
	case INC_PtrHL:
		return "INC_PtrHL"
	case DEC_PtrHL:
		return "DEC_PtrHL"
	case LD_PtrHL_Imm8:
		return "LD_PtrHL_Imm8"
	case SCF:
		return "SCF"

	case JRC_Disp:
		return "JRC_Disp"
	case ADD_HL_SP:
		return "ADD_HL_SP"
	case LD_A_PtrImm16:
		return "LD_A_PtrImm16"
	case DEC_SP:
		return "DEC_SP"
	case INC_A:
		return "INC_A"
	case DEC_A:
		return "DEC_A"
	case LD_A_Imm8:
		return "LD_A_Imm8"
	case CCF:
		return "CCF"

	case LD_B_B:
		return "LD_B_B"
	case LD_B_C:
		return "LD_B_C"
	case LD_B_D:
		return "LD_B_D"
	case LD_B_E:
		return "LD_B_E"
	case LD_B_H:
		return "LD_B_H"
	case LD_B_L:
		return "LD_B_L"
	case LD_B_PtrHL:
		return "LD_B_PtrHL"
	case LD_B_A:
		return "LD_B_A"

	case LD_C_B:
		return "LD_C_B"
	case LD_C_C:
		return "LD_C_C"
	case LD_C_D:
		return "LD_C_D"
	case LD_C_E:
		return "LD_C_E"
	case LD_C_H:
		return "LD_C_H"
	case LD_C_L:
		return "LD_C_L"
	case LD_C_PtrHL:
		return "LD_C_PtrHL"
	case LD_C_A:
		return "LD_C_A"

	case LD_D_B:
		return "LD_D_B"
	case LD_D_C:
		return "LD_D_C"
	case LD_D_D:
		return "LD_D_D"
	case LD_D_E:
		return "LD_D_E"
	case LD_D_H:
		return "LD_D_H"
	case LD_D_L:
		return "LD_D_L"
	case LD_D_PtrHL:
		return "LD_D_PtrHL"
	case LD_D_A:
		return "LD_D_A"

	case LD_E_B:
		return "LD_E_B"
	case LD_E_C:
		return "LD_E_C"
	case LD_E_D:
		return "LD_E_D"
	case LD_E_E:
		return "LD_E_E"
	case LD_E_H:
		return "LD_E_H"
	case LD_E_L:
		return "LD_E_L"
	case LD_E_PtrHL:
		return "LD_E_PtrHL"
	case LD_E_A:
		return "LD_E_A"

	case LD_H_B:
		return "LD_H_B"
	case LD_H_C:
		return "LD_H_C"
	case LD_H_D:
		return "LD_H_D"
	case LD_H_E:
		return "LD_H_E"
	case LD_H_H:
		return "LD_H_H"
	case LD_H_L:
		return "LD_H_L"
	case LD_H_PtrHL:
		return "LD_H_PtrHL"
	case LD_H_A:
		return "LD_H_A"

	case LD_L_B:
		return "LD_L_B"
	case LD_L_C:
		return "LD_L_C"
	case LD_L_D:
		return "LD_L_D"
	case LD_L_E:
		return "LD_L_E"
	case LD_L_H:
		return "LD_L_H"
	case LD_L_L:
		return "LD_L_L"
	case LD_L_PtrHL:
		return "LD_L_PtrHL"
	case LD_L_A:
		return "LD_L_A"

	case LD_PtrHL_B:
		return "LD_PtrHL_B"
	case LD_PtrHL_C:
		return "LD_PtrHL_C"
	case LD_PtrHL_D:
		return "LD_PtrHL_D"
	case LD_PtrHL_E:
		return "LD_PtrHL_E"
	case LD_PtrHL_H:
		return "LD_PtrHL_H"
	case LD_PtrHL_L:
		return "LD_PtrHL_L"
	case HALT:
		return "HALT"
	case LD_PtrHL_A:
		return "LD_PtrHL_A"

	case LD_A_B:
		return "LD_A_B"
	case LD_A_C:
		return "LD_A_C"
	case LD_A_D:
		return "LD_A_D"
	case LD_A_E:
		return "LD_A_E"
	case LD_A_H:
		return "LD_A_H"
	case LD_A_L:
		return "LD_A_L"
	case LD_A_PtrHL:
		return "LD_A_PtrHL"
	case LD_A_A:
		return "LD_A_A"

	// Add
	case ADD_A_B:
		return "ADD_A_B"
	case ADD_A_C:
		return "ADD_A_C"
	case ADD_A_D:
		return "ADD_A_D"
	case ADD_A_E:
		return "ADD_A_E"
	case ADD_A_H:
		return "ADD_A_H"
	case ADD_A_L:
		return "ADD_A_L"
	case ADD_A_PtrHL:
		return "ADD_A_PtrHL"
	case ADD_A_A:
		return "ADD_A_A"

	// Add with carry
	case ADC_A_B:
		return "ADC_A_B"
	case ADC_A_C:
		return "ADC_A_C"
	case ADC_A_D:
		return "ADC_A_D"
	case ADC_A_E:
		return "ADC_A_E"
	case ADC_A_H:
		return "ADC_A_H"
	case ADC_A_L:
		return "ADC_A_L"
	case ADC_A_PtrHL:
		return "ADC_A_PtrHL"
	case ADC_A_A:
		return "ADC_A_A"

	// Subtract
	case SUB_A_B:
		return "SUB_A_B"
	case SUB_A_C:
		return "SUB_A_C"
	case SUB_A_D:
		return "SUB_A_D"
	case SUB_A_E:
		return "SUB_A_E"
	case SUB_A_H:
		return "SUB_A_H"
	case SUB_A_L:
		return "SUB_A_L"
	case SUB_A_PtrHL:
		return "SUB_A_PtrHL"
	case SUB_A_A:
		return "SUB_A_A"

	// Subtract with carry
	case SBC_A_B:
		return "SBC_A_B"
	case SBC_A_C:
		return "SBC_A_C"
	case SBC_A_D:
		return "SBC_A_D"
	case SBC_A_E:
		return "SBC_A_E"
	case SBC_A_H:
		return "SBC_A_H"
	case SBC_A_L:
		return "SBC_A_L"
	case SBC_A_PtrHL:
		return "SBC_A_PtrHL"
	case SBC_A_A:
		return "SBC_A_A"

	// BINARY AND
	case AND_A_B:
		return "AND_A_B"
	case AND_A_C:
		return "AND_A_C"
	case AND_A_D:
		return "AND_A_D"
	case AND_A_E:
		return "AND_A_E"
	case AND_A_H:
		return "AND_A_H"
	case AND_A_L:
		return "AND_A_L"
	case AND_A_PtrHL:
		return "AND_A_PtrHL"
	case AND_A_A:
		return "AND_A_A"

	// Binary XOR
	case XOR_A_B:
		return "XOR_A_B"
	case XOR_A_C:
		return "XOR_A_C"
	case XOR_A_D:
		return "XOR_A_D"
	case XOR_A_E:
		return "XOR_A_E"
	case XOR_A_H:
		return "XOR_A_H"
	case XOR_A_L:
		return "XOR_A_L"
	case XOR_A_PtrHL:
		return "XOR_A_PtrHL"
	case XOR_A_A:
		return "XOR_A_A"

	// Binary OR
	case OR_A_B:
		return "OR_A_B"
	case OR_A_C:
		return "OR_A_C"
	case OR_A_D:
		return "OR_A_D"
	case OR_A_E:
		return "OR_A_E"
	case OR_A_H:
		return "OR_A_H"
	case OR_A_L:
		return "OR_A_L"
	case OR_A_PtrHL:
		return "OR_A_PtrHL"
	case OR_A_A:
		return "OR_A_A"

	// Compare: subtract and set flags but do not set A
	case CP_A_B:
		return "CP_A_B"
	case CP_A_C:
		return "CP_A_C"
	case CP_A_D:
		return "CP_A_D"
	case CP_A_E:
		return "CP_A_E"
	case CP_A_H:
		return "CP_A_H"
	case CP_A_L:
		return "CP_A_L"
	case CP_A_PtrHL:
		return "CP_A_PtrHL"
	case CP_A_A:
		return "CP_A_A"

	case RETNZ:
		return "RETNZ"
	case POP_BC:
		return "POP_BC"
	case JPNZ_Imm16:
		return "JPNZ_Imm16"
	case JP_Imm16:
		return "JP_Imm16"
	case CALLNZ_Imm16:
		return "CALLNZ_Imm16"
	case PUSH_BC:
		return "PUSH_BC"
	case ADD_A_Imm8:
		return "ADD_A_Imm8"
	case RST_0x00:
		return "RST_0x00"

	case RETZ:
		return "RETZ"
	case RET:
		return "RET"
	case JPZ_Imm16:
		return "JPZ_Imm16"
	case CB_Prefix:
		return "CB_Prefix"
	case CALLZ_Imm16:
		return "CALLZ_Imm16"
	case CALL_Imm16:
		return "CALL_Imm16"
	case ADC_A_Imm8:
		return "ADC_A_Imm8"
	case RST_0x08:
		return "RST_0x08"

	case RETNC:
		return "RETNC"
	case POP_DE:
		return "POP_DE"
	case JPNC_Imm16:
		return "JPNC_Imm16"
	case OUT_Port_A:
		return "OUT_Port_A"
	case CALLNC_Imm16:
		return "CALLNC_Imm16"
	case PUSH_DE:
		return "PUSH_DE"
	case SUB_A_Imm8:
		return "SUB_A_Imm8"
	case RST_0x10:
		return "RST_0x10"

	case RETC:
		return "RETC"
	case EXX:
		return "EXX"
	case JPC_Imm16:
		return "JPC_Imm16"
	case IN_A_Port:
		return "IN_A_Port"
	case CALLC_Imm16:
		return "CALLC_Imm16"
	case DD_Prefix:
		return "DD_Prefix"
	case SBC_A_Imm8:
		return "SBC_A_Imm8"
	case RST_0x18:
		return "RST_0x18"

	case RETPO:
		return "RETPO"
	case POP_HL:
		return "POP_HL"
	case JPPO_Imm16:
		return "JPPO_Imm16"
	case EX_PtrSP_HL:
		return "EX_PtrSP_HL"
	case CALLPO_Imm16:
		return "CALLPO_Imm16"
	case PUSH_HL:
		return "PUSH_HL"
	case AND_A_Imm8:
		return "AND_A_Imm8"
	case RST_0x20:
		return "RST_0x20"

	case RETPE:
		return "RETPE"
	case JP_PtrHL:
		return "JP_PtrHL"
	case JPPE_Imm16:
		return "JPPE_Imm16"
	case EX_DE_HL:
		return "EX_DE_HL"
	case CALLPE_Imm16:
		return "CALLPE_Imm16"
	case ED_Prefix:
		return "ED_Prefix"
	case XOR_A_Imm8:
		return "XOR_A_Imm8"
	case RST_0x28:
		return "RST_0x28"

	case RETP:
		return "RETP " // Return is sign flag is not set that is, positive
	case POP_AF:
		return "POP_AF"
	case JPP_Imm16:
		return "JPP_Imm16"
	case DI:
		return "DI" // Disable interrupts
	case CALLP_Imm16:
		return "CALLP_Imm16"
	case PUSH_AF:
		return "PUSH_AF"
	case OR_A_Imm8:
		return "OR_A_Imm8"
	case RST_0x30:
		return "RST_0x30"

	case RETM:
		return "RETM"
	case LD_SP_HL:
		return "LD_SP_HL"
	case JPM_Imm16:
		return "JPM_Imm16"
	case EI:
		return "EI" // Enable interrupts
	case CALLM_Imm16:
		return "CALLM_Imm16"
	case FD_Prefix:
		return "FD_Prefix"
	case CP_A_Imm8:
		return "CP_A_Imm8"
	case RST_0x38:
		return "RST_0x38"
	default:
		return "instruction not possible"
	}
}

type BitOpcode uint8

type BitOpcodeKind uint8

const (
	BitOpcodeKindShift BitOpcodeKind = 0
	BitOpcodeKindTest  BitOpcodeKind = 1
	BitOpcodeKindClear BitOpcodeKind = 2
	BitOpcodeKindSet   BitOpcodeKind = 3
)

type BitOpcodeBit uint8

const (
	BitOpcodeBitRLC BitOpcodeBit = 0
	BitOpcodeBitRRC BitOpcodeBit = 1
	BitOpcodeBitRL  BitOpcodeBit = 2
	BitOpcodeBitRR  BitOpcodeBit = 3
	BitOpcodeBitSLA BitOpcodeBit = 4
	BitOpcodeBitSRA BitOpcodeBit = 5
	BitOpcodeBitSLL BitOpcodeBit = 6
	BitOpcodeBitSRL BitOpcodeBit = 7
)

type RegisterIndex uint8

const (
	RegisterIndexB     RegisterIndex = 0
	RegisterIndexC     RegisterIndex = 1
	RegisterIndexD     RegisterIndex = 2
	RegisterIndexE     RegisterIndex = 3
	RegisterIndexH     RegisterIndex = 4
	RegisterIndexL     RegisterIndex = 5
	RegisterIndexPtrHL RegisterIndex = 6
	RegisterIndexA     RegisterIndex = 7
)

/*
SplitBitCode splits an opcode that is a bit operation into
x, y and z components as per the following:

	Z80 bit instructions are decoded using specific opcode prefixes
	and bit-field patterns within the opcode byte.
	The CB prefix ($CB) decodes bit manipulation instructions
	(Bit, Set, Reset, Rotate/Shift) for general registers and memory
	locations, while DD/FD prefixes combined with CB ($DDCB/$FDCB) extend
	these operations to indexed memory addresses $(IX+d)$ or $(IY+d)$,
	requiring a displacement byte between the prefixes and the CB opcode.

	The decoding algorithm relies on the opcode's three 3-bit fields:

	x (bits 7-6): Determines the instruction group
		(0 for rotate/shift, 1 for Bit test, 2 for Reset, 3 for Set).
	y (bits 5-3): Specifies the bit number
		(0–7) to operate on or the rotation mode.
	z (bits 2-0):
		Identifies the target register or memory location
		namely B, C, D, E, H, L, (HL), A.

	The following table outlines the primary CB-prefixed instruction
	patterns for standard registers and memory:

	x (Bits 7-6)	y (Bits 5-3)	z (Bits 2-0)	Operation	Example Mnemonic
	0	0–7	0–7	Rotate/Shift Register or Memory	RLC r, RL (HL)
	1	0–7	0–7	Test Bit	BIT y, r, BIT y, (HL)
	2	0–7	0–7	Reset Bit	RES y, r, RES y, (HL)
	3	0–7	0–7	Set Bit	SET y, r, SET y, (HL)
	For indexed operations using DDCB or FDCB, the byte sequence is
	[DD/FD] [CB] [displacement] [opcode].
	The displacement byte is a signed 8-bit integer added to the IX or IY
	register to calculate the memory address, effectively replacing the
	(HL) operand in the standard CB table with (IX+d) or (IY+d).
	Undocumented instructions may also utilize these patterns,
	but the official set follows this strict octal-based layout where the
	first octal digit (x) dictates the operation class
*/
func (opcode BitOpcode) SplitBitOpcode() (x BitOpcodeKind, y BitOpcodeBit, z RegisterIndex) {
	x = BitOpcodeKind(opcode >> 6)
	y = BitOpcodeBit(0b00111000&opcode) >> 3
	z = RegisterIndex(0b00000111 & opcode)
	return x, y, z
}

type Instruction interface {
	String() string
	Bytes() []byte
}

func (o BitOpcode) Bytes() []byte {
	return []byte{byte(o)}
}

func (o Opcode) Bytes() []byte {
	return []byte{byte(o)}
}

type Imm8 uint8

func (i Imm8) String() string {
	return strconv.Itoa(int(i))
}

func (i Imm8) Bytes() []byte {
	return []byte{byte(i)}
}

type Imm16 uint16

func (i Imm16) String() string {
	return strconv.Itoa(int(i))
}

func (i Imm16) Bytes() []byte {
	l := uint8(i & 256)
	h := uint8(i >> 6)
	return []byte{l, h}
}

// MiscOpcodes are the opcodes for modified operations after the ED prefix
type MiscOpcode Opcode

const (
	IN_B_PtrBC     MiscOpcode = 0x40
	OUT_PtrBC_B    MiscOpcode = 0x41
	SBC_HL_BC      MiscOpcode = 0x42
	LD_PtrImm16_BC MiscOpcode = 0x43
	NEG            MiscOpcode = 0x44
	RETN           MiscOpcode = 0x45 // Return from NMI handler
	IM0            MiscOpcode = 0x46 // Set interrupt mode to 0
	LD_I_A         MiscOpcode = 0x47 // Load I register (used for IM2 or as a spare)

	IN_C_PtrBC     MiscOpcode = 0x48
	OUT_PtrBC_C    MiscOpcode = 0x49
	ADC_HL_BC      MiscOpcode = 0x4a
	LD_BC_PtrImm16 MiscOpcode = 0x4b

	RETI   MiscOpcode = 0x4d // Return from interrupt handler
	LD_R_A MiscOpcode = 0x4f // Load R register (used for memory refres, or more commonly as RNG)

	IN_D_PtrBC     MiscOpcode = 0x50
	OUT_PtrBC_D    MiscOpcode = 0x51
	SBC_HL_DE      MiscOpcode = 0x52
	LD_PtrImm16_DE MiscOpcode = 0x53
	IM1            MiscOpcode = 0x56 // Set interrupt mode to 1
	LD_A_I         MiscOpcode = 0x57 // Load from I register (used for IM2 or as a spare)

	IN_E_PtrBC     MiscOpcode = 0x58
	OUT_PtrBC_E    MiscOpcode = 0x59
	ADC_HL_DE      MiscOpcode = 0x5a
	LD_DE_PtrImm16 MiscOpcode = 0x5b

	IM2    MiscOpcode = 0x5e // Set interrupt mode to 2
	LD_A_R MiscOpcode = 0x5f // Load R register (used for memory refres, or more commonly as RNG)

	IN_H_PtrBC       MiscOpcode = 0x60
	OUT_PtrBC_H      MiscOpcode = 0x61
	SBC_HL_HL        MiscOpcode = 0x62
	LD_PtrImm16_HL_2 MiscOpcode = 0x63 // Undocumented
	RRD              MiscOpcode = 0x67 // Rotate Right Decimal
	IN_L_PtrBC       MiscOpcode = 0x68
	OUT_PtrBC_L      MiscOpcode = 0x69
	ADC_HL_HL        MiscOpcode = 0x6a
	LD_HL_PtrImm16_2 MiscOpcode = 0x6b
	RLD              MiscOpcode = 0x6f // Rotate Left Decimal

	IN_PtrBC       MiscOpcode = 0x70 // undocumented and not very useful
	OUT_PtrBC_0    MiscOpcode = 0x71 // undocumented and not very useful
	SBC_HL_SP      MiscOpcode = 0x72
	LD_PtrImm16_SP MiscOpcode = 0x73

	IN_A_PtrBC     MiscOpcode = 0x78
	OUT_PtrBC_A    MiscOpcode = 0x79
	ADC_HL_SP      MiscOpcode = 0x7a
	LD_SP_PtrImm16 MiscOpcode = 0x7b

	LDI  MiscOpcode = 0xa0
	CPI  MiscOpcode = 0xa1
	INI  MiscOpcode = 0xa2
	OUTI MiscOpcode = 0xa3
	LDD  MiscOpcode = 0xa8
	CPD  MiscOpcode = 0xa9
	IND  MiscOpcode = 0xaa
	OUTD MiscOpcode = 0xab

	LDIR MiscOpcode = 0xb0
	CPIR MiscOpcode = 0xb1
	INIR MiscOpcode = 0xb2
	OTIR MiscOpcode = 0xb3
	LDDR MiscOpcode = 0xb8
	CPDR MiscOpcode = 0xb9
	INDR MiscOpcode = 0xba
	OTDR MiscOpcode = 0xbb
)

func (m MiscOpcode) String() string {
	switch m {
	case IN_B_PtrBC:
		return "IN_B_PtrBC"
	case OUT_PtrBC_B:
		return "OUT_PtrBC_B"
	case SBC_HL_BC:
		return "SBC_HL_BC"
	case LD_PtrImm16_BC:
		return "LD_PtrImm16_BC"
	case NEG:
		return "NEG"
	case RETN:
		return "RETN"
	case IM0:
		return "IM0"
	case LD_I_A:
		return "LD_I_A"

	case IN_C_PtrBC:
		return "IN_C_PtrBC"
	case OUT_PtrBC_C:
		return "OUT_PtrBC_C"
	case ADC_HL_BC:
		return "ADC_HL_BC"
	case LD_BC_PtrImm16:
		return "LD_BC_PtrImm16"

	case RETI:
		return "RETI"
	case LD_R_A:
		return "LD_R_A"

	case IN_D_PtrBC:
		return "IN_D_PtrBC"
	case OUT_PtrBC_D:
		return "OUT_PtrBC_D"
	case SBC_HL_DE:
		return "SBC_HL_DE"
	case LD_PtrImm16_DE:
		return "LD_PtrImm16_DE"
	case IM1:
		return "IM1"
	case LD_A_I:
		return "LD_A_I"

	case IN_E_PtrBC:
		return "IN_E_PtrBC"
	case OUT_PtrBC_E:
		return "OUT_PtrBC_E"
	case ADC_HL_DE:
		return "ADC_HL_DE"
	case LD_DE_PtrImm16:
		return "LD_DE_PtrImm16"

	case IM2:
		return "IM2"
	case LD_A_R:
		return "LD_A_R"

	case IN_H_PtrBC:
		return "IN_H_PtrBC"
	case OUT_PtrBC_H:
		return "OUT_PtrBC_H"
	case SBC_HL_HL:
		return "SBC_HL_HL"
	case LD_PtrImm16_HL_2:
		return "LD_PtrImm16_HL_2"
	case RRD:
		return "RRD"
	case IN_L_PtrBC:
		return "IN_L_PtrBC"
	case OUT_PtrBC_L:
		return "OUT_PtrBC_L"
	case ADC_HL_HL:
		return "ADC_HL_HL"
	case LD_HL_PtrImm16_2:
		return "LD_HL_PtrImm16_2"
	case RLD:
		return "RLD"

	case IN_PtrBC:
		return "IN_PtrBC"
	case OUT_PtrBC_0:
		return "OUT_PtrBC_0"
	case SBC_HL_SP:
		return "SBC_HL_SP"
	case LD_PtrImm16_SP:
		return "LD_PtrImm16_SP"

	case IN_A_PtrBC:
		return "IN_A_PtrBC"
	case OUT_PtrBC_A:
		return "OUT_PtrBC_A"
	case ADC_HL_SP:
		return "ADC_HL_SP"
	case LD_SP_PtrImm16:
		return "LD_SP_PtrImm16"

	case LDI:
		return "LDI"
	case CPI:
		return "CPI"
	case INI:
		return "INI"
	case OUTI:
		return "OUTI"
	case LDD:
		return "LDD"
	case CPD:
		return "CPD"
	case IND:
		return "IND"
	case OUTD:
		return "OUTD"

	case LDIR:
		return "LDIR"
	case CPIR:
		return "CPIR"
	case INIR:
		return "INIR"
	case OTIR:
		return "OTIR"
	case LDDR:
		return "LDDR"
	case CPDR:
		return "CPDR"
	case INDR:
		return "INDR"
	case OTDR:
		return "OTDR"
	default:
		return "" // possible as not all 256 possible values are defined.
	}
}
