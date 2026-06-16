package pir

import (
	"fmt"
	"strings"
	"testing"
)

func TestZ80GenDataMovement(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want []string // strings that must appear in output
	}{
		{
			name: "NOP",
			prog: &Program{Instrs: []Instr{{Op: NOP}}},
			want: []string{"\tnop"},
		},
		{
			name: "PUSH_B",
			prog: &Program{Instrs: []Instr{{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 42}}}},
			want: []string{"ld (hl), e", "inc hl", "ld (hl), d", "inc hl", "ld e, 42", "ld d, 0"},
		},
		{
			name: "PUSH_W",
			prog: &Program{Instrs: []Instr{{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 0x1000}}}},
			want: []string{"ld (hl), e", "inc hl", "ld (hl), d", "inc hl", "ld de, 4096"},
		},
		{
			name: "GET_B",
			prog: &Program{Instrs: []Instr{{Op: GET_B, Operand: Operand{Type: OpName, Name: "x"}}}},
			want: []string{"ld (hl), e", "ld a, (_v_x)", "ld e, a", "ld d, 0"},
		},
		{
			name: "GET_B",
			prog: &Program{Instrs: []Instr{{Op: GET_B, Operand: Operand{Type: OpName, Name: "x"}}}},
			want: []string{"ld a, (_v_x)"},
		},
		{
			name: "GET_W",
			prog: &Program{Instrs: []Instr{{Op: GET_W, Operand: Operand{Type: OpName, Name: "y"}}}},
			want: []string{"ld de, (_v_y)"},
		},
		{
			name: "PUT_B",
			prog: &Program{Instrs: []Instr{{Op: PUT_B, Operand: Operand{Type: OpName, Name: "x"}}}},
			want: []string{"ld (_v_x), a"},
		},
		{
			name: "PUT_W",
			prog: &Program{Instrs: []Instr{{Op: PUT_W, Operand: Operand{Type: OpName, Name: "y"}}}},
			want: []string{"ld (_v_y), de"},
		},
		{
			name: "PUT_B",
			prog: &Program{Instrs: []Instr{{Op: PUT_B, Operand: Operand{Type: OpName, Name: "x"}}}},
			want: []string{"ld a, e", "ld (_v_x), a", "dec hl", "ld d, (hl)", "dec hl", "ld e, (hl)"},
		},
		{
			name: "PUT_W",
			prog: &Program{Instrs: []Instr{{Op: PUT_W, Operand: Operand{Type: OpName, Name: "y"}}}},
			want: []string{"ld (_v_y), de", "dec hl", "ld d, (hl)"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gen Z80Gen
			out := gen.Gen(tc.prog)
			for _, s := range tc.want {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q\nFull output:\n%s", s, out)
				}
			}
		})
	}
}

func TestZ80GenPointers(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want []string
	}{
		{
			name: "PUSH_A",
			prog: &Program{Instrs: []Instr{{Op: PUSH_A, Operand: Operand{Type: OpName, Name: "arr"}}}},
			want: []string{"ld de, _v_arr"},
		},
		{
			name: "PUSH_D",
			prog: &Program{Instrs: []Instr{{Op: PUSH_D, Operand: Operand{Type: OpName, Name: "mydata"}}}},
			want: []string{"ld de, mydata"},
		},
		{
			name: "READ_B",
			prog: &Program{Instrs: []Instr{{Op: READ_B}}},
			want: []string{"ld a, (de)", "ld e, a", "ld d, 0"},
		},
		{
			name: "READ_W",
			prog: &Program{Instrs: []Instr{{Op: READ_W}}},
			want: []string{"push hl", "ex de, hl", "ld e, (hl)", "inc hl", "ld d, (hl)", "pop hl"},
		},
		{
			name: "WRITE_B",
			prog: &Program{Instrs: []Instr{{Op: WRITE_B}}},
			want: []string{"ld a, e", "dec hl", "ld d, (hl)", "dec hl", "ld e, (hl)", "ld (de), a"},
		},
		{
			name: "WRITE_W",
			prog: &Program{Instrs: []Instr{{Op: WRITE_W}}},
			want: []string{"push hl", "ld b, d", "ld c, e", "pop hl", "ld (hl), a"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gen Z80Gen
			out := gen.Gen(tc.prog)
			for _, s := range tc.want {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q\nFull output:\n%s", s, out)
				}
			}
		})
	}
}

func TestZ80GenMath(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want []string
	}{
		{name: "ADD_B", prog: &Program{Instrs: []Instr{{Op: ADD_B}}}, want: []string{"ld a, e", "dec hl", "ld d, (hl)", "dec hl", "ld e, (hl)", "add a, e"}},
		{name: "ADD_W", prog: &Program{Instrs: []Instr{{Op: ADD_W}}}, want: []string{"ld b, d", "ld c, e", "add hl, de", "ex de, hl"}},
		{name: "SUB_B", prog: &Program{Instrs: []Instr{{Op: SUB_B}}}, want: []string{"ld b, e", "sub a, b"}},
		{name: "SUB_W", prog: &Program{Instrs: []Instr{{Op: SUB_W}}}, want: []string{"ld b, d", "ld c, e", "sbc hl, de", "ex de, hl"}},
		{name: "MUL_B", prog: &Program{Instrs: []Instr{{Op: MUL_B}}}, want: []string{"call _plz_mul"}},
		{name: "MUL_W", prog: &Program{Instrs: []Instr{{Op: MUL_W}}}, want: []string{"call _plz_mul"}},
		{name: "DIV_B", prog: &Program{Instrs: []Instr{{Op: DIV_B}}}, want: []string{"call _plz_div"}},
		{name: "DIV_W", prog: &Program{Instrs: []Instr{{Op: DIV_W}}}, want: []string{"call _plz_div"}},
		{name: "MOD_B", prog: &Program{Instrs: []Instr{{Op: MOD_B}}}, want: []string{"call _plz_mod"}},
		{name: "MOD_W", prog: &Program{Instrs: []Instr{{Op: MOD_W}}}, want: []string{"call _plz_mod"}},
		{name: "AND_B", prog: &Program{Instrs: []Instr{{Op: AND_B}}}, want: []string{"ld a, e", "and e"}},
		{name: "AND_W", prog: &Program{Instrs: []Instr{{Op: AND_W}}}, want: []string{"ld b, d", "ld c, e", "and c", "and b"}},
		{name: "OR_B", prog: &Program{Instrs: []Instr{{Op: OR_B}}}, want: []string{"ld a, e", "or e"}},
		{name: "OR_W", prog: &Program{Instrs: []Instr{{Op: OR_W}}}, want: []string{"ld b, d", "ld c, e", "or c", "or b"}},
		{name: "XOR_B", prog: &Program{Instrs: []Instr{{Op: XOR_B}}}, want: []string{"ld a, e", "xor e"}},
		{name: "XOR_W", prog: &Program{Instrs: []Instr{{Op: XOR_W}}}, want: []string{"ld b, d", "ld c, e", "xor c", "xor b"}},
		{name: "NEG_B", prog: &Program{Instrs: []Instr{{Op: NEG_B}}}, want: []string{"neg"}},
		{name: "NEG_W", prog: &Program{Instrs: []Instr{{Op: NEG_W}}}, want: []string{"cpl", "cpl", "inc hl"}},
		{name: "NOT_B", prog: &Program{Instrs: []Instr{{Op: NOT_B}}}, want: []string{"ld a, e", "or a", "ld e, 0", "jr nz", "inc e"}},
		{name: "NOT_W", prog: &Program{Instrs: []Instr{{Op: NOT_W}}}, want: []string{"ld a, h", "or l", "ld de, 0", "jr nz", "inc de"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gen Z80Gen
			out := gen.Gen(tc.prog)
			for _, s := range tc.want {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q\nFull output:\n%s", s, out)
				}
			}
		})
	}
}

func TestZ80GenCastStack(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want []string
	}{
		{name: "CAST_W", prog: &Program{Instrs: []Instr{{Op: CAST_W}}}, want: []string{"ld d, 0"}},
		{name: "CAST_B", prog: &Program{Instrs: []Instr{{Op: CAST_B}}}, want: []string{"ld d, 0"}},
		{name: "DUP", prog: &Program{Instrs: []Instr{{Op: DUP}}}, want: []string{"ld (hl), e", "inc hl", "ld (hl), d", "inc hl"}},
		{name: "DROP", prog: &Program{Instrs: []Instr{{Op: DROP}}}, want: []string{"dec hl", "ld d, (hl)", "dec hl", "ld e, (hl)"}},
		{name: "SWAP", prog: &Program{Instrs: []Instr{{Op: SWAP}}}, want: []string{"ld b, e", "ld c, d"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gen Z80Gen
			out := gen.Gen(tc.prog)
			for _, s := range tc.want {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q\nFull output:\n%s", s, out)
				}
			}
		})
	}
}

func TestZ80GenComparison(t *testing.T) {
	conds := []struct {
		name string
		cond Condition
	}{
		{"LT", CondLT}, {"GT", CondGT}, {"LE", CondLE},
		{"GE", CondGE}, {"EQ", CondEQ}, {"NE", CondNE},
	}
	// helperName maps our Condition names to the actual runtime helper names
	helperName := map[Condition]string{
		CondLT: "lt",
		CondGT: "gt",
		CondLE: "lte",
		CondGE: "gte",
		CondEQ: "eq",
		CondNE: "ne",
	}
	for _, c := range conds {
		t.Run("IS_B_"+c.name, func(t *testing.T) {
			var gen Z80Gen
			prog := &Program{Instrs: []Instr{{Op: IS_B, Operand: Operand{Type: OpCondition, Cond: c.cond}}}}
			out := gen.Gen(prog)
			if !strings.Contains(out, "call _plz_") {
				t.Errorf("missing call to helper\n%s", out)
			}
			if !strings.Contains(out, "call _plz_"+helperName[c.cond]) {
				t.Errorf("expected call _plz_%s", helperName[c.cond])
			}
		})
		t.Run("IS_W_"+c.name, func(t *testing.T) {
			var gen Z80Gen
			prog := &Program{Instrs: []Instr{{Op: IS_W, Operand: Operand{Type: OpCondition, Cond: c.cond}}}}
			out := gen.Gen(prog)
			if !strings.Contains(out, "call _plz_"+helperName[c.cond]) {
				t.Errorf("expected call _plz_%s\n%s", helperName[c.cond], out)
			}
		})
	}
}

func TestZ80GenControlFlow(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want []string
	}{
		{
			name: "TAG",
			prog: &Program{Instrs: []Instr{{Op: TAG, Operand: Operand{Type: OpName, Name: "loop"}}}},
			want: []string{"loop:"},
		},
		{
			name: "GO",
			prog: &Program{Instrs: []Instr{{Op: GO, Operand: Operand{Type: OpName, Name: "done"}}}},
			want: []string{"jmp done"},
		},
		{
			name: "GO_IF",
			prog: &Program{Instrs: []Instr{{Op: GO_IF, Operand: Operand{Type: OpName, Name: "then"}}}},
			want: []string{"dec hl", "ld d, (hl)", "dec hl", "ld e, (hl)", "ld a, e", "or a", "jp nz, then"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gen Z80Gen
			out := gen.Gen(tc.prog)
			for _, s := range tc.want {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q\nFull output:\n%s", s, out)
				}
			}
		})
	}
}

func TestZ80GenProcedures(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: ROUTE, Operand: Operand{Type: OpName, Name: "fact"}},
		{Op: FRAME, Operand: Operand{Type: OpNumber, Num: 6}},
		{Op: LOCAL_W, Operand: Operand{Type: OpName, Name: "n"}},
		{Op: LOCAL_W, Operand: Operand{Type: OpName, Name: "result"}},
		{Op: LOCAL_W, Operand: Operand{Type: OpName, Name: "i"}},
		{Op: PUT_W, Operand: Operand{Type: OpName, Name: "n"}},
		{Op: RUN, Operand: Operand{Type: OpName, Name: "other"}},
		{Op: DONE},
	}}
	want := []string{
		"fact:",
		"push ix",
		"ld ix, 0",
		"add ix, sp",
		"ld hl, -6",
		"add hl, sp",
		"ld sp, hl",
		"ld (ix), a",
		"ld (ix+1), a",
		"call _plz_other",
		"ld sp, ix",
		"pop ix",
		"ret",
	}
	var gen Z80Gen
	out := gen.Gen(prog)
	for _, s := range want {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q\nFull output:\n%s", s, out)
		}
	}
}

func TestZ80GenDoneInterrupt(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: ROUTE, Operand: Operand{Type: OpName, Name: "tick"}},
		{Op: DONE_INTERRUPT},
	}}
	var gen Z80Gen
	out := gen.Gen(prog)
	if !strings.Contains(out, "reti") {
		t.Errorf("expected reti\n%s", out)
	}
}

func TestZ80GenDoneNMI(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: ROUTE, Operand: Operand{Type: OpName, Name: "pause"}},
		{Op: DONE_NMI},
	}}
	var gen Z80Gen
	out := gen.Gen(prog)
	if !strings.Contains(out, "retn") {
		t.Errorf("expected retn\n%s", out)
	}
}

func TestZ80GenTasks(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: JOB, Operand: Operand{Type: OpName, Name: "music"}},
		{Op: BYE},
		{Op: JOB, Operand: Operand{Type: OpName, Name: "input"}},
		{Op: SLEEP},
	}}
	var gen Z80Gen
	out := gen.Gen(prog)
	want := []string{
		"music:",
		"jp _plz_scheduler",
		"input:",
		"_plz_scheduler:",
		"_plz_tcbs: ds 128",
		"_plz_task0_stack: ds 128",
		"_plz_task1_stack: ds 128",
	}
	for _, s := range want {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q\nFull output:\n%s", s, out)
		}
	}
}

func TestZ80GenPortIO(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want []string
	}{
		{
			name: "IN_B",
			prog: &Program{Instrs: []Instr{{Op: IN_B, Operand: Operand{Type: OpNumber, Num: 0xBE}}}},
			want: []string{"in a, (190)", "ld e, a", "ld d, 0"},
		},
		{
			name: "IN_W",
			prog: &Program{Instrs: []Instr{{Op: IN_W, Operand: Operand{Type: OpNumber, Num: 0xBE}}}},
			want: []string{"in a, (190)", "in a, (191)", "ld d, a"},
		},
		{
			name: "OUT_B",
			prog: &Program{Instrs: []Instr{{Op: OUT_B, Operand: Operand{Type: OpNumber, Num: 0xBE}}}},
			want: []string{"ld a, e", "out (190), a", "dec hl"},
		},
		{
			name: "OUT_W",
			prog: &Program{Instrs: []Instr{{Op: OUT_W, Operand: Operand{Type: OpNumber, Num: 0xBE}}}},
			want: []string{"ld a, e", "out (190), a", "ld a, d", "out (190), a"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gen Z80Gen
			out := gen.Gen(tc.prog)
			for _, s := range tc.want {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q\nFull output:\n%s", s, out)
				}
			}
		})
	}
}

func TestZ80GenInterrupts(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want []string
	}{
		{name: "INT", prog: &Program{Instrs: []Instr{{Op: INT, Operand: Operand{Type: OpName, Name: "tick"}}}}, want: []string{"org 0x0038", "jp _plz_tick"}},
		{name: "NMI", prog: &Program{Instrs: []Instr{{Op: NMI, Operand: Operand{Type: OpName, Name: "pause"}}}}, want: []string{"org 0x0066", "jp _plz_pause"}},
		{name: "HLT", prog: &Program{Instrs: []Instr{{Op: HLT}}}, want: []string{"halt"}},
		{name: "DII", prog: &Program{Instrs: []Instr{{Op: DII}}}, want: []string{"di"}},
		{name: "ENI", prog: &Program{Instrs: []Instr{{Op: ENI}}}, want: []string{"ei"}},
		{name: "SEED", prog: &Program{Instrs: []Instr{{Op: SEED}}}, want: []string{"ld a, r", "ld e, a", "ld d, 0"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gen Z80Gen
			out := gen.Gen(tc.prog)
			for _, s := range tc.want {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q\nFull output:\n%s", s, out)
				}
			}
		})
	}
}

func TestZ80GenBankData(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want []string
	}{
		{name: "BANK", prog: &Program{Instrs: []Instr{{Op: BANK, Operand: Operand{Type: OpNumber, Num: 1}}}}, want: []string{"bank 1"}},
		{name: "SWITCH", prog: &Program{Instrs: []Instr{{Op: SWITCH}}}, want: []string{"ld a, e", "out (0xfffd), a"}},
		{name: "DATA_B", prog: &Program{Instrs: []Instr{{Op: DATA_B, Operand: Operand{Type: OpNumber, Num: 255}}}}, want: []string{"db 255"}},
		{name: "DATA_W", prog: &Program{Instrs: []Instr{{Op: DATA_W, Operand: Operand{Type: OpNumber, Num: 0x1234}}}}, want: []string{"dw 4660"}},
		{name: "SRAM_ON", prog: &Program{Instrs: []Instr{{Op: SRAM_ON}}}, want: []string{"ld a, 8", "ld (0xfffc), a", "ld (hl), e", "inc hl", "ld (hl), d", "inc hl", "ld de, 0x8000"}},
		{name: "SRAM_OFF", prog: &Program{Instrs: []Instr{{Op: SRAM_OFF}}}, want: []string{"ld a, 0", "ld (0xfffc), a"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gen Z80Gen
			out := gen.Gen(tc.prog)
			for _, s := range tc.want {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q\nFull output:\n%s", s, out)
				}
			}
		})
	}
}

func TestZ80GenDataStr(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: TAG, Operand: Operand{Type: OpName, Name: "msg"}},
		{Op: DATA_STR, Operand: Operand{Type: OpString, Str: "hello"}},
	}}
	var gen Z80Gen
	out := gen.Gen(prog)
	if !strings.Contains(out, "db 5") {
		t.Errorf("expected string length prefix\n%s", out)
	}
	for _, ch := range "hello" {
		if !strings.Contains(out, fmt.Sprintf("db %d", byte(ch))) {
			t.Errorf("expected db %d\n%s", byte(ch), out)
		}
	}
}

func TestZ80GenInline(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: INLINE, Operand: Operand{Type: OpString, Str: "nop"}},
	}}
	var gen Z80Gen
	out := gen.Gen(prog)
	if !strings.Contains(out, "\tnop") {
		t.Errorf("expected inline nop\n%s", out)
	}
}

func TestZ80GenVars(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: VAR, Operand: Operand{Type: OpName, Name: "x"}},
		{Op: ALLOC, Operand: Operand{Type: OpNumber, Num: 2}},
		{Op: VAR, Operand: Operand{Type: OpName, Name: "y"}},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 1}},
	}}
	var gen Z80Gen
	out := gen.Gen(prog)
	if !strings.Contains(out, "x: ds") {
		t.Errorf("expected x variable declaration")
	}
	if !strings.Contains(out, "y: ds") {
		t.Errorf("expected y variable declaration")
	}
}

func TestZ80GenProgramStructure(t *testing.T) {
	prog := &Program{Instrs: []Instr{{Op: NOP}}}
	var gen Z80Gen
	gen.cfg = DefaultConfig()
	out := gen.Gen(prog)
	structure := []string{
		"org 0x0000",
		"main:",
		"di",
		"im 1",
		"ld sp, 57328", // 0xDFF0
		"_plz_all_done:",
		"halt",
	}
	for _, s := range structure {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q\nFull output:\n%s", s, out)
		}
	}
}

func TestZ80GenRuntimeHelpers(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: MUL_W},
		{Op: DIV_W},
		{Op: MOD_W},
		{Op: IS_W, Operand: Operand{Type: OpCondition, Cond: CondLT}},
	}}
	var gen Z80Gen
	out := gen.Gen(prog)
	helpers := []string{
		"_plz_mul:",
		"_plz_div:",
		"_plz_divmod:",
		"_plz_mod:",
		"_plz_eq:",
		"_plz_ne:",
		"_plz_lt:",
		"_plz_gt:",
		"_plz_lte:",
		"_plz_gte:",
	}
	for _, s := range helpers {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q", s)
		}
	}
}

func TestZ80GenConfig(t *testing.T) {
	cfg := Z80Config{StackBase: 0xFF00, HeapBase: 0xE000}
	prog := &Program{Instrs: []Instr{
		{Op: VAR, Operand: Operand{Type: OpName, Name: "x"}},
		{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 0}},
	}}
	var gen Z80Gen
	gen.cfg = cfg
	out := gen.Gen(prog)
	if !strings.Contains(out, "ld sp, 65280") { // 0xFF00
		t.Errorf("expected custom stack base\n%s", out)
	}
	// x should be at HeapBase (0xE000)
	if !strings.Contains(out, "x: ds") {
		t.Errorf("expected x declaration\n%s", out)
	}
}

func TestZ80GenDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.StackBase != 0xDFF0 {
		t.Errorf("StackBase = %d, want 57328", cfg.StackBase)
	}
	if cfg.HeapBase != 0xC000 {
		t.Errorf("HeapBase = %d, want 49152", cfg.HeapBase)
	}
}

func TestZ80GenSaveLoad(t *testing.T) {
	prog := &Program{Instrs: []Instr{
		{Op: SAVE},
		{Op: LOAD},
	}}
	var gen Z80Gen
	out := gen.Gen(prog)
	if !strings.Contains(out, "_plz_save:") {
		t.Errorf("expected _plz_save helper\n%s", out)
	}
	if !strings.Contains(out, "_plz_load:") {
		t.Errorf("expected _plz_load helper\n%s", out)
	}
	if !strings.Contains(out, "call _plz_save") {
		t.Errorf("expected call _plz_save\n%s", out)
	}
	if !strings.Contains(out, "call _plz_load") {
		t.Errorf("expected call _plz_load\n%s", out)
	}
}

func TestZ80GenShift(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want string
	}{
		{name: "SHL_B", prog: &Program{Instrs: []Instr{{Op: SHL_B}}}, want: "sla a"},
		{name: "SHL_W", prog: &Program{Instrs: []Instr{{Op: SHL_W}}}, want: "add hl, hl"},
		{name: "SHR_B", prog: &Program{Instrs: []Instr{{Op: SHR_B}}}, want: "srl a"},
		{name: "SHR_W", prog: &Program{Instrs: []Instr{{Op: SHR_W}}}, want: "srl h"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gen Z80Gen
			out := gen.Gen(tc.prog)
			if !strings.Contains(out, tc.want) {
				t.Errorf("expected %q\n%s", tc.want, out)
			}
		})
	}
}

// ── Full program tests ──────────────────────────────────────────────

func TestZ80GenFullAdd(t *testing.T) {
	// Minimal: compute 3 + 4 and store to result
	src := `PUSH_W 4
PUSH_W 3
ADD_W
PUT_W result
TAG halt
HLT
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	var gen Z80Gen
	out := gen.Gen(prog)
	// Should have the assembly for ADD_W sequence
	if !strings.Contains(out, "add hl, de") {
		t.Errorf("expected add hl, de\n%s", out)
	}
	if !strings.Contains(out, "halt") {
		t.Errorf("expected halt\n%s", out)
	}
}

func TestZ80GenFullFactorial(t *testing.T) {
	src := `PUSH_W 5
RUN fact
DONE

ROUTE fact
FRAME 6
LOCAL_W n
LOCAL_W result
LOCAL_W i
PUT_W n
GET_W n
PUSH_W 1
IS_W LE
GO_IF base
PUSH_W 1
PUT_W result
PUSH_W 2
PUT_W i
TAG loop
GET_W i
GET_W n
IS_W GT
GO_IF done
GET_W result
GET_W i
MUL_W
PUT_W result
GET_W i
PUSH_W 1
ADD_W
PUT_W i
GO loop
TAG base
PUSH_W 1
DONE
TAG done
GET_W result
DONE
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	var gen Z80Gen
	out := gen.Gen(prog)
	key := []string{
		"fact:",
		"push ix",
		"call _plz_mul",
		"jmp loop",
		"ret",
	}
	for _, s := range key {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q\n%s", s, out)
		}
	}
}

func TestZ80GenRoundTrip(t *testing.T) {
	// Parse a complex program, generate Z80, verify structure
	src := `VAR x
ALLOC 2
VAR y
PUSH_B 42
PUT_B x
PUSH_W 1000
PUT_W y
GET_B x
GET_W y
ADD_W
PUT_W y
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	var gen Z80Gen
	out := gen.Gen(prog)
	check := []string{
		"_v_x: ds",
		"_v_y: ds",
		"ld a, e",
		"ld (_v_x), a",
		"ld (_v_y), de",
	}
	for _, s := range check {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q\n%s", out, s)
		}
	}
}
