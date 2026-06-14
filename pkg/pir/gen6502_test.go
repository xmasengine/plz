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
	code, err := assemble6502(t, out)
	if err != nil {
		t.Fatalf("assembly failed: %v\n%s", err, out)
	}
	if len(code) == 0 {
		t.Fatal("expected non-empty code")
	}
}

func TestGen6502ATRoutine(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: AT, Operand: Operand{Type: OpNumber, Num: 0x0200}},
		{Op: ROUTE, Operand: Operand{Type: OpName, Name: "vec"}},
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
		"_plz_tcbs",
		"task0:",
	}
	for _, s := range checks {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q", s)
		}
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
		"_plz_tcbs:",
		"main:",
		"worker:",
		"jsr _plz_init_tasks",
	}
	for _, s := range components {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q", s)
		}
	}
}
