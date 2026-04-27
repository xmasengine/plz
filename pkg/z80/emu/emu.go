package emu

import "io"
import "os"
import "context"

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

type ReaderWriterIO struct {
	Index   int
	Readers [255]io.Reader
	Writers [255]io.Writer
}

func (b *ReaderWriterIO) In(port byte) byte {
	rd := b.Readers[port]
	if rd == nil {
		return 0
	}
	buf := []byte{0}
	rd.Read(buf[:])
	return buf[0]
}

func (b *ReaderWriterIO) Out(port byte, val byte) {
	wr := b.Writers[port]
	if wr == nil {
		return
	}
	buf := []byte{val}
	wr.Write(buf[:])
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

func WithReaderWriterIO(c *CPU) {
	c.IO = &ReaderWriterIO{}
}

func WithReader(c *CPU, port byte, rd io.Reader) {
	rwi := c.IO.(*ReaderWriterIO)
	rwi.Readers[port] = rd
}

func WithBinary(bin ...byte) func(*CPU) {
	return func(c *CPU) {
		for i, b := range bin {
			c.Memory.Set(uint16(i), b)
		}
	}
}

func RunFile(ctx context.Context, name string, opts ...cpuOption) error {
	buf, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	opts = append(opts, WithBinary(buf...))
	cpu := NewCPU(opts...)
	return cpu.Run(ctx)
}
