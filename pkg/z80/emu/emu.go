package emu

import "errors"

import "github.com/xmasengine/plz/pkg/z80/isa"

type Registers struct {
	A byte
	B byte
	C byte
	D byte
	E byte
	F isa.Flag
	H byte
	L byte
}

func (r *Registers) SetBC(v uint16) {
	lo := uint8(v & 0xff)
	hi := uint8(v >> 8)
	r.B = hi
	r.C = lo
}

func (r *Registers) SetDE(v uint16) {
	lo := uint8(v & 0xff)
	hi := uint8(v >> 8)
	r.D = hi
	r.E = lo
}

func (r *Registers) SetHL(v uint16) {
	lo := uint8(v & 0xff)
	hi := uint8(v >> 8)
	r.H = hi
	r.L = lo
}

func (r *Registers) BC() uint16 {
	return uint16(r.B<<8) | uint16(r.C)
}

func (r *Registers) DE() uint16 {
	return uint16(r.D<<8) | uint16(r.E)
}

func (r *Registers) HL() uint16 {
	return uint16(r.H<<8) | uint16(r.L)
}

type Memory interface {
	Get(addr uint16) byte
	Put(addr uint16, val byte)
}

type IO interface {
	Input(port byte) byte
	Output(port byte, val byte)
}

type CPU struct {
	Registers
	Shadow    Registers
	Flags     byte
	SP        uint16
	IP        uint16
	IFF1      bool
	IFF2      bool
	Interrupt chan byte
	NMI       chan struct{}
	Clock     chan struct{}
	Wait      byte
	Memory
	IO
}

var InstructionNotImplemented = errors.New("instruction not implemented")

func (c *CPU) GetImm16() uint16 {
	lo := c.GetNext()
	hi := c.GetNext()
	return (uint16(lo) + (uint16(hi) << 8))
}

func (c *CPU) GetImm8() uint8 {
	return c.GetNext()
}

func (c *CPU) GetDisp() int16 {
	return int16(int8(c.GetNext())) // Not too sure this is correct.
}

func (c *CPU) GetNext() uint8 {
	b := c.Memory.Get(c.IP)
	c.IP++
	return b
}

func (c *CPU) Ptr(to uint16) uint8 {
	return c.Memory.Get(to)
}

func (c *CPU) SetPtr(to uint16, val uint8) {
	c.Memory.Put(to, val)
}

// jr gets a displacement and jumps relatively
func (c CPU) jr() {
	disp := c.GetDisp()
	c.IP -= 2
	c.IP = uint16(int16(c.IP) + disp)
}

func (c *CPU) Step() error {
	opcode := isa.Opcode(c.GetNext())
	c.Wait = opcode.Wait()
	switch opcode {
	case isa.NOP:
	case isa.LD_BC_Imm16:
		c.SetBC(c.GetImm16())
	case isa.LD_PtrBC_A:
		c.SetPtr(c.BC(), c.A)
	case isa.INC_BC:
		c.SetBC(c.BC() + 1)
	case isa.INC_B:
		c.B++
	case isa.DEC_B:
		c.B--
	case isa.LD_B_Imm8:
		c.B = c.GetImm8()

	case isa.RLCA:
		bit := c.A >> 7
		c.A = c.A << 1
		if bit > 0 {
			c.F.SetFlag(isa.FlagCarry)
			c.A |= 1
		} else {
			c.F.ClearFlag(isa.FlagCarry)
			c.A &= 0xfe
		}

	case isa.EX_AF_xAF:
		c.A, c.Shadow.A = c.Shadow.A, c.A
		c.F, c.Shadow.F = c.Shadow.F, c.F
	case isa.ADD_HL_BC:
		c.SetHL(c.HL() + c.BC())
	case isa.DEC_BC:
		c.SetBC(c.BC() - 1)
	case isa.INC_C:
		c.C++
	case isa.DEC_C:
		c.C--
	case isa.LD_C_Imm8:
		c.C = c.GetImm8()

	case isa.RRCA:
		bit := c.A & 0xfe
		c.A = c.A >> 1
		if bit > 0 {
			c.F.SetFlag(isa.FlagCarry)
			c.A |= 0xfe
		} else {
			c.F.ClearFlag(isa.FlagCarry)
			c.A &= 0xfe
		}

	case isa.DJNZ_Disp:
		c.B--
		if c.B != 0 {
			c.jr()
		}
	case isa.LD_DE_Imm16:
		c.SetDE(c.GetImm16())
	case isa.LD_PtrDE_A:
		c.SetPtr(c.DE(), c.A)
	case isa.INC_DE:
		c.SetDE(c.DE() + 1)
	case isa.INC_D:
		c.D++
	case isa.DEC_D:
		c.D--
	case isa.LD_D_Imm8:
		c.D = c.GetImm8()
	case isa.RLA:
		old := c.F.IsFlag(isa.FlagCarry)
		bit := c.A >> 7
		c.A = c.A << 1
		if bit > 0 {
			c.F.SetFlag(isa.FlagCarry)
		} else {
			c.F.ClearFlag(isa.FlagCarry)
		}
		if old {
			c.A |= 1
		} else {
			c.A &= 0xfe
		}

	case isa.JR_Disp:
		c.jr()
	case isa.ADD_HL_DE:
		c.SetHL(c.HL() + c.DE())
	case isa.DEC_DE:
		c.SetDE(c.DE() - 1)
	case isa.INC_E:
		c.E++
	case isa.DEC_E:
		c.E--
	case isa.LD_E_Imm8:
		c.E = c.GetImm8()

	case isa.RRA:
		old := c.F.IsFlag(isa.FlagCarry)

		bit := c.A & 0b10000000
		c.A = c.A >> 1

		if bit > 0 {
			c.F.SetFlag(isa.FlagCarry)
		} else {
			c.F.ClearFlag(isa.FlagCarry)
		}
		if old {
			c.A |= 0b10000000
		} else {
			c.A &= 0b01111111
		}

		if bit > 0 {
			c.F.SetFlag(isa.FlagCarry)
			c.A |= 0xfe
		} else {
			c.F.ClearFlag(isa.FlagCarry)
			c.A &= 0xfe
		}

	case isa.JRNZ_Disp:
		if !c.F.IsFlag(isa.FlagZero) {
			c.jr()
		}
	case isa.LD_HL_Imm16:
		c.SetHL(c.GetImm16())
	case isa.LD_PtrImm16_HL:
		addr := c.GetImm16()
		c.SetPtr(addr, c.L)
		c.SetPtr(addr+1, c.H)
	case isa.INC_HL:
		c.SetHL(c.HL() + 1)
	case isa.INC_H:
		c.H++
	case isa.DEC_H:
		c.H--
	case isa.LD_H_Imm8:
		c.D = c.GetImm8()
	case isa.DAA:
		// Not sure what DAA is supposed to do.
		return InstructionNotImplemented

	case isa.JRZ_Disp:
		if c.F.IsFlag(isa.FlagZero) {
			c.jr()
		}
	case isa.ADD_HL_HL:
		c.SetHL(c.HL() + c.HL())
	case isa.DEC_HL:
		c.SetHL(c.HL() - 1)
	case isa.INC_L:
		c.L++
	case isa.DEC_L:
		c.L--
	case isa.LD_L_Imm8:
		c.L = c.GetImm8()

	case isa.CPL:
		c.A = ^c.A

	case isa.JRNC_Disp:
		if !c.F.IsFlag(isa.FlagCarry) {
			c.jr()
		}
	case isa.LD_SP_Imm16:
		c.SP = c.GetImm16()
	case isa.LD_PtrImm16_A:
		addr := c.GetImm16()
		c.SetPtr(addr, c.A)
	case isa.INC_SP:
		c.SP++
	case isa.INC_PtrHL:
		value := c.Ptr(c.HL())
		c.SetPtr(c.HL(), value+1)
	case isa.DEC_PtrHL:
		value := c.Ptr(c.HL())
		c.SetPtr(c.HL(), value-1)
	case isa.LD_PtrHL_Imm8:
		c.D = c.GetImm8()
	case isa.SCF:
		c.F.SetFlag(isa.FlagCarry)
		c.F.ClearFlag(isa.FlagHalfCarry)
		c.F.ClearFlag(isa.FlagNegative)

	case isa.JRC_Disp:
		if c.F.IsFlag(isa.FlagCarry) {
			c.jr()
		}
	case isa.ADD_HL_SP:
		c.SetHL(c.HL() + c.SP)
	case isa.LD_A_PtrImm16:
		c.A = c.Ptr(c.GetImm16())
	case isa.DEC_SP:
		c.SP--
	case isa.INC_A:
		c.A++
	case isa.DEC_A:
		c.A--
	case isa.LD_A_Imm8:
		c.A = c.GetImm8()

	case isa.CCF:
		if c.F.IsFlag(isa.FlagCarry) {
			c.F.ClearFlag(isa.FlagCarry)
			c.F.ClearFlag(isa.FlagHalfCarry)
		} else {
			c.F.SetFlag(isa.FlagCarry)
			c.F.SetFlag(isa.FlagHalfCarry)
		}
		c.F.ClearFlag(isa.FlagNegative)

	default:
		return InstructionNotImplemented
	}
	return nil
}

/*
	NOP Opcode = iota
	LD_BC_Imm16
	LD_PtrBC_A
	INC_BC
	INC_B
	DEC_B
	LD_B_Imm8
	RLCA

/
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
*/
