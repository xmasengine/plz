package emu

import "testing"
import "context"
import "time"

// Opcode constants for assembling test programs in memory.
const (
	opHALT       byte = 0x76
	opNOP        byte = 0x00
	opLD_A_Imm8  byte = 0x3E // LD A, n
	opOUT_Port_A byte = 0xD3 // OUT (n), A
	opJR_Disp    byte = 0x18 // JR d
)

func TestNewCPU(t *testing.T) {
	cpu := NewCPU()
	if lm, ok := cpu.Memory.(*LinearMemory); !ok || lm == nil {
		t.Fatalf("Memory is nil or not linear memory")
	}

	if io, ok := cpu.IO.(*ByteIO); !ok || io == nil {
		t.Fatalf("io is nil or not byte io")
	}
}

func helperTestOpcodes(t *testing.T, inPort, outPort int, in, expected string, op ...byte) {
	t.Helper()
	program := Memory(op...)
	cpu := NewCPU(program)
	io := cpu.IO.(*ByteIO)
	io.InBytes[inPort] = []byte(in)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second*10)
	defer cancel()
	err := cpu.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %s", err)
	}
	observed := string(io.OutBytes[outPort])
	if expected != observed {
		t.Fatalf("Output not correct, expected >%s<, observed >%s<: >%v<, >%v<", expected, observed, []byte(expected), []byte(observed))
	}
}

func TestEmuRunUntilHalted(t *testing.T) {
	helperTestOpcodes(t, 0, 0, "", "", opHALT)
	// The traditional greeting. We expect HELLO WORLD in the output.
	helperTestOpcodes(t, 0, 7, "", "HELLO WORLD",
		opLD_A_Imm8, byte('H'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('E'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('L'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('L'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('O'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte(' '), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('W'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('O'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('R'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('L'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('D'), opOUT_Port_A, byte(7),
		opHALT)
	helperTestOpcodes(t, 0, 7, "", "WORLD",
		opJR_Disp, byte(4*6),
		opLD_A_Imm8, byte('H'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('E'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('L'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('L'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('O'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte(' '), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('W'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('O'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('R'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('L'), opOUT_Port_A, byte(7),
		opLD_A_Imm8, byte('D'), opOUT_Port_A, byte(7),
		opHALT)
	helperTestOpcodes(t, 0, 0, "", "", opNOP, opHALT)
}
