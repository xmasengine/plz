package asm

import "github.com/xmasengine/plz/pkg/z80/emu"

import "testing"
import "strings"

func helperTestProgram(t *testing.T, inPort, outPort int, in, expected string, program string) {
	t.Helper()
	rd := strings.NewReader(program)
	opcodes := AssembleBinary(rd)
	t.Logf("opcodes: %v", opcodes)

	cpu := emu.NewCPU(emu.Opcodes(opcodes...))
	io := cpu.IO.(*emu.ByteIO)
	io.In[inPort] = []byte(in)
	cpu.StepsUntilError = 10000

	err := cpu.RunUntilHalted()
	if err != nil {
		t.Fatalf("Run failed: %s", err)
	}
	observed := string(io.Out[outPort])
	t.Logf("output: %d: %s", outPort, observed)
	if expected != observed {
		t.Fatalf("Output not correct, expected >%s<, observed >%s<: >%v<, >%v<", expected, observed, []byte(expected), []byte(observed))
	}
}

func TestEmuRunUntilHalted(t *testing.T) {
	helperTestProgram(t, 0, 0, "", "", `HALT`)
	// The traditional greeting. We expect HELLO WORLD in the output.
	helperTestProgram(t, 0, 7, "", "HELLO WORLD",
		`LD_A_Imm8 'H' OUT_Port_A 7
		LD_A_Imm8 'E' OUT_Port_A 7
		LD_A_Imm8 'L' OUT_Port_A 7
		LD_A_Imm8 'L' OUT_Port_A 7
		LD_A_Imm8 'O' OUT_Port_A 7
		LD_A_Imm8 ' ' OUT_Port_A 7
		LD_A_Imm8 'W' OUT_Port_A 7
		LD_A_Imm8 'O' OUT_Port_A 7
		LD_A_Imm8 'R' OUT_Port_A 7
		LD_A_Imm8 'L' OUT_Port_A 7
		LD_A_Imm8 'D' OUT_Port_A 7
		HALT
	`)

	helperTestProgram(t, 0, 7, "", "WORLD",
		`JR_Disp 26
		LD_A_Imm8 'H' OUT_Port_A 7
		LD_A_Imm8 'E' OUT_Port_A 7
		LD_A_Imm8 'L' OUT_Port_A 7
		LD_A_Imm8 'L' OUT_Port_A 7
		LD_A_Imm8 'O' OUT_Port_A 7
		LD_A_Imm8 ' ' OUT_Port_A 7
		LD_A_Imm8 'W' OUT_Port_A 7
		LD_A_Imm8 'O' OUT_Port_A 7
		LD_A_Imm8 'R' OUT_Port_A 7
		LD_A_Imm8 'L' OUT_Port_A 7
		LD_A_Imm8 'D' OUT_Port_A 7
		HALT
	`)
	helperTestProgram(t, 0, 7, "", "WORLD",
		`JP_Imm16 skip
		LD_A_Imm8 'H' OUT_Port_A 7
		LD_A_Imm8 'E' OUT_Port_A 7
		LD_A_Imm8 'L' OUT_Port_A 7
		LD_A_Imm8 'L' OUT_Port_A 7
		LD_A_Imm8 'O' OUT_Port_A 7
		LD_A_Imm8 ' ' OUT_Port_A 7
		:skip
		LD_A_Imm8 'W' OUT_Port_A 7
		LD_A_Imm8 'O' OUT_Port_A 7
		LD_A_Imm8 'R' OUT_Port_A 7
		LD_A_Imm8 'L' OUT_Port_A 7
		LD_A_Imm8 'D' OUT_Port_A 7
		HALT
	`)
	helperTestProgram(t, 0, 7, "", "HELLO",
		`JP_Imm16 start
		:hello
		LD_A_Imm8 'H' OUT_Port_A 7
		LD_A_Imm8 'E' OUT_Port_A 7
		LD_A_Imm8 'L' OUT_Port_A 7
		LD_A_Imm8 'L' OUT_Port_A 7
		LD_A_Imm8 'O' OUT_Port_A 7
		RET
		:start
		CALL_Imm16 hello
		HALT
	`)
}
