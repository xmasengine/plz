package sms_test

import (
	"context"
	"testing"

	"github.com/xmasengine/plz/pkg/sms"
	"github.com/xmasengine/plz/pkg/z80/emu"
)

func TestControlPortRegisterWrite(t *testing.T) {
	v := sms.New(false)
	v.WriteControl(0x04)
	v.WriteControl(0x80 | 0x00)
	if v.Reg(0) != 0x04 {
		t.Fatalf("reg[0] = %#02x, want 0x04", v.Reg(0))
	}
}

func TestControlPortVRAMAddr(t *testing.T) {
	v := sms.New(false)
	v.WriteControl(0x34)
	v.WriteControl(0x41)
	if v.AddrReg() != 0x0134 {
		t.Fatalf("addrReg = %#04x, want 0x0134", v.AddrReg())
	}
	if v.CodeReg() != 1 {
		t.Fatalf("codeReg = %d, want 1", v.CodeReg())
	}
}

func TestControlPortCRAMAddr(t *testing.T) {
	v := sms.New(false)
	v.WriteControl(0x00)
	v.WriteControl(0xC0)
	if v.CodeReg() != 3 {
		t.Fatalf("codeReg = %d, want 3", v.CodeReg())
	}
}

func TestWriteVRAM(t *testing.T) {
	v := sms.New(false)
	v.WriteControl(0x00)
	v.WriteControl(0x40)
	v.WriteData(0xAB)
	if v.VRAMAt(0) != 0xAB {
		t.Fatalf("VRAM[0] = %#02x, want 0xAB", v.VRAMAt(0))
	}
}

func TestWriteCRAM(t *testing.T) {
	v := sms.New(false)
	v.WriteControl(0x00)
	v.WriteControl(0xC0)
	v.WriteData(0x2B)
	if v.CRAMAt(0) != 0x2B {
		t.Fatalf("CRAM[0] = %#02x, want 0x2B", v.CRAMAt(0))
	}
}

func TestWriteReadVRAM(t *testing.T) {
	v := sms.New(false)

	v.WriteControl(0x10)
	v.WriteControl(0x40)
	v.WriteData(0xAA)

	v.WriteControl(0x10)
	v.WriteControl(0x00)
	val := v.ReadData()
	_ = val

	v.WriteControl(0x10)
	v.WriteControl(0x00)
	val = v.ReadData()
	if val != 0xAA {
		t.Fatalf("read = %#02x, want 0xAA", val)
	}
}

func TestFrameBufferAfterTick(t *testing.T) {
	v := sms.New(false)
	total := uint32(sms.LinesNTSC) * sms.HblankCycles
	v.Tick(total)
	if !v.FrameReady() {
		t.Fatal("frame should be ready after NTSC frame")
	}
	fb := v.Framebuffer()
	if len(fb) == 0 {
		t.Fatal("framebuffer should not be empty")
	}
}

func TestMultipleFrames(t *testing.T) {
	v := sms.New(false)
	for i := 0; i < 10; i++ {
		v.Tick(sms.LinesNTSC * sms.HblankCycles)
	}
}

func TestSMSIORouting(t *testing.T) {
	v := sms.New(false)

	io := &emu.SMSIO{VDP: v}

	io.Out(0xBF, 0x00)
	io.Out(0xBF, 0x40)
	io.Out(0xBE, 0x42)
	if v.VRAMAt(0) != 0x42 {
		t.Fatalf("VRAM[0] = %#02x, want 0x42", v.VRAMAt(0))
	}

	_ = context.Background()
}

func TestBasicFrameRender(t *testing.T) {
	v := sms.New(false)

	v.WriteControl(0x04)
	v.WriteControl(0x80 | 0x00)

	v.WriteControl(0x0E)
	v.WriteControl(0x80 | 0x02)

	v.WriteControl(0x00)
	v.WriteControl(0x40)
	for i := 0; i < 32; i++ {
		v.WriteData(0xFF)
	}

	v.WriteControl(0x00)
	v.WriteControl(0x40 | ((0x3800 >> 8) & 0x3F))
	v.WriteData(0x00)
	v.WriteData(0x08)

	v.WriteControl(0x00)
	v.WriteControl(0xC0)
	v.WriteData(0x3F)

	v.Tick(sms.LinesNTSC * sms.HblankCycles)

	if !v.FrameReady() {
		t.Fatal("frame should be ready")
	}
	fb := v.Framebuffer()
	if len(fb) < sms.ScreenWidth*4 {
		t.Fatal("framebuffer too small")
	}
}

func TestSMSSystem(t *testing.T) {
	v := sms.New(false)
	cpu := emu.NewCPU(emu.WithVDP(v))

	cpu.IO.Out(0xBF, 0x04)
	cpu.IO.Out(0xBF, 0x80|0x00)
	if v.Reg(0) != 0x04 {
		t.Fatalf("reg[0] = %#02x, want 0x04", v.Reg(0))
	}

	cpu.IO.Out(0xBF, 0x0E)
	cpu.IO.Out(0xBF, 0x80|0x02)

	cpu.IO.Out(0xBF, 0x00)
	cpu.IO.Out(0xBF, 0x40)
	for i := 0; i < 32; i++ {
		cpu.IO.Out(0xBE, 0xFF)
	}

	cpu.IO.Out(0xBF, 0x00)
	cpu.IO.Out(0xBF, 0x40|((0x3800>>8)&0x3F))
	cpu.IO.Out(0xBE, 0x00)
	cpu.IO.Out(0xBE, 0x08)

	cpu.IO.Out(0xBF, 0x00)
	cpu.IO.Out(0xBF, 0xC0)
	v.WriteData(0x3F)

	v.Tick(sms.LinesNTSC * sms.HblankCycles)

	if !v.FrameReady() {
		t.Fatal("frame should be ready")
	}
}
