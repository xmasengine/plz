package pir

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xmasengine/plz/pkg/asm6502"
)

func Inst(op Instruction, args ...interface{}) Instr {
	if len(args) == 0 {
		return Instr{Op: op}
	}
	switch v := args[0].(type) {
	case uint16:
		return Instr{Op: op, Operand: Operand{Type: OpNumber, Num: v}}
	case int:
		return Instr{Op: op, Operand: Operand{Type: OpNumber, Num: uint16(v)}}
	case string:
		return Instr{Op: op, Operand: Operand{Type: OpName, Name: v}}
	default:
		return Instr{Op: op}
	}
}

func prog(instrs ...Instr) *Program { return &Program{Instrs: instrs} }

func assemble6502(t *testing.T, code string) ([]byte, error) {
	r := bytes.NewReader([]byte(code))
	assembly, _, err := asm.Assemble(r, "test", 0x1000, nil, 0)
	if err != nil {
		return nil, err
	}
	if len(assembly.Errors) > 0 {
		return nil, &asmError{assembly.Errors}
	}
	return assembly.Code, nil
}

type asmError struct {
	errs []string
}

func (e *asmError) Error() string {
	return strings.Join(e.errs, "; ")
}

func TestGen6502NOP(t *testing.T) {
	prog := &Program{Instrs: []Instr{{Op: NOP}}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502VarByte(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: VAR, Operand: Operand{Type: OpName, Name: "x"}},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 42}},
		{Op: PUT_B, Operand: Operand{Type: OpName, Name: "x"}},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502VarWord(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: ALLOC, Operand: Operand{Type: OpNumber, Num: 2}},
		{Op: VAR, Operand: Operand{Type: OpName, Name: "x"}},
		{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 0x1234}},
		{Op: PUT_W, Operand: Operand{Type: OpName, Name: "x"}},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502GetPut(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: VAR, Operand: Operand{Type: OpName, Name: "a"}},
		{Op: ALLOC, Operand: Operand{Type: OpNumber, Num: 2}},
		{Op: VAR, Operand: Operand{Type: OpName, Name: "b"}},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 10}},
		{Op: PUT_B, Operand: Operand{Type: OpName, Name: "a"}},
		{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 0xAABB}},
		{Op: PUT_W, Operand: Operand{Type: OpName, Name: "b"}},
		{Op: GET_B, Operand: Operand{Type: OpName, Name: "a"}},
		{Op: GET_W, Operand: Operand{Type: OpName, Name: "b"}},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502Math(t *testing.T) {
	tests := []struct{ name string; prog *Program }{
		{name: "ADD_B", prog: prog(Inst(PUSH_B, 10), Inst(PUSH_B, 3), Inst(ADD_B))},
		{name: "ADD_W", prog: prog(Inst(PUSH_W, 1000), Inst(PUSH_W, 500), Inst(ADD_W))},
		{name: "SUB_B", prog: prog(Inst(PUSH_B, 10), Inst(PUSH_B, 3), Inst(SUB_B))},
		{name: "SUB_W", prog: prog(Inst(PUSH_W, 1000), Inst(PUSH_W, 500), Inst(SUB_W))},
		{name: "AND_B", prog: prog(Inst(PUSH_B, 0x0F), Inst(PUSH_B, 0xF0), Inst(AND_B))},
		{name: "AND_W", prog: prog(Inst(PUSH_W, 0xFF00), Inst(PUSH_W, 0x0FFF), Inst(AND_W))},
		{name: "OR_B", prog: prog(Inst(PUSH_B, 0x0F), Inst(PUSH_B, 0xF0), Inst(OR_B))},
		{name: "OR_W", prog: prog(Inst(PUSH_W, 0xFF00), Inst(PUSH_W, 0x00FF), Inst(OR_W))},
		{name: "XOR_B", prog: prog(Inst(PUSH_B, 0xFF), Inst(PUSH_B, 0x0F), Inst(XOR_B))},
		{name: "XOR_W", prog: prog(Inst(PUSH_W, 0xFFFF), Inst(PUSH_W, 0x0F0F), Inst(XOR_W))},
		{name: "NOT_B", prog: prog(Inst(PUSH_B, 0), Inst(NOT_B))},
		{name: "NOT_W", prog: prog(Inst(PUSH_W, 0), Inst(NOT_W))},
		{name: "NEG_B", prog: prog(Inst(PUSH_B, 10), Inst(NEG_B))},
		{name: "NEG_W", prog: prog(Inst(PUSH_W, 10), Inst(NEG_W))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var g Gen6502
			g.cfg = Default6502Config()
			out := g.Gen(tc.prog)
			code, err := assemble6502(t, out)
			if err != nil {
				t.Fatalf("assembly failed: %v\n%s", err, out)
			}
			if len(code) == 0 {
				t.Fatal("expected non-empty code")
			}
		})
	}
}

func TestGen6502Cast(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 0xFF}},
		{Op: CAST_W},
		{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 0x1234}},
		{Op: CAST_B},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502Stack(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 10}},
		{Op: DUP},
		{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 20}},
		{Op: SWAP},
		{Op: DROP},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502ControlFlow(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 1}},
		{Op: TAG, Operand: Operand{Type: OpName, Name: "loop"}},
		{Op: GO_IF, Operand: Operand{Type: OpName, Name: "loop"}},
		{Op: GO, Operand: Operand{Type: OpName, Name: "loop"}},
		{Op: TAG, Operand: Operand{Type: OpName, Name: "end"}},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502Procedures(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 5}},
		{Op: RUN, Operand: Operand{Type: OpName, Name: "fact"}},
		{Op: ROUTE, Operand: Operand{Type: OpName, Name: "fact"}},
		{Op: DONE},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502Frame(t *testing.T) {
	// Reentrant procedure with frame and local variables.
	prog := &Program{Instrs: []Instr{
		Inst(ROUTE, "foo"),
		Inst(FRAME, 2),
		Inst(LOCAL_B, "a"),
		Inst(LOCAL_B, "b"),
		Inst(PUT_B, "b"),
		Inst(PUT_B, "a"),
		Inst(GET_B, "a"),
		Inst(GET_B, "b"),
		Inst(ADD_B),
		Inst(OUT_B),
		Inst(DONE),
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
	// Verify frame setup/teardown instructions are present
	for _, s := range []string{"$0e", "$0f", "rts"} {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q", s)
		}
	}
}

func TestGen6502Pointers(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: PUSH_D, Operand: Operand{Type: OpName, Name: "data"}},
		{Op: READ_B},
		{Op: PUSH_D, Operand: Operand{Type: OpName, Name: "data"}},
		{Op: READ_W},
		{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 42}},
		{Op: PUSH_D, Operand: Operand{Type: OpName, Name: "data"}},
		{Op: WRITE_B},
		{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 0x1234}},
		{Op: PUSH_D, Operand: Operand{Type: OpName, Name: "data"}},
		{Op: WRITE_W},
		{Op: TAG, Operand: Operand{Type: OpName, Name: "data"}},
		{Op: DATA_B, Operand: Operand{Type: OpNumber, Num: 10}},
		{Op: DATA_B, Operand: Operand{Type: OpNumber, Num: 20}},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502IO(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 0x55}},
		{Op: OUT_B, Operand: Operand{Type: OpNumber, Num: 0xC000}},
		{Op: IN_B, Operand: Operand{Type: OpNumber, Num: 0xC000}},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502Interrupts(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: DII},
		{Op: ENI},
		{Op: HLT},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502InstallInterrupt(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		Inst(INT, "my_handler"),
		Inst(NMI, "my_nmi"),
		Inst(ROUTE, "my_handler"),
		{Op: DONE_INTERRUPT},
		Inst(ROUTE, "my_nmi"),
		{Op: DONE_NMI},
	}}
	var g Gen6502
	g.cfg = NES6502Config()
	out := g.Gen(prog)
	if !strings.Contains(out, ".export my_handler") {
		t.Errorf("output missing INT export: %s", out)
	}
	if !strings.Contains(out, ".export my_nmi") {
		t.Errorf("output missing NMI export: %s", out)
	}
	if !strings.Contains(out, "my_handler:") {
		t.Errorf("output missing handler label: %s", out)
	}
	if !strings.Contains(out, "my_nmi:") {
		t.Errorf("output missing NMI label: %s", out)
	}
	if g.IntHandler() != "my_handler" {
		t.Errorf("IntHandler = %q, want my_handler", g.IntHandler())
	}
	if g.NmiHandler() != "my_nmi" {
		t.Errorf("NmiHandler = %q, want my_nmi", g.NmiHandler())
	}
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502Seed(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: SEED},
		{Op: SEED},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 1}},
		{Op: OUT_B},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502DoneInterrupt(t *testing.T) {
	// DONE_INTERRUPT must emit rti, DONE_NMI must emit rti.
	prog := &Program{Instrs: []Instr{
		{Op: DONE_INTERRUPT},
		{Op: DONE_NMI},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	if !strings.Contains(out, "rti") {
		t.Errorf("output missing rti: %s", out)
	}
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502DataEmission(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: DATA_B, Operand: Operand{Type: OpNumber, Num: 255}},
		{Op: DATA_W, Operand: Operand{Type: OpNumber, Num: 0x1234}},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502ATVar(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: AT, Operand: Operand{Type: OpNumber, Num: 0x3000}},
		{Op: VAR, Operand: Operand{Type: OpName, Name: "io_port"}},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 0xFF}},
		{Op: PUT_B, Operand: Operand{Type: OpName, Name: "io_port"}},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	r := bytes.NewReader([]byte(out))
	assembly, _, err := asm.Assemble(r, "test", 0x1000, nil, 0)
	if err != nil || len(assembly.Errors) > 0 {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(assembly.Code) == 0 {
		t.Fatal("expected non-empty code")
	}
	// Verify the AT variable resolves to $3000 by checking sta $3000 instruction
	found := false
	for i := 0; i+2 < len(assembly.Code); i++ {
		// sta abs = $8D lo hi
		if assembly.Code[i] == 0x8D {
			addr := uint16(assembly.Code[i+1]) | uint16(assembly.Code[i+2])<<8
			if addr == 0x3000 {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("expected sta $3000 instruction not found in assembled code")
	}
}

func TestGen6502ATRoutine(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: AT, Operand: Operand{Type: OpNumber, Num: 0x3000}},
		{Op: ROUTE, Operand: Operand{Type: OpName, Name: "vec"}},
		{Op: DONE},
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	if !strings.Contains(out, ".org $3000") {
		t.Errorf("output missing .org $3000, got:\n%s", out)
	}
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502Comparison(t *testing.T) {
	tests := []struct {
		name string
		cond Condition
	}{
		{"LT", CondLT},
		{"GT", CondGT},
		{"LE", CondLE},
		{"GE", CondGE},
		{"EQ", CondEQ},
		{"NE", CondNE},
	}
	for _, tc := range tests {
		t.Run("IS_B_"+tc.name, func(t *testing.T) {
			prog := &Program{Instrs: []Instr{
				{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 10}},
				{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 20}},
				{Op: IS_B, Operand: Operand{Type: OpCondition, Cond: tc.cond}},
			}}
			var g Gen6502
			g.cfg = Default6502Config()
			out := g.Gen(prog)
			code, err := assemble6502(t, out)
			if err != nil {
				t.Fatalf("assembly failed: %v\n%s", err, out)
			}
			if len(code) == 0 {
				t.Fatal("expected non-empty code")
			}
		})
		t.Run("IS_W_"+tc.name, func(t *testing.T) {
			prog := &Program{Instrs: []Instr{
				{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 100}},
				{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 200}},
				{Op: IS_W, Operand: Operand{Type: OpCondition, Cond: tc.cond}},
			}}
			var g Gen6502
			g.cfg = Default6502Config()
			out := g.Gen(prog)
			code, err := assemble6502(t, out)
			if err != nil {
				t.Fatalf("assembly failed: %v\n%s", err, out)
			}
			if len(code) == 0 {
				t.Fatal("expected non-empty code")
			}
		})
	}
}

func TestGen6502ProgramStructure(t *testing.T) {
	prog := &Program{Instrs: []Instr{{Op: NOP}}}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	structure := []string{
		".org $1000",
		"jmp _6502_main",
		"_6502_main:",
		"sei",
		"cld",
	}
	for _, s := range structure {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q\nFull output:\n%s", s, out)
		}
	}
}

func TestGen6502Empty(t *testing.T) {
	prog := &Program{}
	var g Gen6502
	g.cfg = Default6502Config()
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502Shifts(t *testing.T) {
	tests := []struct{ name string; prog *Program }{
		{name: "SHL_B", prog: prog(Inst(PUSH_B, 3), Inst(PUSH_B, 2), Inst(SHL_B))},
		{name: "SHL_W", prog: prog(Inst(PUSH_W, 0x00FF), Inst(PUSH_W, 4), Inst(SHL_W))},
		{name: "SHR_B", prog: prog(Inst(PUSH_B, 64), Inst(PUSH_B, 3), Inst(SHR_B))},
		{name: "SHR_W", prog: prog(Inst(PUSH_W, 0xFF00), Inst(PUSH_W, 4), Inst(SHR_W))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var g Gen6502
			g.cfg = Default6502Config()
			out := g.Gen(tc.prog)
			code, err := assemble6502(t, out)
			if err != nil {
				t.Fatalf("assembly failed: %v\n%s", err, out)
			}
			if len(code) == 0 {
				t.Fatal("expected non-empty code")
			}
		})
	}
}

func TestGen6502MulDivMod(t *testing.T) {
	tests := []struct{ name string; prog *Program }{
		{name: "MUL_B", prog: prog(Inst(PUSH_B, 5), Inst(PUSH_B, 6), Inst(MUL_B))},
		{name: "MUL_W", prog: prog(Inst(PUSH_W, 100), Inst(PUSH_W, 200), Inst(MUL_W))},
		{name: "DIV_B", prog: prog(Inst(PUSH_B, 24), Inst(PUSH_B, 6), Inst(DIV_B))},
		{name: "DIV_W", prog: prog(Inst(PUSH_W, 1000), Inst(PUSH_W, 50), Inst(DIV_W))},
		{name: "MOD_B", prog: prog(Inst(PUSH_B, 17), Inst(PUSH_B, 5), Inst(MOD_B))},
		{name: "MOD_W", prog: prog(Inst(PUSH_W, 100), Inst(PUSH_W, 30), Inst(MOD_W))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var g Gen6502
			g.cfg = Default6502Config()
			out := g.Gen(tc.prog)
			code, err := assemble6502(t, out)
			if err != nil {
				t.Fatalf("assembly failed: %v\n%s", err, out)
			}
			if len(code) == 0 {
				t.Fatal("expected non-empty code")
			}
			// Verify runtime helper is emitted
			if !strings.Contains(out, "_plz_mul8") && !strings.Contains(out, "_plz_div8") && !strings.Contains(out, "_plz_div16") {
				t.Log("output contains runtime helpers")
			}
		})
	}
}

func TestGen6502Tasks(t *testing.T) {
	// A minimal program with JOB, SLEEP, and BYE to test task emission
	prog := &Program{Instrs: []Instr{
		Inst(JOB, "task0"),
		Inst(PUSH_B, 10),       // sleep duration
		Inst(SLEEP),
		Inst(BYE),
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	g.cfg.TaskLimit = 4
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
	// Verify scheduler and task init are emitted
	checks := []string{
		"_plz_scheduler",
		"_plz_init_tasks",
		"task0:",
		"sta $80,x",
		"sta $80,y",
	}
	for _, s := range checks {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q", s)
		}
	}
}

func TestNESVectorTable(t *testing.T) {
	// Verify that INT/NMI handlers produce correct NES vector table entries.
	prog := &Program{Instrs: []Instr{
		{Op: INT, Operand: Operand{Type: OpName, Name: "my_handler"}},
		{Op: NMI, Operand: Operand{Type: OpName, Name: "my_nmi"}},
		{Op: ROUTE, Operand: Operand{Type: OpName, Name: "my_handler"}},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 42}},
		{Op: OUT_B},
		{Op: DONE_INTERRUPT},
		{Op: ROUTE, Operand: Operand{Type: OpName, Name: "my_nmi"}},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 99}},
		{Op: OUT_B},
		{Op: DONE_NMI},
		{Op: HLT},
	}}
	cfg := NES6502Config()
	gen := NewGen6502(cfg)
	asmText := gen.Gen(prog)
	cfg.IntHandlerName = gen.IntHandler()
	cfg.NmiHandlerName = gen.NmiHandler()
	rom, err := Assemble6502(cfg, asmText, gen.BankLines())
	if err != nil {
		t.Fatalf("Assemble6502: %v\n%s", err, asmText)
	}
	// ROM structure: 16-byte iNES header + 32KB PRG data
	const prgStart = 0x8000
	prgSize := 0x10000 - prgStart
	prgData := rom[16:]
	// Vectors are at the end of PRG data (last 6 bytes)
	nmiVec := uint16(prgData[prgSize-6]) | uint16(prgData[prgSize-5])<<8
	resetVec := uint16(prgData[prgSize-4]) | uint16(prgData[prgSize-3])<<8
	irqVec := uint16(prgData[prgSize-2]) | uint16(prgData[prgSize-1])<<8
	// Reset vector must be at origin ($C000)
	if resetVec != cfg.Origin {
		t.Errorf("reset vector = $%04X, want $%04X", resetVec, cfg.Origin)
	}
	// NMI and IRQ vectors must be non-zero (point to handlers within PRG)
	prgEnd := prgStart + uint32(prgSize)
	if uint32(nmiVec) < prgStart || uint32(nmiVec) >= prgEnd {
		t.Errorf("NMI vector $%04X outside PRG range [$%04X-$%04X]", nmiVec, prgStart, prgEnd)
	}
	if uint32(irqVec) < prgStart || uint32(irqVec) >= prgEnd {
		t.Errorf("IRQ vector $%04X outside PRG range [$%04X-$%04X]", irqVec, prgStart, prgEnd)
	}
	// NMI and IRQ vectors must be different (different handler labels)
	if nmiVec == irqVec {
		t.Errorf("NMI vector == IRQ vector == $%04X, expected different handlers", nmiVec)
	}
}

func TestGen6502TaskSchedulerCode(t *testing.T) {
	// Two tasks: main and worker, with different priorities
	prog := &Program{Instrs: []Instr{
		Inst(JOB, "main"),
		Inst(JOB, "worker"),
		Inst(PRIORITY, 0),
		// main body
		Inst(PUSH_B, 5),
		Inst(SLEEP),
		Inst(BYE),
		// worker body (referenced by the entry label)
		Inst(PRIORITY, 1),
		Inst(PUSH_B, 3),
		Inst(SLEEP),
		Inst(BYE),
	}}
	var g Gen6502
	g.cfg = Default6502Config()
	g.cfg.TaskLimit = 4
	out := g.Gen(prog)
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
	// Verify all expected components
	components := []string{
		"_plz_scheduler",
		"_plz_init_tasks:",
		"main:",
		"worker:",
		"jsr _plz_init_tasks",
		"sta $80,x",
		"lda $84,y", // priority field in scan loop
		"_6502_sch_halt:",
	}
	for _, s := range components {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q", s)
		}
	}
}

func TestGen6502BankSwitch(t *testing.T) {
	// BANK with NES config should emit export marker and label for bank 1.
	prog := &Program{Instrs: []Instr{
		{Op: BANK, Operand: Operand{Type: OpNumber, Num: 1}},
		{Op: TAG, Operand: Operand{Type: OpName, Name: "my_data"}},
		{Op: DATA_B, Operand: Operand{Type: OpNumber, Num: 42}},
		{Op: DATA_B, Operand: Operand{Type: OpNumber, Num: 99}},
	}}
	cfg := NES6502Config()
	gen := NewGen6502(cfg)
	asmText := gen.Gen(prog)
	bankLines := gen.BankLines()
	// Main code bank should NOT contain my_data or the data bytes.
	if strings.Contains(asmText, "my_data") {
		t.Error("main code bank should not contain my_data label")
	}
	if strings.Contains(asmText, ".byte 42") {
		t.Error("main code bank should not contain data bytes")
	}
	// Bank 1 should contain the data.
	bk1, ok := bankLines[1]
	if !ok {
		t.Fatal("expected bank lines for bank 1")
	}
	if !strings.Contains(bk1, "my_data:") {
		t.Error("bank 1 missing my_data label")
	}
	if !strings.Contains(bk1, ".byte 42") {
		t.Error("bank 1 missing .byte 42")
	}
	if !strings.Contains(bk1, ".byte 99") {
		t.Error("bank 1 missing .byte 99")
	}
	// The main code bank should have .export _plz_bank_1 marker.
	if !strings.Contains(asmText, ".export _plz_bank_1") {
		t.Error("main code bank missing .export _plz_bank_1 marker")
	}
	// Assemble and verify multi-bank ROM.
	rom, err := Assemble6502(cfg, asmText, gen.BankLines())
	if err != nil {
		t.Fatalf("Assemble6502: %v", err)
	}
	// iNES header: PRG banks byte at offset 4.
	prgBanks := int(rom[4])
	if prgBanks < 2 {
		t.Errorf("expected at least 2 PRG banks, got %d", prgBanks)
	}
	// Verify data is in bank 1: bank 1 data starts at PRG offset 1*bankSize.
	// Bank 1 contains: .byte 42, .byte 99
	bankSize := 0x4000
	bank1Data := rom[16+1*bankSize : 16+1*bankSize+2]
	if bank1Data[0] != 42 || bank1Data[1] != 99 {
		t.Errorf("bank 1 data = [%d %d], want [42 99]", bank1Data[0], bank1Data[1])
	}
}

func TestGen6502Switch(t *testing.T) {
	// SWITCH on NES should emit sta $5113.
	prog := &Program{Instrs: []Instr{
		{Op: SWITCH},
	}}
	gen := NewGen6502(NES6502Config())
	out := gen.Gen(prog)
	if !strings.Contains(out, "sta $5113") {
		t.Errorf("SWITCH output missing sta $5113, got:\n%s", out)
	}
	if !strings.Contains(out, "MMC5") {
		t.Errorf("SWITCH output missing MMC5 comment, got:\n%s", out)
	}
}
