package isa

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
	FlagSign      Flag = 1 << 7
	FlagZero      Flag = 1 << 6
	FlagHalfCarry Flag = 1 << 4
	FlagParity    Flag = 1 << 2
	FlagNegative  Flag = 1 << 1
	FlagCarry     Flag = 1 << 0
	FlagOverflow       = FlagParity
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

/*
A+	XOR B	4		XOR C	4		XOR D	    4		XOR E	4		    XOR H	4		XOR L	4		XOR (HL)	7		XOR A	4
B-	OR B	4		OR C	4		OR D	    4		OR E	4		    OR H	4		OR L	4		OR (HL)	7		OR A	4
B+	CP B	4		CP C	4		CP D	    4		CP E	4		    CP H	4		CP L	4		CP (HL)	7		CP A	4
C-	RET NZ	11	5	POP BC	10		JP NZ,nn	10		JP nn	10		    CALL NZ,nn	17	10	PUSH BC	11		ADD A,n	7		RST 00	11
C+	RET Z	11	5	RET	    10		JP Z,nn	    10		--- CB ---			CALL Z,nn	17	10	CALL nn	17		ADC A,n	7		RST 08	11
D-	RET NC	11	5	POP DE	10		JP NC,nn	10		OUT (n),A	11		CALL NC,nn	17	10	PUSH DE	11		SUB n	7		RST 10	11
D+	RET C	11	5	EXX	    4	    JP C,nn	    10		IN A,(n)	11		CALL C,nn	17	10	--- DD ---			SBC A,n	7		RST 18	11
E-	RET PO	11	5	POP HL	10		JP PO,nn	10		EX (SP),HL	19		CALL PO,nn	17	10	PUSH HL	11		AND n	7		RST 20	11
E+	RET PE	11	5	JP (HL)	4		JP PE,nn	10		EX DE,HL	4		CALL PE,nn	17	10	--- ED ---			XOR n	7		RST 28	11
F-	RET P	11	5	POP AF	10		JP P,nn	10		    DI	4		        CALL P,nn	17	10	PUSH AF	11		OR n	7		RST 30	11
F+	RET M	11	5	LD SP,HL	6	JP M,nn	10		    EI	4		        CALL M,nn	17	10	--- FD ---			CP n	7		RST 38	11

*/

/*

	0 / 8			1 / 9			2 / A			3 / B			4 / C			5 / D			6 / E			7 / F
0-	NOP	4		LD BC,nn	10		LD (BC),A	7		INC BC	6		INC B	4		DEC B	4		LD B,n	7		RLCA	4
0+	EX AF,AF'	4		ADD HL,BC	11		LD A,(BC)	7		DEC BC	6		INC C	4		DEC C	4		LD C,n	7		RRCA	4
1-	DJNZ d	13	8	LD DE,nn	10		LD (DE),A	7		INC DE	6		INC D	4		DEC D	4		LD D,n	7		RLA	4
1+	JR d	12		ADD HL,DE	11		LD A,(DE)	7		DEC DE	6		INC E	4		DEC E	4		LD E,n	7		RRA	4
2-	JR NZ,d	12	7	LD HL,nn	10		LD (nn),HL	16		INC HL	6		INC H	4		DEC H	4		LD H,n	7		DAA	4
2+	JR Z,d	12	7	ADD HL,HL	11		LD HL,(nn)	16		DEC HL	6		INC L	4		DEC L	4		LD L,n	7		CPL	4
*/

/*

3-	JR NC,d	12	7	LD SP,nn	10		LD (nn),A	13		INC SP	6		INC (HL)	7		DEC (HL)	7		LD (HL),n	10		SCF	4
3+	JR C,d	12	7	ADD HL,SP	11		LD A,(nn)	13		DEC SP	6		INC A	4		DEC A	4		LD A,n	7		CCF	4

4-	LD B,B	4		LD B,C	4		LD B,D	4		LD B,E	4		LD B,H	4		LD B,L	4		LD B,(HL)	7		LD B,A	4
4+	LD C,B	4		LD C,C	4		LD C,D	4		LD C,E	4		LD C,H	4		LD C,L	4		LD C,(HL)	7		LD C,A	4
5-	LD D,B	4		LD D,C	4		LD D,D	4		LD D,E	4		LD D,H	4		LD D,L	4		LD D,(HL)	7		LD D,A	4
5+	LD E,B	4		LD E,C	4		LD E,D	4		LD E,E	4		LD E,H	4		LD E,L	4		LD E,(HL)	7		LD E,A	4
6-	LD H,B	4		LD H,C	4		LD H,D	4		LD H,E	4		LD H,H	4		LD H,L	4		LD H,(HL)	7		LD H,A	4
6+	LD L,B	4		LD L,C	4		LD L,D	4		LD L,E	4		LD L,H	4		LD L,L	4		LD L,(HL)	7		LD L,A	4
7-	LD (HL),B	7		LD (HL),C	7		LD (HL),D	7		LD (HL),E	7		LD (HL),H	7		LD (HL),L	7		HALT	4		LD (HL),A	7
7+	LD A,B	4		LD A,C	4		LD A,D	4		LD A,E	4		LD A,H	4		LD A,L	4		LD A,(HL)	7		LD A,A	4
8-	ADD A,B	4		ADD A,C	4		ADD A,D	4		ADD A,E	4		ADD A,H	4		ADD A,L	4		ADD A,(HL)	7		ADD A,A	4
8+	ADC A,B	4		ADC A,C	4		ADC A,D	4		ADC A,E	4		ADC A,H	4		ADC A,L	4		ADC A,(HL)	7		ADC A,A	4
9-	SUB B	4		SUB C	4		SUB D	4		SUB E	4		SUB H	4		SUB L	4		SUB (HL)	7		SUB A	4
9+	SBC A,B	4		SBC A,C	4		SBC A,D	4		SBC A,E	4		SBC A,H	4		SBC A,L	4		SBC A,(HL)	7		SBC A,A	4
A-	AND B	4		AND C	4		AND D	4		AND E	4		AND H	4		AND L	4		AND (HL)	7		AND A	4

*/

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
func (opcode Opcode) SplitBitOpcode() (x BitOpcodeKind, y BitOpcodeBit, z RegisterIndex) {
	x = BitOpcodeKind(opcode >> 5)
	y = BitOpcodeBit(0b00111000&opcode) >> 3
	z = RegisterIndex(0b00000111 & opcode)
	return x, y, z
}

type ld_a struct{}

func (ld_a) Imm8(b byte) []Opcode {
	return []Opcode{LD_A_Imm8, Opcode(b)}
}

func (ld_a) B() Opcode {
	return LD_A_B
}

var LD_A ld_a
