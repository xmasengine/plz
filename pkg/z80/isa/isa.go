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
	FlagSign           Flag = 1 << 7
	FlagZero           Flag = 1 << 6
	FlagHalfCarry      Flag = 1 << 4
	FlagParityOverflow Flag = 1 << 2
	FlagNegative       Flag = 1 << 1
	FlagCarry          Flag = 1 << 0
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

	LD_PtrLH_B
	LD_PtrLH_C
	LD_PtrLH_D
	LD_PtrLH_E
	LD_PtrLH_H
	LD_PtrLH_L
	HALT // Exception
	LD_PtrLH_A

	LD_A_N
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
