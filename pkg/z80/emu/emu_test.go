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

func helperTestOpcodes(t *testing.T, inPort, outPort int, in, expected string, op ...isa.Opcode) {
	t.Helper()
	program := Opcodes(op...)
	cpu := NewCPU(program)
	io := cpu.IO.(*ByteIO)
	io.In[inPort] = []byte(in)
	cpu.StepsUntilError = 10000

	err := cpu.RunUntilHalted()
	if err != nil {
		t.Fatalf("Run failed: %s", err)
	}
	observed := string(io.Out[outPort])
	if expected != observed {
		t.Fatalf("Output not correct, expected >%s<, observed >%s<: >%v<, >%v<", expected, observed, []byte(expected), []byte(observed))
	}
}

func TestEmuRunUntilHalted(t *testing.T) {
	helperTestOpcodes(t, 0, 0, "", "", isa.HALT)
	// The traditional greeting. We expect HELLO WORLD in the output.
	helperTestOpcodes(t, 0, 7, "", "HELLO WORLD",
		isa.LD_A_Imm8, isa.Opcode('H'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('E'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('L'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('L'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('O'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode(' '), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('W'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('O'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('R'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('L'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('D'), isa.OUT_Port_A, isa.Opcode(7),
		isa.HALT)
	helperTestOpcodes(t, 0, 7, "", "WORLD",
		isa.JR_Disp, isa.Opcode(4*6+2),
		isa.LD_A_Imm8, isa.Opcode('H'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('E'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('L'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('L'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('O'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode(' '), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('W'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('O'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('R'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('L'), isa.OUT_Port_A, isa.Opcode(7),
		isa.LD_A_Imm8, isa.Opcode('D'), isa.OUT_Port_A, isa.Opcode(7),
		isa.HALT)
	helperTestOpcodes(t, 0, 0, "", "", isa.NOP, isa.HALT)
}
