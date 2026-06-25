package plz_test

import (
	"bytes"
	"testing"

	"github.com/xmasengine/plz/pkg/cpu6502"
	asm "github.com/xmasengine/plz/pkg/asm6502"
	"github.com/xmasengine/plz/pkg/pir"
)

// brkDone6502 is set true when the BRK handler fires.
type brkDone6502 struct {
	done bool
}

func (b *brkDone6502) OnBrk(*cpu.CPU) {
	b.done = true
}

// assembleAndRun6502 assembles PIR instructions into 6502, runs them in the
// emulator, and returns memory state.
func assembleAndRun6502(t *testing.T, prog *pir.Program) *cpu.FlatMemory {
	t.Helper()
	cfg := pir.Default6502Config()
	asmText := pir.NewGen6502(cfg).Gen(prog)
	return run6502asm(t, asmText, cfg.Origin)
}

// run6502asm assembles 6502 assembly text and runs it in the emulator.
func run6502asm(t *testing.T, code string, origin uint16) *cpu.FlatMemory {
	t.Helper()
	r := bytes.NewReader([]byte(code))
	assembly, _, err := asm.Assemble(r, "test", origin, nil, 0)
	if err != nil || len(assembly.Errors) > 0 {
		for _, e := range assembly.Errors {
			t.Logf("asm error: %s", e)
		}
		t.Fatalf("assemble: %v\n%s", err, code)
	}

	mem := cpu.NewFlatMemory()
	off := int(origin)
	for i, b := range assembly.Code {
		mem.StoreByte(uint16(off+i), b)
	}

	emu := cpu.NewCPU(cpu.NMOS, mem)
	emu.Reg.PC = origin

	bd := &brkDone6502{}
	emu.AttachBrkHandler(bd)

	const maxSteps = 500000
	for i := 0; i < maxSteps && !bd.done; i++ {
		emu.Step()
	}
	if !bd.done {
		t.Fatalf("program did not complete (BRK not reached) after %d steps; PC=$%04X", maxSteps, emu.Reg.PC)
	}

	return mem
}

func Test6502_PushB(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 42}},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 42 {
		t.Errorf("expected 42 on stack at $3000, got %d", v)
	}
}

func Test6502_PushW(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_W, Operand: pir.Operand{Type: pir.OpNumber, Num: 0x1234}},
		{Op: pir.HLT},
	}})
	got := uint16(mem.LoadByte(0x3000)) | uint16(mem.LoadByte(0x3001))<<8
	if got != 0x1234 {
		t.Errorf("expected $1234 on stack at $3000, got $%04X", got)
	}
}

func Test6502_AddB(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 10}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 20}},
		{Op: pir.ADD_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 30 {
		t.Errorf("expected 30 on stack, got %d", v)
	}
}

func Test6502_AddW(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_W, Operand: pir.Operand{Type: pir.OpNumber, Num: 1000}},
		{Op: pir.PUSH_W, Operand: pir.Operand{Type: pir.OpNumber, Num: 2000}},
		{Op: pir.ADD_W},
		{Op: pir.HLT},
	}})
	got := uint16(mem.LoadByte(0x3000)) | uint16(mem.LoadByte(0x3001))<<8
	if got != 3000 {
		t.Errorf("expected 3000 on stack, got %d", got)
	}
}

func Test6502_SubB(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 50}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 8}},
		{Op: pir.SUB_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 42 {
		t.Errorf("expected 42 on stack, got %d", v)
	}
}

func Test6502_SubW(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_W, Operand: pir.Operand{Type: pir.OpNumber, Num: 100}},
		{Op: pir.PUSH_W, Operand: pir.Operand{Type: pir.OpNumber, Num: 30}},
		{Op: pir.SUB_W},
		{Op: pir.HLT},
	}})
	got := uint16(mem.LoadByte(0x3000)) | uint16(mem.LoadByte(0x3001))<<8
	if got != 70 {
		t.Errorf("expected 70 on stack, got %d", got)
	}
}

func Test6502_MulB(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 7}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 6}},
		{Op: pir.MUL_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 42 {
		t.Errorf("expected 42 on stack, got %d", v)
	}
}

func Test6502_DivB(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 42}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 7}},
		{Op: pir.DIV_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 6 {
		t.Errorf("expected 6 on stack, got %d", v)
	}
}

func Test6502_ModB(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 10}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 3}},
		{Op: pir.MOD_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 1 {
		t.Errorf("expected 1 on stack, got %d", v)
	}
}

func Test6502_NotB(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 0}},
		{Op: pir.NOT_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 1 {
		t.Errorf("expected 1 (NOT 0) on stack, got %d", v)
	}

	mem = assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 1}},
		{Op: pir.NOT_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 0 {
		t.Errorf("expected 0 (NOT 1) on stack, got %d", v)
	}
}

func Test6502_NotW(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_W, Operand: pir.Operand{Type: pir.OpNumber, Num: 0}},
		{Op: pir.NOT_W},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 1 {
		t.Errorf("expected 1 (NOT 0) on stack, got %d", v)
	}
}

func Test6502_AndOrXorB(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 0x0F}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 0x03}},
		{Op: pir.AND_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 0x03 {
		t.Errorf("expected $03 (0F AND 03) on stack, got $%02X", v)
	}

	mem = assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 0x0F}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 0x03}},
		{Op: pir.OR_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 0x0F {
		t.Errorf("expected $0F (0F OR 03) on stack, got $%02X", v)
	}

	mem = assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 0xFF}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 0x0F}},
		{Op: pir.XOR_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 0xF0 {
		t.Errorf("expected $F0 (FF XOR 0F) on stack, got $%02X", v)
	}
}

func Test6502_NegB(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 1}},
		{Op: pir.NEG_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 0xFF {
		t.Errorf("expected $FF (NEG 1) on stack, got $%02X", v)
	}
}

func Test6502_CompareB(t *testing.T) {
	// EQ: 5 == 5
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 5}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 5}},
		{Op: pir.IS_B, Operand: pir.Operand{Type: pir.OpCondition, Cond: pir.CondEQ}},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 1 {
		t.Errorf("expected 1 (5 EQ 5) on stack, got %d", v)
	}

	// LT: 3 < 5
	mem = assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 3}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 5}},
		{Op: pir.IS_B, Operand: pir.Operand{Type: pir.OpCondition, Cond: pir.CondLT}},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 1 {
		t.Errorf("expected 1 (3 LT 5) on stack, got %d", v)
	}

	// GT: 10 > 5
	mem = assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 10}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 5}},
		{Op: pir.IS_B, Operand: pir.Operand{Type: pir.OpCondition, Cond: pir.CondGT}},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 1 {
		t.Errorf("expected 1 (10 GT 5) on stack, got %d", v)
	}

	// NE: 3 != 5
	mem = assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 3}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 5}},
		{Op: pir.IS_B, Operand: pir.Operand{Type: pir.OpCondition, Cond: pir.CondNE}},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 1 {
		t.Errorf("expected 1 (3 NE 5) on stack, got %d", v)
	}
}

func Test6502_Shifts(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 1}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 3}},
		{Op: pir.SHL_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 8 {
		t.Errorf("expected 8 (1 SHL 3) on stack, got %d", v)
	}

	mem = assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 0x80}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 2}},
		{Op: pir.SHR_B},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 0x20 {
		t.Errorf("expected $20 ($80 SHR 2) on stack, got $%02X", v)
	}
}

func Test6502_DupDropSwap(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 10}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 20}},
		{Op: pir.SWAP},
		{Op: pir.DROP},
		{Op: pir.HLT},
	}})
	// After SWAP: TOS (20) goes to NEXT position, NEXT (10) becomes TOS.
	// After DROP: TOS (10) removed, leaving 20.
	if v := mem.LoadByte(0x3000); v != 20 {
		t.Errorf("expected 20 on stack after SWAP/DROP, got %d", v)
	}
}

func Test6502_GoIf(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 1}},
		{Op: pir.GO_IF, Operand: pir.Operand{Type: pir.OpName, Name: "_skip"}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 99}},
		{Op: pir.TAG, Operand: pir.Operand{Type: pir.OpName, Name: "_skip"}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 42}},
		{Op: pir.HLT},
	}})
	// GO_IF pops 1 (truthy), branches to _skip. PUSH 99 skipped, PUSH 42 runs.
	if v := mem.LoadByte(0x3000); v != 42 {
		t.Errorf("expected 42 on stack (branched past 99, pushed 42), got %d", v)
	}
}

func Test6502_GoIfFalse(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 0}},
		{Op: pir.GO_IF, Operand: pir.Operand{Type: pir.OpName, Name: "_skip"}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 77}},
		{Op: pir.TAG, Operand: pir.Operand{Type: pir.OpName, Name: "_skip"}},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 77 {
		t.Errorf("expected 77 on stack (GO_IF false, no branch), got %d", v)
	}
}

func Test6502_GetPutVar(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.VAR, Operand: pir.Operand{Type: pir.OpName, Name: "x"}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 42}},
		{Op: pir.PUT_B, Operand: pir.Operand{Type: pir.OpName, Name: "x"}},
		{Op: pir.GET_B, Operand: pir.Operand{Type: pir.OpName, Name: "x"}},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 42 {
		t.Errorf("expected 42 on stack (from GET_B x), got %d", v)
	}
}

func Test6502_GetPutVarWord(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.VAR, Operand: pir.Operand{Type: pir.OpName, Name: "x"}},
		{Op: pir.PUSH_W, Operand: pir.Operand{Type: pir.OpNumber, Num: 0x1234}},
		{Op: pir.PUT_W, Operand: pir.Operand{Type: pir.OpName, Name: "x"}},
		{Op: pir.GET_W, Operand: pir.Operand{Type: pir.OpName, Name: "x"}},
		{Op: pir.HLT},
	}})
	got := uint16(mem.LoadByte(0x3000)) | uint16(mem.LoadByte(0x3001))<<8
	if got != 0x1234 {
		t.Errorf("expected $1234 on stack (from GET_W x), got $%04X", got)
	}
}

func Test6502_VarCopy(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.VAR, Operand: pir.Operand{Type: pir.OpName, Name: "x"}},
		{Op: pir.VAR, Operand: pir.Operand{Type: pir.OpName, Name: "y"}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 7}},
		{Op: pir.PUT_B, Operand: pir.Operand{Type: pir.OpName, Name: "x"}},
		{Op: pir.GET_B, Operand: pir.Operand{Type: pir.OpName, Name: "x"}},
		{Op: pir.PUT_B, Operand: pir.Operand{Type: pir.OpName, Name: "y"}},
		{Op: pir.GET_B, Operand: pir.Operand{Type: pir.OpName, Name: "y"}},
		{Op: pir.HLT},
	}})
	if v := mem.LoadByte(0x3000); v != 7 {
		t.Errorf("expected 7 on stack (y = x = 7), got %d", v)
	}
}

func Test6502_NmiHandler(t *testing.T) {
	// Verify that an NMI handler executes correctly on the 6502 emulator.
	prog := &pir.Program{Instrs: []pir.Instr{
		{Op: pir.NMI, Operand: pir.Operand{Type: pir.OpName, Name: "my_nmi"}},
		{Op: pir.ROUTE, Operand: pir.Operand{Type: pir.OpName, Name: "my_nmi"}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 42}},
		{Op: pir.OUT_B},
		{Op: pir.DONE_NMI},
		{Op: pir.HLT},
	}}
	cfg := pir.Default6502Config()
	gen := pir.NewGen6502(cfg)
	asmText := gen.Gen(prog)

	// Assemble and get source map with exports
	r := bytes.NewReader([]byte(asmText))
	assembly, sourceMap, err := asm.Assemble(r, "test", cfg.Origin, nil, 0)
	if err != nil || len(assembly.Errors) > 0 {
		for _, e := range assembly.Errors {
			t.Logf("asm error: %s", e)
		}
		t.Fatalf("assemble: %v\n%s", err, asmText)
	}

	// Find NMI handler address from exports
	var nmiAddr uint16
	for _, ex := range sourceMap.Exports {
		if ex.Label == "my_nmi" {
			nmiAddr = ex.Address
		}
	}
	if nmiAddr == 0 {
		t.Fatal("NMI handler export not found in source map")
	}
	t.Logf("NMI handler at $%04X", nmiAddr)

	// Load code into memory
	mem := cpu.NewFlatMemory()
	off := int(cfg.Origin)
	for i, b := range assembly.Code {
		mem.StoreByte(uint16(off+i), b)
	}
	// Set NMI vector at $FFFA
	mem.StoreByte(0xFFFA, byte(nmiAddr))
	mem.StoreByte(0xFFFB, byte(nmiAddr>>8))

	emu := cpu.NewCPU(cpu.NMOS, mem)
	emu.Reg.PC = cfg.Origin

	// Step past the initial jmp to main
	emu.Step()
	// Run until the RTI returns (we step the init + HLT)
	const maxSteps = 5000
	for i := 0; i < maxSteps; i++ {
		if emu.Reg.PC == 0xFFFF || emu.Reg.PC == 0x0000 {
			break
		}
		emu.Step()
	}
	// Now trigger NMI
	emu.NMI()
	// Step through the handler
	for i := 0; i < maxSteps; i++ {
		if emu.Reg.PC == 0xFFFF || emu.Reg.PC == 0x0000 {
			break
		}
		emu.Step()
	}
	// Check that the NMI handler wrote 42 to output
	if v := mem.LoadByte(cfg.OutputBase); v != 42 {
		t.Errorf("expected 42 at output, got %d", v)
	}
}

func Test6502_IrqHandler(t *testing.T) {
	// Verify that an IRQ handler executes correctly on the 6502 emulator.
	prog := &pir.Program{Instrs: []pir.Instr{
		{Op: pir.INT, Operand: pir.Operand{Type: pir.OpName, Name: "my_irq"}},
		{Op: pir.ROUTE, Operand: pir.Operand{Type: pir.OpName, Name: "my_irq"}},
		{Op: pir.PUSH_B, Operand: pir.Operand{Type: pir.OpNumber, Num: 77}},
		{Op: pir.OUT_B},
		{Op: pir.DONE_INTERRUPT},
		{Op: pir.HLT},
	}}
	cfg := pir.Default6502Config()
	gen := pir.NewGen6502(cfg)
	asmText := gen.Gen(prog)

	// Assemble and get source map with exports
	r := bytes.NewReader([]byte(asmText))
	assembly, sourceMap, err := asm.Assemble(r, "test", cfg.Origin, nil, 0)
	if err != nil || len(assembly.Errors) > 0 {
		for _, e := range assembly.Errors {
			t.Logf("asm error: %s", e)
		}
		t.Fatalf("assemble: %v\n%s", err, asmText)
	}

	// Find IRQ handler address from exports
	var irqAddr uint16
	for _, ex := range sourceMap.Exports {
		if ex.Label == "my_irq" {
			irqAddr = ex.Address
		}
	}
	if irqAddr == 0 {
		t.Fatal("IRQ handler export not found in source map")
	}
	t.Logf("IRQ handler at $%04X", irqAddr)

	mem := cpu.NewFlatMemory()
	off := int(cfg.Origin)
	for i, b := range assembly.Code {
		mem.StoreByte(uint16(off+i), b)
	}
	// Set IRQ vector at $FFFE
	mem.StoreByte(0xFFFE, byte(irqAddr))
	mem.StoreByte(0xFFFF, byte(irqAddr>>8))

	emu := cpu.NewCPU(cpu.NMOS, mem)
	emu.Reg.PC = cfg.Origin

	// Run past the initial jmp
	const maxSteps = 5000
	for i := 0; i < maxSteps; i++ {
		if emu.Reg.PC == 0xFFFF || emu.Reg.PC == 0x0000 {
			break
		}
		emu.Step()
	}
	// Enable interrupts before triggering IRQ
	emu.Reg.InterruptDisable = false
	emu.IRQ()
	for i := 0; i < maxSteps; i++ {
		if emu.Reg.PC == 0xFFFF || emu.Reg.PC == 0x0000 {
			break
		}
		emu.Step()
	}
	if v := mem.LoadByte(cfg.OutputBase); v != 77 {
		t.Errorf("expected 77 at output, got %d", v)
	}
}

func TestIntegrationPLZ_NmiHandler(t *testing.T) {
	// Full pipeline: PL/Z source → PIR → NES → emulator, then trigger NMI.
	pirProg := compilePIR(t, `
PROCEDURE my_nmi() NMI
  OUTPUT 0 42
END
NMI my_nmi
HALT`)

	cfg := pir.NES6502Config()
	gen := pir.NewGen6502(cfg)
	asmText := gen.Gen(pirProg)
	cfg.IntHandlerName = gen.IntHandler()
	cfg.NmiHandlerName = gen.NmiHandler()

	rom, err := pir.Assemble6502(cfg, asmText, gen.BankLines())
	if err != nil {
		t.Fatalf("Assemble6502: %v\n%s", err, asmText)
	}

	prgData := rom[16:]
	prgSize := int(rom[4]) * 0x4000
	mem := cpu.NewFlatMemory()
	for i, b := range prgData[:prgSize] {
		mem.StoreByte(uint16(0x8000+i), b)
	}

	resetVec := uint16(mem.LoadByte(0xFFFC)) | uint16(mem.LoadByte(0xFFFD))<<8
	emu := cpu.NewCPU(cpu.NMOS, mem)
	emu.Reg.PC = resetVec
	bd := &brkDone6502{}
	emu.AttachBrkHandler(bd)

	// Step past initial jmp to main
	emu.Step()
	// Run main program to BRK
	const maxSteps = 500000
	for i := 0; i < maxSteps && !bd.done; i++ {
		emu.Step()
	}
	if !bd.done {
		t.Fatalf("program did not complete after %d steps; PC=$%04X", maxSteps, emu.Reg.PC)
	}

	// Trigger NMI and step through handler
	emu.NMI()
	for i := 0; i < maxSteps; i++ {
		if emu.Reg.PC == 0xFFFF || emu.Reg.PC == 0x0000 {
			break
		}
		emu.Step()
	}

	// Check that NMI handler wrote 42 to output
	if v := mem.LoadByte(cfg.OutputBase); v != 42 {
		t.Errorf("expected 42 at output (from NMI), got %d", v)
	}
}

func Test6502_AddWMulti(t *testing.T) {
	mem := assembleAndRun6502(t, &pir.Program{Instrs: []pir.Instr{
		{Op: pir.PUSH_W, Operand: pir.Operand{Type: pir.OpNumber, Num: 100}},
		{Op: pir.PUSH_W, Operand: pir.Operand{Type: pir.OpNumber, Num: 200}},
		{Op: pir.ADD_W},
		{Op: pir.PUSH_W, Operand: pir.Operand{Type: pir.OpNumber, Num: 300}},
		{Op: pir.ADD_W},
		{Op: pir.HLT},
	}})
	got := uint16(mem.LoadByte(0x3000)) | uint16(mem.LoadByte(0x3001))<<8
	if got != 600 {
		t.Errorf("expected 600 on stack ((100+200)+300), got %d", got)
	}
}
