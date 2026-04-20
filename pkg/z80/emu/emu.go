package emu

import "github.com/koron-go/z80"
import "github.com/xmasengine/plz/pkg/z80/isa"

type CPU = z80.CPU

type LinearMemory [1 << 16]uint8

func (l LinearMemory) Get(addr uint16) byte {
	return l[addr]
}

func (l *LinearMemory) Set(addr uint16, v byte) {
	l[addr] = v
}

type ByteIO struct {
	Index    int
	InBytes  [255][]byte
	OutBytes [255][]byte
}

func (b *ByteIO) In(port byte) byte {
	if b.Index >= len(b.InBytes[port]) {
		return 0
	}
	res := b.InBytes[port][b.Index]
	b.Index++
	return res
}

func (b *ByteIO) Out(port byte, val byte) {
	b.OutBytes[port] = append(b.OutBytes[port], val)
}

func Opcodes(ops ...isa.Opcode) func(*CPU) {
	return func(c *CPU) {
		for i, op := range ops {
			c.Memory.Set(uint16(i), byte(op))
		}
	}
}

func Instructions(ins ...isa.Instruction) func(*CPU) {
	return func(c *CPU) {
		addr := uint16(0)
		for _, in := range ins {
			by := in.Bytes()
			for o, b := range by {
				c.Memory.Set(addr+uint16(o), b)
			}
			addr += uint16(len(by))
		}
	}
}

type cpuOption func(c *CPU)

func NewCPU(opts ...cpuOption) *CPU {
	cpu := &CPU{}
	cpu.Memory = &LinearMemory{}
	cpu.IO = &ByteIO{}
	for _, opt := range opts {
		opt(cpu)
	}
	return cpu
}
