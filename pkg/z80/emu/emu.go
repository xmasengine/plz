package emu

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/koron-go/z80"
	"github.com/xmasengine/plz/pkg/sms"
)

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

type SMSIO struct {
	ByteIO
	VDP *sms.VDP
}

func (io *SMSIO) In(port byte) byte {
	switch {
	case port >= 0x80 && port < 0xC0:
		if port&1 == 0 {
			return io.VDP.ReadData()
		}
		return io.VDP.ReadControl()
	case port >= 0x40 && port < 0x80:
		if port&1 == 0 {
			return io.VDP.VCounter()
		}
		return io.VDP.HCounter()
	default:
		return io.ByteIO.In(port)
	}
}

func (io *SMSIO) Out(port byte, val byte) {
	switch {
	case port >= 0x80 && port < 0xC0:
		if port&1 == 0 {
			io.VDP.WriteData(val)
		} else {
			io.VDP.WriteControl(val)
		}
	case port >= 0x40 && port < 0x80:
	default:
		io.ByteIO.Out(port, val)
	}
}

type ReaderWriterIO struct {
	Index   int
	Readers [255]io.Reader
	Writers [255]io.Writer
	Errors  []error
}

func (b *ReaderWriterIO) In(port byte) byte {
	rd := b.Readers[port]
	if rd == nil {
		return 0
	}
	buf := []byte{0}
	_, err := rd.Read(buf[:])
	if err != nil {
		b.Errors = append(b.Errors, fmt.Errorf("port %d read: %w", port, err))
	}
	return buf[0]
}

func (b *ReaderWriterIO) Out(port byte, val byte) {
	wr := b.Writers[port]
	if wr == nil {
		return
	}
	buf := []byte{val}
	_, err := wr.Write(buf[:])
	if err != nil {
		b.Errors = append(b.Errors, fmt.Errorf("port %d write: %w", port, err))
	}
}

func Memory(buf ...byte) func(*CPU) {
	return func(c *CPU) {
		for i, b := range buf {
			c.Memory.Set(uint16(i), b)
		}
	}
}

type CPUOption func(c *CPU)

func NewCPU(opts ...CPUOption) *CPU {
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

func WithReader(port byte, rd io.Reader) func(*CPU) {
	return func(c *CPU) {
		rwi := c.IO.(*ReaderWriterIO)
		rwi.Readers[port] = rd
	}
}

func WithWriter(port byte, wr io.Writer) func(*CPU) {
	return func(c *CPU) {
		rwi := c.IO.(*ReaderWriterIO)
		rwi.Writers[port] = wr
	}
}

func WithVDP(v *sms.VDP) func(*CPU) {
	return func(c *CPU) {
		c.IO = &SMSIO{ByteIO: ByteIO{}, VDP: v}
	}
}

func RunSMS(ctx context.Context, cpu *CPU, v *sms.VDP) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if cpu.HALT {
			// Interrupts disabled → HALT is permanent; exit.
			if !cpu.IFF1 {
				return nil
			}
			// Interrupts enabled — tick VDP until one fires.
			if v.IntPending() {
				cpu.Interrupt = z80.IM1Interrupt()
				cpu.Step()
				cpu.HALT = false
			} else {
				v.Tick(4)
			}
			continue
		}

		cpu.Interrupt = nil
		if v.IntPending() {
			cpu.Interrupt = z80.IM1Interrupt()
		}

		cpu.Step()
		v.Tick(4)
	}
}

func WithBinary(bin ...byte) func(*CPU) {
	return func(c *CPU) {
		for i, b := range bin {
			c.Memory.Set(uint16(i), b)
		}
	}
}

func RunFile(ctx context.Context, name string, opts ...CPUOption) error {
	buf, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	opts = append(opts, WithBinary(buf...))
	cpu := NewCPU(opts...)
	return cpu.Run(ctx)
}

func RunSMSFile(ctx context.Context, name string, v *sms.VDP, opts ...CPUOption) error {
	buf, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	opts = append(opts, WithBinary(buf...), WithVDP(v))
	cpu := NewCPU(opts...)
	return RunSMS(ctx, cpu, v)
}
