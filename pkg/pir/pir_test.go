package pir

import (
	"testing"
)

func TestInstrStringNoOperand(t *testing.T) {
	tests := []struct {
		op   Instruction
		want string
	}{
		{NOP, "NOP"},
		{ADD_B, "ADD_B"},
		{ADD_W, "ADD_W"},
		{SUB_B, "SUB_B"},
		{SUB_W, "SUB_W"},
		{MUL_B, "MUL_B"},
		{MUL_W, "MUL_W"},
		{DIV_B, "DIV_B"},
		{DIV_W, "DIV_W"},
		{MOD_B, "MOD_B"},
		{MOD_W, "MOD_W"},
		{SHL_B, "SHL_B"},
		{SHL_W, "SHL_W"},
		{SHR_B, "SHR_B"},
		{SHR_W, "SHR_W"},
		{AND_B, "AND_B"},
		{AND_W, "AND_W"},
		{OR_B, "OR_B"},
		{OR_W, "OR_W"},
		{XOR_B, "XOR_B"},
		{XOR_W, "XOR_W"},
		{NEG_B, "NEG_B"},
		{NEG_W, "NEG_W"},
		{NOT_B, "NOT_B"},
		{NOT_W, "NOT_W"},
		{CAST_W, "CAST_W"},
		{CAST_B, "CAST_B"},
		{DUP, "DUP"},
		{DROP, "DROP"},
		{SWAP, "SWAP"},
		{READ_B, "READ_B"},
		{READ_W, "READ_W"},
		{WRITE_B, "WRITE_B"},
		{WRITE_W, "WRITE_W"},
		{BYE, "BYE"},
		{YIELD, "YIELD"},
		{SLEEP, "SLEEP"},
		{HLT, "HLT"},
		{DII, "DII"},
		{ENI, "ENI"},
		{SEED, "SEED"},
		{SWITCH, "SWITCH"},
		{SAVE, "SAVE"},
		{LOAD, "LOAD"},
		{SRAM_ON, "SRAM_ON"},
		{SRAM_OFF, "SRAM_OFF"},
	}
	for _, tc := range tests {
		instr := Instr{Op: tc.op}
		if got := instr.String(); got != tc.want {
			t.Errorf("%s.String() = %q, want %q", tc.want, got, tc.want)
		}
	}
}

func TestInstrStringNumber(t *testing.T) {
	tests := []struct {
		op   Instruction
		n    uint16
		want string
	}{
		{PUSH_B, 42, "PUSH_B 42"},
		{PUSH_W, 65535, "PUSH_W 65535"},
		{AT, 0xC000, "AT 49152"},
		{ALLOC, 4, "ALLOC 4"},
		{FRAME, 6, "FRAME 6"},
		{PRIORITY, 4, "PRIORITY 4"},
		{BANK, 1, "BANK 1"},
		{DATA_B, 255, "DATA_B 255"},
		{DATA_W, 0x1234, "DATA_W 4660"},
		{PRAGMA, 1, "PRAGMA 1"},
	}
	for _, tc := range tests {
		instr := Instr{Op: tc.op, Operand: Operand{Type: OpNumber, Num: tc.n}}
		if got := instr.String(); got != tc.want {
			t.Errorf("Instr{%v, %d}.String() = %q, want %q", tc.op, tc.n, got, tc.want)
		}
	}
}

func TestInstrStringName(t *testing.T) {
	tests := []struct {
		op   Instruction
		name string
		want string
	}{
		{VAR, "x", "VAR x"},
		{GET_B, "x", "GET_B x"},
		{GET_W, "y", "GET_W y"},
		{PUT_B, "x", "PUT_B x"},
		{PUT_W, "y", "PUT_W y"},
		{PUSH_A, "arr", "PUSH_A arr"},
		{PUSH_D, "data", "PUSH_D data"},
		{TAG, "loop", "TAG loop"},
		{GO, "done", "GO done"},
		{GO_IF, "then", "GO_IF then"},
		{ROUTE, "foo", "ROUTE foo"},
		{RUN, "bar", "RUN bar"},
		{LOCAL_B, "tmp", "LOCAL_B tmp"},
		{LOCAL_W, "acc", "LOCAL_W acc"},
		{JOB, "music", "JOB music"},
		{STOP, "music", "STOP music"},
		{START, "music", "START music"},
		{INT, "tick", "INT tick"},
		{NMI, "pause", "NMI pause"},
	}
	for _, tc := range tests {
		instr := Instr{Op: tc.op, Operand: Operand{Type: OpName, Name: tc.name}}
		if got := instr.String(); got != tc.want {
			t.Errorf("Instr{%v, %s}.String() = %q, want %q", tc.op, tc.name, got, tc.want)
		}
	}
}

func TestInstrStringString(t *testing.T) {
	tests := []struct {
		op   Instruction
		s    string
		want string
	}{
		{DATA_STR, "hello", `DATA_STR "hello"`},
		{INLINE, "nop", `INLINE "nop"`},
	}
	for _, tc := range tests {
		instr := Instr{Op: tc.op, Operand: Operand{Type: OpString, Str: tc.s}}
		if got := instr.String(); got != tc.want {
			t.Errorf("Instr{%v, %q}.String() = %q, want %q", tc.op, tc.s, got, tc.want)
		}
	}
}

func TestInstrStringCondition(t *testing.T) {
	tests := []struct {
		op   Instruction
		cond Condition
		want string
	}{
		{IS_B, CondLT, "IS_B LT"},
		{IS_B, CondGT, "IS_B GT"},
		{IS_B, CondLE, "IS_B LE"},
		{IS_B, CondGE, "IS_B GE"},
		{IS_B, CondEQ, "IS_B EQ"},
		{IS_B, CondNE, "IS_B NE"},
		{IS_W, CondLT, "IS_W LT"},
	}
	for _, tc := range tests {
		instr := Instr{Op: tc.op, Operand: Operand{Type: OpCondition, Cond: tc.cond}}
		if got := instr.String(); got != tc.want {
			t.Errorf("Instr{%v, %v}.String() = %q, want %q", tc.op, tc.cond, got, tc.want)
		}
	}
}

func TestProgramString(t *testing.T) {
	prog := &Program{
		Instrs: []Instr{
			{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 5}},
			{Op: RUN, Operand: Operand{Type: OpName, Name: "fact"}},
			{Op: DONE},
		},
	}
	want := "PUSH_W 5\nRUN fact\nDONE\n"
	if got := prog.String(); got != want {
		t.Errorf("Program.String() = %q, want %q", got, want)
	}
}

func TestParseSimple(t *testing.T) {
	src := `PUSH_W 5
RUN fact
DONE
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Instrs) != 3 {
		t.Fatalf("got %d instrs, want 3", len(prog.Instrs))
	}
	if prog.Instrs[0].Op != PUSH_W || prog.Instrs[0].Operand.Num != 5 {
		t.Errorf("instr[0] = %v, want PUSH_W 5", prog.Instrs[0])
	}
	if prog.Instrs[1].Op != RUN || prog.Instrs[1].Operand.Name != "fact" {
		t.Errorf("instr[1] = %v, want RUN fact", prog.Instrs[1])
	}
	if prog.Instrs[2].Op != DONE {
		t.Errorf("instr[2] = %v, want DONE", prog.Instrs[2])
	}
}

func TestParseRoundTrip(t *testing.T) {
	src := `// Factorial in PIR
PUSH_W 5
RUN fact
DONE

ROUTE fact
FRAME 6
LOCAL_W n
LOCAL_W result
LOCAL_W i
PUT_W n
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got := prog.String()
	// Re-parse the output
	prog2, err := Parse(got)
	if err != nil {
		t.Fatalf("second Parse: %v", err)
	}
	got2 := prog2.String()
	if got != got2 {
		t.Errorf("round-trip mismatch:\nfirst:\n%s\nsecond:\n%s", got, got2)
	}
}

func TestParseAllOpcodes(t *testing.T) {
	for op, name := range instructionNames {
		src := name + "\n"
		switch expectedOperand(op) {
		case OpNumber:
			src = name + " 0\n"
		case OpName:
			src = name + " x\n"
		case OpString:
			src = name + ` "x"` + "\n"
		case OpCondition:
			src = name + " EQ\n"
		}
		prog, err := Parse(src)
		if err != nil {
			t.Errorf("Parse %q: %v", name, err)
			continue
		}
		if len(prog.Instrs) != 1 || prog.Instrs[0].Op != op {
			t.Errorf("Parse %q: got %v, want %v", name, prog.Instrs[0].Op, op)
		}
	}
}

func TestParseOperands(t *testing.T) {
	tests := []struct {
		src  string
		op   Instruction
		want string // expected String()
	}{
		{"PUSH_B 42", PUSH_B, "PUSH_B 42"},
		{"PUSH_W 0x1000", PUSH_W, "PUSH_W 4096"},
		{"GET_B x", GET_B, "GET_B x"},
		{"DATA_STR \"hello\"", DATA_STR, `DATA_STR "hello"`},
		{"INLINE \"nop\"", INLINE, `INLINE "nop"`},
		{"IS_B LT", IS_B, "IS_B LT"},
		{"IS_W EQ", IS_W, "IS_W EQ"},
	}
	for _, tc := range tests {
		prog, err := Parse(tc.src)
		if err != nil {
			t.Errorf("Parse %q: %v", tc.src, err)
			continue
		}
		got := prog.Instrs[0].String()
		if got != tc.want {
			t.Errorf("Parse(%q).String() = %q, want %q", tc.src, got, tc.want)
		}
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		src  string
		cont string // must contain
	}{
		{"BAD_INSTR", "unknown instruction"},
		{"ADD_B 42", "takes no operand"},
		{"PUSH_B", "needs a number"},
		{"GET_B", "needs a name"},
		{"IS_B", "needs a condition"},
		{"IS_B BAD", "unknown condition"},
		{"DATA_STR", "needs a string"},
		{"PUSH_B bad", "bad number"},
	}
	for _, tc := range tests {
		_, err := Parse(tc.src)
		if err == nil {
			t.Errorf("Parse(%q) succeeded, expected error containing %q", tc.src, tc.cont)
			continue
		}
		if !contains(err.Error(), tc.cont) {
			t.Errorf("Parse(%q) error = %q, want containing %q", tc.src, err.Error(), tc.cont)
		}
	}
}

func TestEmptyProgram(t *testing.T) {
	prog, err := Parse("")
	if err != nil {
		t.Fatalf("Parse empty: %v", err)
	}
	if len(prog.Instrs) != 0 {
		t.Errorf("expected 0 instrs, got %d", len(prog.Instrs))
	}
}

func TestCommentsOnly(t *testing.T) {
	src := "// just a comment\n// another\n"
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Instrs) != 0 {
		t.Errorf("expected 0 instrs, got %d", len(prog.Instrs))
	}
}

func TestHexNumber(t *testing.T) {
	prog, err := Parse("PUSH_W 0xABCD")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if prog.Instrs[0].Operand.Num != 0xABCD {
		t.Errorf("got %d, want 43981", prog.Instrs[0].Operand.Num)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
