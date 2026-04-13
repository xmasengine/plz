package emu

import "github.com/xmasengine/plz/pkg/z80/isa"

import "testing"

func TestNewCPU(t *testing.T) {
	cpu := NewCPU()
	if cpu.Interrupt == nil {
		t.Fatalf("Interrupt is nil")
	}
	if cpu.NMI == nil {
		t.Fatalf("NMI is nil")
	}
	if cpu.Clock == nil {
		t.Fatalf("Clock is nil")
	}
	if lm, ok := cpu.Memory.(*LinearMemory); !ok || lm == nil {
		t.Fatalf("Memory is nil or not linear memory")
	}

	if io, ok := cpu.IO.(*ByteIO); !ok || io == nil {
		t.Fatalf("io is nil or not byte io")
	}
}

func helperTestProgram(t *testing.T, inPort, outPort int, in, expected string, op ...isa.Opcode) {
	t.Helper()
	prog1 := Program(isa.HALT)
	cpu := NewCPU(prog1)
	io := cpu.IO.(*ByteIO)
	io.In[inPort] = []byte(in)

	err := cpu.RunUntilHalted()
	if err != nil {
		t.Fatalf("Run failed: %s", err)
	}
	observed := string(io.Out[outPort])
	if observed != expected {
		t.Fatalf("Output not correct, expected %s, observed %s", expected, observed)
	}
}

func TestEmuRunUntilHalted(t *testing.T) {
	prog1 := Program(isa.HALT)
	cpu := NewCPU(prog1)
	err := cpu.RunUntilHalted()
	if err != nil {
		t.Fatalf("Run failed: %s", err)
	}
	helperTestProgram(t, 0, 0, "", "", isa.HALT)
	helperTestProgram(t, 0, 0, "", "H", isa.LD_A_Imm8, isa.Opcode('H'), isa.OUT_Port_A, isa.Opcode(0))
}
