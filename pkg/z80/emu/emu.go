package emu

import "errors"

import "github.com/xmasengine/plz/pkg/z80/isa"

type Registers struct {
	A     byte
	B     byte
	C     byte
	D     byte
	E     byte
	F     isa.Flag
	H     byte
	L     byte
	UseIX bool
	UseIY bool
	IX    uint16
	IY    uint16
	Disp  int16
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

func (r *Registers) SetAF(v uint16) {
	lo := uint8(v & 0xff)
	hi := uint8(v >> 8)
	r.A = hi
	r.F = isa.Flag(lo)
}

func (r *Registers) SetHL(v uint16) {
	if r.UseIX {
		r.UseIX = false
		r.IX = v
		return
	}
	if r.UseIY {
		r.UseIY = false
		r.IY = v
		return
	}

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

func (r *Registers) AF() uint16 {
	return uint16(r.A<<8) | uint16(r.F)
}

func (r *Registers) HL() uint16 {
	if r.UseIX {
		r.UseIX = false
		return r.IX
	}
	if r.UseIY {
		r.UseIY = false
		return r.IY
	}

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
	Shadow                   Registers
	Flags                    byte
	SP                       uint16
	IP                       uint16
	IX                       uint16
	IY                       uint16
	Halted                   bool
	IFF1                     bool
	IFF2                     bool
	Interrupt                chan byte
	NMI                      chan struct{}
	Clock                    chan struct{}
	Wait                     byte
	PrefixCBBitInstructions  bool
	PrefixEDMiscInstructions bool
	Memory
	IO
}

type cpuOption func(*CPU)

type LinearMemory [1 << 16]uint8

func (l LinearMemory) Get(addr uint16) byte {
	return l[addr]
}

func (l *LinearMemory) Put(addr uint16, v byte) {
	l[addr] = v
}

type ByteIO struct {
	Index int
	In    [255][]byte
	Out   [255][]byte
}

func (b *ByteIO) Input(port byte) byte {
	if b.Index >= len(b.In[port]) {
		return 0
	}
	res := b.In[port][b.Index]
	b.Index++
	return res
}

func (b *ByteIO) Output(port byte, val byte) {
	b.Out[port] = append(b.Out[port], val)
}

func Program(program ...isa.Opcode) func(*CPU) {
	return func(c *CPU) {
		for i, op := range program {
			c.Memory.Put(uint16(i), byte(op))
		}
	}
}

func NewCPU(opts ...cpuOption) *CPU {
	cpu := &CPU{}
	cpu.NMI = make(chan struct{})
	cpu.Clock = make(chan struct{})
	cpu.Interrupt = make(chan byte)
	cpu.Memory = &LinearMemory{}
	cpu.IO = &ByteIO{}

	for _, opt := range opts {
		opt(cpu)
	}
	return cpu
}

var InstructionNotImplemented = errors.New("instruction not implemented")
var InstructionNotCorrect = errors.New("instruction not correct")

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

func (c *CPU) SetPtr16(to uint16, val uint16) {
	c.Memory.Put(to, uint8(val&0xff))
	c.Memory.Put(to+1, uint8(val>>8))
}

func (c *CPU) GetPtr16(to uint16) uint16 {
	return uint16(c.Ptr(to)) + (uint16(c.Ptr(to+1)) << 8)
}

func (c *CPU) SetPtrHL(val uint8) {
	c.SetPtr(c.HL(), val)
}

func (c *CPU) PtrHL() uint8 {
	return c.Ptr(c.HL())
}

func (c *CPU) PtrHL16() uint16 {
	return c.GetPtr16(c.HL())
}

func (c *CPU) PtrSP() uint16 {
	return c.GetPtr16(c.SP)
}

func (c *CPU) SetPtrSP(value uint16) {
	c.SetPtr16(c.SP, value)
}

func displace(addr uint16, disp int16) uint16 {
	return uint16(int16(addr) + disp) // XXX not too sure of this.
}

// jr gets a displacement and jumps relatively
func (c *CPU) jr() {
	disp := c.GetDisp()
	c.IP -= 2
	c.IP = displace(c.IP, disp)
}

// jp jumps absolutely
func (c *CPU) jp(to uint16) {
	c.IP = to
}

func (c *CPU) add(value uint8) {
	// TODO: set flags correctly
	c.A += value
}

func (c *CPU) adc(value uint8) {
	// TODO: set flags correctly
	c.A += value
	if c.F.IsFlag(isa.FlagCarry) {
		c.A++
		c.F.ClearFlag(isa.FlagCarry)
	}
}

func (c *CPU) sub(value uint8) {
	// TODO: set flags correctly
	c.A -= value
}

func (c *CPU) sbc(value uint8) {
	// TODO: set flags correctly
	c.A -= value
	if c.F.IsFlag(isa.FlagCarry) {
		c.A--
		c.F.ClearFlag(isa.FlagCarry)
	}
}

func (c *CPU) and(value uint8) {
	// TODO: set flags correctly
	c.A &= value
}

func (c *CPU) xor(value uint8) {
	// TODO: set flags correctly
	c.A ^= value
}

func (c *CPU) or(value uint8) {
	// TODO: set flags correctly
	c.A |= value
}

func (c *CPU) cmp(value uint8) {
	// TODO: set flags correctly
	res := c.A - value
	if res == 0 {
		c.F.SetFlag(isa.FlagZero)
	} else {
		c.F.ClearFlag(isa.FlagZero)
	}
	if int8(res) < 0 {
		c.F.SetFlag(isa.FlagNegative)
	} else {
		c.F.ClearFlag(isa.FlagNegative)
	}
}

func (c *CPU) pop() uint16 {
	res := c.PtrSP()
	c.SP += 2
	return res
}

func (c *CPU) push(value uint16) {
	c.SP -= 2
	c.SetPtrSP(value)
}

func (c *CPU) ret() {
	c.IP = c.pop()
}

func (c *CPU) call(to uint16) {
	c.push(c.IP + 3)
	c.jp(to)
}

func (c *CPU) rst(to uint16) {
	c.push(c.IP)
	c.jp(to)
}

func (c *CPU) out(port uint8, value uint8) {
	c.IO.Output(port, value)
}

func (c *CPU) in(port uint8) uint8 {
	return c.IO.Input(port)
}

func (c *CPU) RunUntilHalted() error {
	for !c.Halted {
		err := c.Step()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CPU) Step() error {
	if c.UseIX || c.UseIY {
		c.Disp = c.GetDisp()
	}

	if c.PrefixCBBitInstructions || c.PrefixEDMiscInstructions {
		// TODO, prefixed instructions
		return InstructionNotImplemented
	}

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
		c.SetPtrHL(value + 1)
	case isa.DEC_PtrHL:
		value := c.Ptr(c.HL())
		c.SetPtrHL(value - 1)
	case isa.LD_PtrHL_Imm8:
		c.SetPtrHL(c.GetImm8())
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

	case isa.LD_B_B:
		c.B = c.B
	case isa.LD_B_C:
		c.B = c.C
	case isa.LD_B_D:
		c.B = c.D
	case isa.LD_B_E:
		c.B = c.E
	case isa.LD_B_H:
		c.B = c.H
	case isa.LD_B_L:
		c.B = c.L
	case isa.LD_B_PtrHL:
		c.B = c.PtrHL()
	case isa.LD_B_A:
		c.B = c.A

	case isa.LD_C_B:
		c.C = c.B
	case isa.LD_C_C:
		c.C = c.C
	case isa.LD_C_D:
		c.C = c.D
	case isa.LD_C_E:
		c.C = c.E
	case isa.LD_C_H:
		c.C = c.H
	case isa.LD_C_L:
		c.C = c.L
	case isa.LD_C_PtrHL:
		c.C = c.PtrHL()
	case isa.LD_C_A:
		c.C = c.A

	case isa.LD_D_B:
		c.D = c.B
	case isa.LD_D_C:
		c.D = c.C
	case isa.LD_D_D:
		c.D = c.D
	case isa.LD_D_E:
		c.D = c.E
	case isa.LD_D_H:
		c.D = c.H
	case isa.LD_D_L:
		c.D = c.L
	case isa.LD_D_PtrHL:
		c.D = c.PtrHL()
	case isa.LD_D_A:
		c.D = c.A

	case isa.LD_E_B:
		c.E = c.B
	case isa.LD_E_C:
		c.E = c.C
	case isa.LD_E_D:
		c.E = c.D
	case isa.LD_E_E:
		c.E = c.E
	case isa.LD_E_H:
		c.E = c.H
	case isa.LD_E_L:
		c.E = c.L
	case isa.LD_E_PtrHL:
		c.E = c.PtrHL()
	case isa.LD_E_A:
		c.E = c.A

	case isa.LD_H_B:
		c.H = c.B
	case isa.LD_H_C:
		c.H = c.C
	case isa.LD_H_D:
		c.H = c.D
	case isa.LD_H_E:
		c.H = c.E
	case isa.LD_H_H:
		c.H = c.H
	case isa.LD_H_L:
		c.H = c.L
	case isa.LD_H_PtrHL:
		c.H = c.PtrHL()
	case isa.LD_H_A:
		c.H = c.A

	case isa.LD_L_B:
		c.L = c.B
	case isa.LD_L_C:
		c.L = c.C
	case isa.LD_L_D:
		c.L = c.D
	case isa.LD_L_E:
		c.L = c.E
	case isa.LD_L_H:
		c.L = c.H
	case isa.LD_L_L:
		c.L = c.L
	case isa.LD_L_PtrHL:
		c.L = c.Ptr(c.HL())
	case isa.LD_L_A:
		c.L = c.A

	case isa.LD_PtrHL_B:
		c.SetPtrHL(c.B)
	case isa.LD_PtrHL_C:
		c.SetPtrHL(c.C)
	case isa.LD_PtrHL_D:
		c.SetPtrHL(c.D)
	case isa.LD_PtrHL_E:
		c.SetPtrHL(c.E)
	case isa.LD_PtrHL_H:
		c.SetPtrHL(c.H)
	case isa.LD_PtrHL_L:
		c.SetPtrHL(c.L)
	case isa.HALT:
		c.Halted = true
	case isa.LD_PtrHL_A:
		c.SetPtrHL(c.A)

	case isa.LD_A_B:
		c.A = c.B
	case isa.LD_A_C:
		c.A = c.C
	case isa.LD_A_D:
		c.A = c.D
	case isa.LD_A_E:
		c.A = c.E
	case isa.LD_A_H:
		c.A = c.H
	case isa.LD_A_L:
		c.A = c.L
	case isa.LD_A_PtrHL:
		c.A = c.Ptr(c.HL())
	case isa.LD_A_A:
		c.A = c.A

	case isa.ADD_A_B:
		c.add(c.B)
	case isa.ADD_A_C:
		c.add(c.C)
	case isa.ADD_A_D:
		c.add(c.D)
	case isa.ADD_A_E:
		c.add(c.E)
	case isa.ADD_A_H:
		c.add(c.H)
	case isa.ADD_A_L:
		c.add(c.L)
	case isa.ADD_A_PtrHL:
		c.add(c.PtrHL())
	case isa.ADD_A_A:
		c.add(c.A)

	case isa.ADC_A_B:
		c.adc(c.B)
	case isa.ADC_A_C:
		c.adc(c.C)
	case isa.ADC_A_D:
		c.adc(c.D)
	case isa.ADC_A_E:
		c.adc(c.E)
	case isa.ADC_A_H:
		c.adc(c.H)
	case isa.ADC_A_L:
		c.adc(c.L)
	case isa.ADC_A_PtrHL:
		c.adc(c.PtrHL())
	case isa.ADC_A_A:
		c.adc(c.A)

	case isa.SUB_A_B:
		c.sub(c.B)
	case isa.SUB_A_C:
		c.sub(c.C)
	case isa.SUB_A_D:
		c.sub(c.D)
	case isa.SUB_A_E:
		c.sub(c.E)
	case isa.SUB_A_H:
		c.sub(c.H)
	case isa.SUB_A_L:
		c.sub(c.L)
	case isa.SUB_A_PtrHL:
		c.sub(c.PtrHL())
	case isa.SUB_A_A:
		c.sub(c.A)

	case isa.SBC_A_B:
		c.sbc(c.B)
	case isa.SBC_A_C:
		c.sbc(c.C)
	case isa.SBC_A_D:
		c.sbc(c.D)
	case isa.SBC_A_E:
		c.sbc(c.E)
	case isa.SBC_A_H:
		c.sbc(c.H)
	case isa.SBC_A_L:
		c.sbc(c.L)
	case isa.SBC_A_PtrHL:
		c.sbc(c.PtrHL())
	case isa.SBC_A_A:
		c.sbc(c.A)

	case isa.AND_A_B:
		c.and(c.B)
	case isa.AND_A_C:
		c.and(c.C)
	case isa.AND_A_D:
		c.and(c.D)
	case isa.AND_A_E:
		c.and(c.E)
	case isa.AND_A_H:
		c.and(c.H)
	case isa.AND_A_L:
		c.and(c.L)
	case isa.AND_A_PtrHL:
		c.and(c.PtrHL())
	case isa.AND_A_A:
		c.and(c.A)

	case isa.XOR_A_B:
		c.xor(c.B)
	case isa.XOR_A_C:
		c.xor(c.C)
	case isa.XOR_A_D:
		c.xor(c.D)
	case isa.XOR_A_E:
		c.xor(c.E)
	case isa.XOR_A_H:
		c.xor(c.H)
	case isa.XOR_A_L:
		c.xor(c.L)
	case isa.XOR_A_PtrHL:
		c.xor(c.PtrHL())
	case isa.XOR_A_A:
		c.xor(c.A)

	case isa.OR_A_B:
		c.or(c.B)
	case isa.OR_A_C:
		c.or(c.C)
	case isa.OR_A_D:
		c.or(c.D)
	case isa.OR_A_E:
		c.or(c.E)
	case isa.OR_A_H:
		c.or(c.H)
	case isa.OR_A_L:
		c.or(c.L)
	case isa.OR_A_PtrHL:
		c.or(c.PtrHL())
	case isa.OR_A_A:
		c.or(c.A)

	case isa.CP_A_B:
		c.cmp(c.B)
	case isa.CP_A_C:
		c.cmp(c.C)
	case isa.CP_A_D:
		c.cmp(c.D)
	case isa.CP_A_E:
		c.cmp(c.E)
	case isa.CP_A_H:
		c.cmp(c.H)
	case isa.CP_A_L:
		c.cmp(c.L)
	case isa.CP_A_PtrHL:
		c.cmp(c.PtrHL())
	case isa.CP_A_A:
		c.cmp(c.A)

	case isa.RETNZ:
		if !c.F.IsFlag(isa.FlagZero) {
			c.ret()
		}
	case isa.POP_BC:
		c.SetBC(c.pop())
	case isa.JPNZ_Imm16:
		if !c.F.IsFlag(isa.FlagZero) {
			c.jp(c.GetImm16())
		}
	case isa.JP_Imm16:
		c.jp(c.GetImm16())
	case isa.CALLNZ_Imm16:
		if !c.F.IsFlag(isa.FlagZero) {
			c.call(c.GetImm16())
		}

	case isa.PUSH_BC:
		c.push(c.BC())
	case isa.ADD_A_Imm8:
		c.add(c.GetImm8())
	case isa.RST_0x00:
		c.rst(0x00)

	case isa.RETZ:
		if c.F.IsFlag(isa.FlagZero) {
			c.ret()
		}

	case isa.RET:
		c.ret()
	case isa.JPZ_Imm16:
		if c.F.IsFlag(isa.FlagZero) {
			c.jp(c.GetImm16())
		}

	case isa.CB_Prefix:
		c.PrefixCBBitInstructions = true
	case isa.CALLZ_Imm16:
		if c.F.IsFlag(isa.FlagZero) {
			c.call(c.GetImm16())
		}
	case isa.CALL_Imm16:
		c.call(c.GetImm16())
	case isa.ADC_A_Imm8:
		c.adc(c.GetImm8())
	case isa.RST_0x08:
		c.rst(0x08)

	case isa.RETNC:
		if !c.F.IsFlag(isa.FlagCarry) {
			c.ret()
		}
	case isa.POP_DE:
		c.SetDE(c.pop())
	case isa.JPNC_Imm16:
		if !c.F.IsFlag(isa.FlagCarry) {
			c.jp(c.GetImm16())
		}
	case isa.OUT_Port_A:
		c.out(c.GetImm8(), c.A)
	case isa.CALLNC_Imm16:
		if !c.F.IsFlag(isa.FlagCarry) {
			c.call(c.GetImm16())
		}

	case isa.PUSH_DE:
		c.push(c.DE())
	case isa.SUB_A_Imm8:
		c.sub(c.GetImm8())
	case isa.RST_0x10:
		c.rst(0x10)

	case isa.RETC:
		if c.F.IsFlag(isa.FlagCarry) {
			c.ret()
		}

	case isa.EXX:
		c.Registers, c.Shadow = c.Shadow, c.Registers
	case isa.JPC_Imm16:
		if c.F.IsFlag(isa.FlagCarry) {
			c.jp(c.GetImm16())
		}

	case isa.IN_A_Port:
		c.A = c.in(c.GetImm8())
	case isa.CALLC_Imm16:
		if c.F.IsFlag(isa.FlagCarry) {
			c.call(c.GetImm16())
		}
	case isa.DD_Prefix:
		c.UseIX = true
	case isa.SBC_A_Imm8:
		c.sbc(c.GetImm8())
	case isa.RST_0x18:
		c.rst(0x18)

	case isa.RETPO:
		if !c.F.IsFlag(isa.FlagParity) {
			c.ret()
		}
	case isa.POP_HL:
		c.SetHL(c.pop())
	case isa.JPPO_Imm16:
		if !c.F.IsFlag(isa.FlagParity) {
			c.jp(c.GetImm16())
		}
	case isa.EX_PtrSP_HL:
		sp := c.PtrSP()
		hl := c.HL()
		c.SetHL(sp)
		c.SetPtrSP(hl)
	case isa.CALLPO_Imm16:
		if !c.F.IsFlag(isa.FlagParity) {
			c.call(c.GetImm16())
		}

	case isa.PUSH_HL:
		c.push(c.HL())
	case isa.AND_A_Imm8:
		c.and(c.GetImm8())
	case isa.RST_0x20:
		c.rst(0x20)

	case isa.RETPE:
		if c.F.IsFlag(isa.FlagParity) {
			c.ret()
		}

	case isa.JP_PtrHL:
		c.jp(c.PtrHL16())
	case isa.JPPE_Imm16:
		if c.F.IsFlag(isa.FlagParity) {
			c.jp(c.GetImm16())
		}

	case isa.EX_DE_HL:
		de := c.DE()
		hl := c.HL()
		c.SetHL(de)
		c.SetDE(hl)
	case isa.CALLPE_Imm16:
		if c.F.IsFlag(isa.FlagParity) {
			c.call(c.GetImm16())
		}
	case isa.ED_Prefix:
		c.PrefixEDMiscInstructions = true
	case isa.XOR_A_Imm8:
		c.xor(c.GetImm8())
	case isa.RST_0x28:
		c.rst(0x28)

	case isa.RETP:
		if !c.F.IsFlag(isa.FlagNegative) {
			c.ret()
		}
	case isa.POP_AF:
		c.SetAF(c.pop())
	case isa.JPP_Imm16:
		if !c.F.IsFlag(isa.FlagNegative) {
			c.jp(c.GetImm16())
		}
	case isa.DI:
		c.IFF1 = false
		c.IFF2 = false
	case isa.CALLP_Imm16:
		if !c.F.IsFlag(isa.FlagNegative) {
			c.call(c.GetImm16())
		}

	case isa.PUSH_AF:
		c.push(c.AF())
	case isa.OR_A_Imm8:
		c.or(c.GetImm8())
	case isa.RST_0x30:
		c.rst(0x30)

	case isa.RETM:
		if c.F.IsFlag(isa.FlagNegative) {
			c.ret()
		}

	case isa.LD_SP_HL:
		c.SP = c.HL()
	case isa.JPM_Imm16:
		if c.F.IsFlag(isa.FlagNegative) {
			c.jp(c.GetImm16())
		}

	case isa.EI:
		c.IFF1 = true
		c.IFF2 = false
	case isa.CALLM_Imm16:
		if c.F.IsFlag(isa.FlagNegative) {
			c.call(c.GetImm16())
		}
	case isa.FD_Prefix:
		c.UseIY = true
	case isa.CP_A_Imm8:
		c.cmp(c.GetImm8())
	case isa.RST_0x38:
		c.rst(0x38)

	default:
		return InstructionNotImplemented
	}
	return nil
}

func (c *CPU) bitInstruction(opcode isa.Opcode) error {
	x, y, z := opcode.SplitBitOpcode()
	if c.UseIY || c.UseIX {
		c.Disp = c.GetDisp()
	}

	switch x {
	case isa.BitOpcodeKindShift:
		return c.bitInstructionShift(y, z)
	case isa.BitOpcodeKindTest:
		return c.bitInstructionTest(uint8(y), z)
	case isa.BitOpcodeKindClear:
		return c.bitInstructionClear(uint8(y), z)
	case isa.BitOpcodeKindSet:
		return c.bitInstructionSet(uint8(y), z)
	default:
		return InstructionNotCorrect
	}
}

func (c *CPU) RegB() uint8 { return c.B }
func (c *CPU) RegC() uint8 { return c.C }
func (c *CPU) RegD() uint8 { return c.D }
func (c *CPU) RegE() uint8 { return c.E }
func (c *CPU) RegH() uint8 { return c.H }
func (c *CPU) RegL() uint8 { return c.L }
func (c *CPU) RegPtrIX() uint8 {
	disp := c.Disp
	c.Disp = 0
	c.UseIX = false
	return c.Ptr(displace(c.IX, disp))
}

func (c *CPU) RegPtrIY() uint8 {
	disp := c.Disp
	c.Disp = 0
	c.UseIY = false
	return c.Ptr(displace(c.IY, disp))
}

func (c *CPU) RegPtrHL() uint8 {
	if c.UseIX {
		return c.RegPtrIX()
	}
	if c.UseIY {
		return c.RegPtrIY()
	}

	return c.PtrHL()
}
func (c *CPU) RegA() uint8 { return c.A }

func (c *CPU) Reg(reg isa.RegisterIndex) func() uint8 {
	switch reg {
	case isa.RegisterIndexB:
		return c.RegB
	case isa.RegisterIndexC:
		return c.RegC
	case isa.RegisterIndexD:
		return c.RegD
	case isa.RegisterIndexE:
		return c.RegE
	case isa.RegisterIndexH:
		return c.RegH
	case isa.RegisterIndexL:
		return c.RegL
	case isa.RegisterIndexPtrHL:
		return c.RegPtrHL
	case isa.RegisterIndexA:
		return c.RegA
	default:
		return nil
	}
}

func (c *CPU) SetRegB(v uint8) { c.B = v }
func (c *CPU) SetRegC(v uint8) { c.C = v }
func (c *CPU) SetRegD(v uint8) { c.D = v }
func (c *CPU) SetRegE(v uint8) { c.E = v }
func (c *CPU) SetRegH(v uint8) { c.H = v }
func (c *CPU) SetRegL(v uint8) { c.L = v }
func (c *CPU) SetRegPtrIX(v uint8) {
	disp := c.Disp
	c.Disp = 0
	c.UseIX = false
	c.SetPtr(displace(c.IX, disp), v)
}

func (c *CPU) SetRegPtrIY(v uint8) {
	disp := c.Disp
	c.Disp = 0
	c.UseIY = false
	c.SetPtr(displace(c.IY, disp), v)
}

func (c *CPU) SetRegPtrHL(v uint8) {
	if c.UseIX {
		c.SetRegPtrIX(v)
		return
	}
	if c.UseIY {
		c.SetRegPtrIY(v)
		return
	}

	c.SetPtrHL(v)
}

func (c *CPU) SetRegA(v uint8) { c.A = v }

func (c *CPU) SetReg(reg isa.RegisterIndex) func(uint8) {
	switch reg {
	case isa.RegisterIndexB:
		return c.SetRegB
	case isa.RegisterIndexC:
		return c.SetRegC
	case isa.RegisterIndexD:
		return c.SetRegD
	case isa.RegisterIndexE:
		return c.SetRegE
	case isa.RegisterIndexH:
		return c.SetRegH
	case isa.RegisterIndexL:
		return c.SetRegL
	case isa.RegisterIndexPtrHL:
		return c.SetRegPtrHL
	case isa.RegisterIndexA:
		return c.SetRegA
	default:
		return nil
	}
}

func (c *CPU) bitInstructionShift(bit isa.BitOpcodeBit, reg isa.RegisterIndex) error {
	return InstructionNotImplemented
}

func (c *CPU) bitInstructionTest(bit uint8, reg isa.RegisterIndex) error {
	getter := c.Reg(reg)
	if getter == nil {
		return InstructionNotCorrect
	}
	flag := uint8(1 << bit)
	isSet := (getter() & flag) == flag
	c.F.SetFlag(isa.FlagHalfCarry)
	c.F.ClearFlag(isa.FlagNegative)
	if isSet {
		c.F.ClearFlag(isa.FlagZero)
	} else {
		c.F.SetFlag(isa.FlagZero)
	}
	return nil
}

func (c *CPU) bitInstructionSet(bit uint8, reg isa.RegisterIndex) error {
	getter := c.Reg(reg)
	if getter == nil {
		return InstructionNotCorrect
	}
	setter := c.SetReg(reg)
	if setter == nil {
		return InstructionNotCorrect
	}
	flag := uint8(1 << bit)
	v := (getter() | flag)
	setter(v)

	return nil
}

func (c *CPU) bitInstructionClear(bit uint8, reg isa.RegisterIndex) error {
	getter := c.Reg(reg)
	if getter == nil {
		return InstructionNotCorrect
	}
	setter := c.SetReg(reg)
	if setter == nil {
		return InstructionNotCorrect
	}
	flag := uint8(1 << bit)
	v := (getter() & ^flag)
	setter(v)

	return nil
}
