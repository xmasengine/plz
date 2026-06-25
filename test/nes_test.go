package plz_test

import (
	"testing"

	"github.com/xmasengine/plz/pkg/cpu6502"
	"github.com/xmasengine/plz/pkg/pir"
)

// runNES takes a PIR program, generates a NES ROM, runs it on the CPU
// emulator (with iNES header stripped), and returns memory state.
func runNES(t *testing.T, prog *pir.Program) *cpu.FlatMemory {
	t.Helper()
	cfg := pir.NES6502Config()
	gen := pir.NewGen6502(cfg)
	asmText := gen.Gen(prog)
	cfg.IntHandlerName = gen.IntHandler()
	cfg.NmiHandlerName = gen.NmiHandler()
	rom, err := pir.Assemble6502(cfg, asmText, gen.BankLines())
	if err != nil {
		t.Fatalf("Assemble6502: %v\n%s", err, asmText)
	}
	// Strip iNES header; PRG data follows
	prgData := rom[16:]

	mem := cpu.NewFlatMemory()
	// NES maps PRG ROM at $8000-$FFFF
	for i, b := range prgData {
		mem.StoreByte(uint16(0x8000+i), b)
	}

	resetVec := uint16(mem.LoadByte(0xFFFC)) | uint16(mem.LoadByte(0xFFFD))<<8

	emu := cpu.NewCPU(cpu.NMOS, mem)
	emu.Reg.PC = resetVec

	bd := &brkDone6502{}
	emu.AttachBrkHandler(bd)

	const maxSteps = 500000
	for i := 0; i < maxSteps && !bd.done; i++ {
		emu.Step()
	}
	if !bd.done {
		t.Fatalf("NES program did not complete (BRK not reached) after %d steps; PC=$%04X", maxSteps, emu.Reg.PC)
	}

	return mem
}

func TestNES_PushB(t *testing.T) {
	mem := runNES(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 42}},
		{Op: pir.HLT},
	}})
	// NES data stack starts at $0200 (NES6502Config.StackBase)
	if v := mem.LoadByte(0x0200); v != 42 {
		t.Errorf("expected 42 on stack at $0200, got %d", v)
	}
}

func TestNES_AddB(t *testing.T) {
	mem := runNES(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 10}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 20}},
		{Op: pir.ADD_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x0200); v != 30 {
		t.Errorf("expected 30 on stack, got %d", v)
	}
}

func TestNES_Vars(t *testing.T) {
	mem := runNES(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.VAR, Operand: pir.Operand{Type: pir.OpName, Name: "x"}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 77}},
		{Op: pir.PUT_B, Operand: pir.Operand{Type: pir.OpName, Name: "x"}},
		{Op: pir.GET_B, Operand: pir.Operand{Type: pir.OpName, Name: "x"}},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x0200); v != 77 {
		t.Errorf("expected 77 on stack (from variable x), got %d", v)
	}
}

func TestNES_GoIf(t *testing.T) {
	mem := runNES(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 1}},
		{Op: pir.GO_IF, Operand: pir.Operand{Type: pir.OpName, Name: "_skip"}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 99}},
		{Op: pir.TAG, Operand: pir.Operand{Type: pir.OpName, Name: "_skip"}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 42}},
		{Op: pir.HLT},
	}})
	// GO_IF branches past PUSH 99; PUSH 42 should be on stack
	if v := mem.LoadByte(0x0200); v != 42 {
		t.Errorf("expected 42 on stack (branched past 99), got %d", v)
	}
}
