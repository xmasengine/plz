package asm

import "github.com/xmasengine/plz/pkg/z80/emu"

import "testing"
import "strings"
import "context"
import "time"

func helperTestProgram(t *testing.T, inPort, outPort int, in, expected string, program string) {
	t.Helper()
	rd := strings.NewReader(program)
	opcodes, err := AssembleBinary(rd)
	t.Logf("opcodes: %v", opcodes)
	if err != nil {
		if err.Error() != expected {
			t.Fatalf("unexpected error: %s expecetd %s", err, expected)
		}
		return
	}

	cpu := emu.NewCPU(emu.Opcodes(opcodes...))
	io := cpu.IO.(*emu.ByteIO)
	io.InBytes[inPort] = []byte(in)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second*10)
	defer cancel()

	err = cpu.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %s", err)
	}
	observed := string(io.OutBytes[outPort])
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
		`JR_Disp 24
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
	helperTestProgram(t, 0, 7, "", "error: undefined reference to start",
		`JP_Imm16 start
		:hello
		LD_A_Imm8 'H' OUT_Port_A 7
		LD_A_Imm8 'E' OUT_Port_A 7
		LD_A_Imm8 'L' OUT_Port_A 7
		LD_A_Imm8 'L' OUT_Port_A 7
		LD_A_Imm8 'O' OUT_Port_A 7
		RET
		:startee_not_defined
		CALL_Imm16 hello
		HALT
	`)
	helperTestProgram(t, 0, 7, "", "+",
		`XOR_A_A
		SET_1_A
		BIT_1_A
		JPNZ_Imm16 ok
		HALT
		:ok
		LD_A_Imm8 '+' OUT_Port_A 7
		HALT
	`)
}
