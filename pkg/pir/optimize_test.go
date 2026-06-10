package pir

import (
	"testing"
)

func TestOptimizeEmpty(t *testing.T) {
	prog := &Program{}
	got := Optimize(prog)
	if len(got.Instrs) != 0 {
		t.Errorf("expected 0 instrs, got %d", len(got.Instrs))
	}
}

func TestOptimizeNOP(t *testing.T) {
	prog := &Program{Instrs: []Instr{{Op: NOP}}}
	got := Optimize(prog)
	if len(got.Instrs) != 0 {
		t.Errorf("expected NOP to be removed, got %d instrs", len(got.Instrs))
	}
}

// ── Constant folding ──

func TestFoldBinOpConst(t *testing.T) {
	tests := []struct {
		name  string
		prog  *Program
		want  string // expected String()
		count int    // expected instruction count after folding
	}{
		{"ADD_B const", mkprog(pushB(3), pushB(4), addB()), "PUSH_W 7\n", 1},
		{"ADD_W const", mkprog(pushW(100), pushW(200), addW()), "PUSH_W 300\n", 1},
		{"SUB_B const", mkprog(pushB(10), pushB(3), subB()), "PUSH_W 7\n", 1},
		{"SUB_W const", mkprog(pushW(255), pushW(1), subW()), "PUSH_W 254\n", 1},
		{"MUL_B const", mkprog(pushB(5), pushB(6), mulB()), "PUSH_W 30\n", 1},
		{"MUL_W const", mkprog(pushW(7), pushW(8), mulW()), "PUSH_W 56\n", 1},
		{"DIV_B const", mkprog(pushB(24), pushB(6), divB()), "PUSH_W 4\n", 1},
		{"DIV_W const", mkprog(pushW(100), pushW(25), divW()), "PUSH_W 4\n", 1},
		{"MOD_B const", mkprog(pushB(17), pushB(5), modB()), "PUSH_W 2\n", 1},
		{"MOD_W const", mkprog(pushW(100), pushW(30), modW()), "PUSH_W 10\n", 1},
		{"AND_B const", mkprog(pushB(0xAB), pushB(0x0F), andB()), "PUSH_W 11\n", 1},
		{"AND_W const", mkprog(pushW(0xFF00), pushW(0x0FFF), andW()), "PUSH_W 3840\n", 1},
		{"OR_B const", mkprog(pushB(0xF0), pushB(0x0F), orB()), "PUSH_W 255\n", 1},
		{"OR_W const", mkprog(pushW(0xFF00), pushW(0x00FF), orW()), "PUSH_W 65535\n", 1},
		{"XOR_B const", mkprog(pushB(0xFF), pushB(0x0F), xorB()), "PUSH_W 240\n", 1},
		{"XOR_W const", mkprog(pushW(0xFF00), pushW(0x0F0F), xorW()), "PUSH_W 61455\n", 1},
		{"SHL_B const", mkprog(pushB(3), pushB(2), shlB()), "PUSH_W 12\n", 1},
		{"SHL_W const", mkprog(pushW(0x00FF), pushW(8), shlW()), "PUSH_W 65280\n", 1},
		{"SHR_B const", mkprog(pushB(64), pushB(3), shrB()), "PUSH_W 8\n", 1},
		{"SHR_W const", mkprog(pushW(0xFF00), pushW(4), shrW()), "PUSH_W 4080\n", 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Optimize(tc.prog)
			if s := got.String(); s != tc.want {
				t.Errorf("Optimize:\ngot:\n%s\nwant:\n%s", s, tc.want)
			}
			if len(got.Instrs) != tc.count {
				t.Errorf("got %d instrs, want %d", len(got.Instrs), tc.count)
			}
		})
	}
}

func TestFoldUnaryOpConst(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want string
	}{
		{"NEG_B const", mkprog(pushB(1), negB()), "PUSH_W 255\n"},
		{"NEG_W const", mkprog(pushW(1), negW()), "PUSH_W 65535\n"},
		{"NOT_B const nonzero", mkprog(pushB(42), notB()), "PUSH_W 0\n"},
		{"NOT_B const zero", mkprog(pushB(0), notB()), "PUSH_W 1\n"},
		{"NOT_W const nonzero", mkprog(pushW(0x8000), notW()), "PUSH_W 0\n"},
		{"NOT_W const zero", mkprog(pushW(0), notW()), "PUSH_W 1\n"},
		{"CAST_B const", mkprog(pushW(0xABCD), castB()), "PUSH_W 205\n"},
		{"CAST_W const", mkprog(pushB(42), castW()), "PUSH_W 42\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Optimize(tc.prog)
			if s := got.String(); s != tc.want {
				t.Errorf("Optimize:\ngot:\n%s\nwant:\n%s", s, tc.want)
			}
		})
	}
}

// ── Identity elimination ──

func TestIdentityElimination(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want string // after optimization, the program string
	}{
		// x + 0
		{"ADD_B x 0", mkprog(getB("x"), pushB(0), addB()), "GET_B x\n"},
		{"ADD_W x 0", mkprog(getW("x"), pushW(0), addW()), "GET_W x\n"},
		// x - 0
		{"SUB_B x 0", mkprog(getB("x"), pushB(0), subB()), "GET_B x\n"},
		{"SUB_W x 0", mkprog(getW("x"), pushW(0), subW()), "GET_W x\n"},
		// x * 1
		{"MUL_B x 1", mkprog(getB("x"), pushB(1), mulB()), "GET_B x\n"},
		{"MUL_W x 1", mkprog(getW("x"), pushW(1), mulW()), "GET_W x\n"},
		// x * 0
		{"MUL_B x 0", mkprog(getB("x"), pushB(0), mulB()), "PUSH_W 0\n"},
		{"MUL_W x 0", mkprog(getW("x"), pushW(0), mulW()), "PUSH_W 0\n"},
		// x / 1
		{"DIV_B x 1", mkprog(getB("x"), pushB(1), divB()), "GET_B x\n"},
		{"DIV_W x 1", mkprog(getW("x"), pushW(1), divW()), "GET_W x\n"},
		// x % 1
		{"MOD_B x 1", mkprog(getB("x"), pushB(1), modB()), "PUSH_W 0\n"},
		{"MOD_W x 1", mkprog(getW("x"), pushW(1), modW()), "PUSH_W 0\n"},
		// x | 0
		{"OR_B x 0", mkprog(getB("x"), pushB(0), orB()), "GET_B x\n"},
		{"OR_W x 0", mkprog(getW("x"), pushW(0), orW()), "GET_W x\n"},
		// x & 0
		{"AND_B x 0", mkprog(getB("x"), pushB(0), andB()), "PUSH_W 0\n"},
		{"AND_W x 0", mkprog(getW("x"), pushW(0), andW()), "PUSH_W 0\n"},
		// x << 0
		{"SHL_B x 0", mkprog(getB("x"), pushB(0), shlB()), "GET_B x\n"},
		{"SHL_W x 0", mkprog(getW("x"), pushW(0), shlW()), "GET_W x\n"},
		// x >> 0
		{"SHR_B x 0", mkprog(getB("x"), pushB(0), shrB()), "GET_B x\n"},
		{"SHR_W x 0", mkprog(getW("x"), pushW(0), shrW()), "GET_W x\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Optimize(tc.prog)
			if s := got.String(); s != tc.want {
				t.Errorf("Optimize:\ngot:\n%s\nwant:\n%s", s, tc.want)
			}
		})
	}

	// Also check that identity with left=const works: 0 + x → 0 + x (not folded)
	t.Run("ADD_B 0 x not folded", func(t *testing.T) {
		prog := &Program{Instrs: []Instr{
			{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 0}},
			{Op: GET_B, Operand: Operand{Type: OpName, Name: "x"}},
			{Op: ADD_B},
		}}
		got := Optimize(prog)
		if len(got.Instrs) != 3 {
			t.Errorf("expected 3 instrs (no folding for 0 + unknown), got %d: %s", len(got.Instrs), got.String())
		}
	})
}

// ── Strength reduction ──

func TestStrengthReduceMulPow2(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want string // should become SHL
	}{
		{"MUL_B x 2", mkprog(getB("x"), pushB(2), mulB()), "GET_B x\nPUSH_B 1\nSHL_B\n"},
		{"MUL_W x 4", mkprog(getW("x"), pushW(4), mulW()), "GET_W x\nPUSH_B 2\nSHL_W\n"},
		{"MUL_W x 8", mkprog(getW("x"), pushW(8), mulW()), "GET_W x\nPUSH_B 3\nSHL_W\n"},
		{"MUL_B x 16", mkprog(getB("x"), pushB(16), mulB()), "GET_B x\nPUSH_B 4\nSHL_B\n"},
		{"MUL_W x 32", mkprog(getW("x"), pushW(32), mulW()), "GET_W x\nPUSH_B 5\nSHL_W\n"},
		{"MUL_W x 256", mkprog(getW("x"), pushW(256), mulW()), "GET_W x\nPUSH_B 8\nSHL_W\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Optimize(tc.prog)
			if s := got.String(); s != tc.want {
				t.Errorf("Optimize:\ngot:\n%s\nwant:\n%s", s, tc.want)
			}
		})
	}
}

func TestStrengthReduceDivPow2(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want string // should become SHR
	}{
		{"DIV_B x 2", mkprog(getB("x"), pushB(2), divB()), "GET_B x\nPUSH_B 1\nSHR_B\n"},
		{"DIV_W x 4", mkprog(getW("x"), pushW(4), divW()), "GET_W x\nPUSH_B 2\nSHR_W\n"},
		{"DIV_W x 16", mkprog(getW("x"), pushW(16), divW()), "GET_W x\nPUSH_B 4\nSHR_W\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Optimize(tc.prog)
			if s := got.String(); s != tc.want {
				t.Errorf("Optimize:\ngot:\n%s\nwant:\n%s", s, tc.want)
			}
		})
	}
}

func TestStrengthReduceModPow2(t *testing.T) {
	tests := []struct {
		name string
		prog *Program
		want string // should become AND
	}{
		{"MOD_B x 2", mkprog(getB("x"), pushB(2), modB()), "GET_B x\nPUSH_W 1\nAND_B\n"},
		{"MOD_W x 4", mkprog(getW("x"), pushW(4), modW()), "GET_W x\nPUSH_W 3\nAND_W\n"},
		{"MOD_W x 8", mkprog(getW("x"), pushW(8), modW()), "GET_W x\nPUSH_W 7\nAND_W\n"},
		{"MOD_B x 64", mkprog(getB("x"), pushB(64), modB()), "GET_B x\nPUSH_W 63\nAND_B\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Optimize(tc.prog)
			if s := got.String(); s != tc.want {
				t.Errorf("Optimize:\ngot:\n%s\nwant:\n%s", s, tc.want)
			}
		})
	}
}

// ── Cascading constant folding ──

func TestFoldCascade(t *testing.T) {
	// (10 + 20) * (5 - 3) → PUSH_W 60
	prog := &Program{Instrs: []Instr{
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 10}},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 20}},
		{Op: ADD_B},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 5}},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 3}},
		{Op: SUB_B},
		{Op: MUL_B},
	}}
	got := Optimize(prog)
	if len(got.Instrs) != 1 || got.Instrs[0].Op != PUSH_W || got.Instrs[0].Operand.Num != 60 {
		t.Errorf("expected PUSH_W 60, got %s", got.String())
	}
}

func TestFoldCascadeComplex(t *testing.T) {
	// ((40 / 5) << 2) | 1 → (8 << 2) | 1 → 32 | 1 → 33
	prog := &Program{Instrs: []Instr{
		{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 40}},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 5}},
		{Op: DIV_W},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 2}},
		{Op: SHL_W},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 1}},
		{Op: OR_W},
	}}
	got := Optimize(prog)
	if len(got.Instrs) != 1 || got.Instrs[0].Op != PUSH_W || got.Instrs[0].Operand.Num != 33 {
		t.Errorf("expected PUSH_W 33, got %s", got.String())
	}
}

// ── Non-foldable (unknown operands) ──

func TestSkipFoldWithNonConst(t *testing.T) {
	// x + y should NOT fold
	prog := &Program{Instrs: []Instr{
		{Op: GET_B, Operand: Operand{Type: OpName, Name: "x"}},
		{Op: GET_B, Operand: Operand{Type: OpName, Name: "y"}},
		{Op: ADD_B},
	}}
	got := Optimize(prog)
	if len(got.Instrs) != 3 {
		t.Errorf("expected 3 instrs unchanged, got %d: %s", len(got.Instrs), got.String())
	}
}

func TestSkipFoldPartialNonConst(t *testing.T) {
	// 5 + x should NOT fold (identity not applicable for ADD with 0 only)
	prog := &Program{Instrs: []Instr{
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 5}},
		{Op: GET_B, Operand: Operand{Type: OpName, Name: "x"}},
		{Op: ADD_B},
	}}
	got := Optimize(prog)
	if len(got.Instrs) != 3 {
		t.Errorf("expected 3 instrs unchanged (5 + x not foldable), got %d: %s", len(got.Instrs), got.String())
	}
}

// ── Non-power-of-2 not reduced ──

func TestSkipReduceNonPow2(t *testing.T) {
	// MUL by 3 should NOT reduce
	prog := &Program{Instrs: []Instr{
		{Op: GET_B, Operand: Operand{Type: OpName, Name: "x"}},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 3}},
		{Op: MUL_B},
	}}
	got := Optimize(prog)
	if len(got.Instrs) != 3 {
		t.Errorf("expected 3 instrs (non-pow2), got %d: %s", len(got.Instrs), got.String())
	}
}

// ── DUP/DROP/SWAP ──

func TestFoldThroughDUP(t *testing.T) {
	// DUP duplicates known constant
	prog := &Program{Instrs: []Instr{
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 5}},
		{Op: DUP},
		{Op: ADD_B},
	}}
	got := Optimize(prog)
	if len(got.Instrs) != 1 || got.Instrs[0].Op != PUSH_W || got.Instrs[0].Operand.Num != 10 {
		t.Errorf("expected PUSH_W 10, got %s", got.String())
	}
}

func TestFoldSWAPConst(t *testing.T) {
	// 10 20 SWAP -> 20 10; then SUB -> 10
	// SWAP is preserved (structural instruction), but the PUSH values
	// and SUB are folded.
	prog := &Program{Instrs: []Instr{
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 10}},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 20}},
		{Op: SWAP},
		{Op: SUB_B},
	}}
	want := "SWAP\nPUSH_W 10\n"
	got := Optimize(prog)
	if s := got.String(); s != want {
		t.Errorf("Optimize:\ngot:\n%s\nwant:\n%s", s, want)
	}
}

func TestDropClearsState(t *testing.T) {
	// PUSH 5; DROP; PUSH 7 → PUSH 5 and DROP survive (optimizer does not
	// eliminate dead push-drop pairs), but PUSH 7 is the only other value.
	prog := &Program{Instrs: []Instr{
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 5}},
		{Op: DROP},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 7}},
	}}
	got := Optimize(prog)
	// DROP correctly tracks stack state: after DROP, stack is empty,
	// then PUSH 7 is pushed. No folding occurs though.
	if len(got.Instrs) != 3 {
		t.Errorf("expected 3 instrs (dead push/drop not eliminated), got %d: %s", len(got.Instrs), got.String())
	}
}

func TestDropAfterNonConst(t *testing.T) {
	// GET x; DROP; PUSH 5 → GET and DROP survive (optimizer does not
	// eliminate dead code), PUSH 5 is added afterward.
	prog := &Program{Instrs: []Instr{
		{Op: GET_B, Operand: Operand{Type: OpName, Name: "x"}},
		{Op: DROP},
		{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: 5}},
	}}
	got := Optimize(prog)
	if len(got.Instrs) != 3 {
		t.Errorf("expected 3 instrs, got %d: %s", len(got.Instrs), got.String())
	}
}

// ── Unchanged instructions ──

func TestUnknownPreventsFolding(t *testing.T) {
	// An unrecognized opcode (e.g. HLT) calls unknown() which marks all
	// stack entries as unknown, preventing folding of subsequent operations.
	prog := &Program{Instrs: []Instr{
		{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 100}},
		{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: 200}},
		{Op: HLT},
		{Op: ADD_W},
	}}
	got := Optimize(prog)
	// After unknown(), ADD_W sees unknown operands and should not fold.
	// HLT itself is preserved.
	if len(got.Instrs) != 4 {
		t.Errorf("expected 4 instrs preserved, got %d: %s", len(got.Instrs), got.String())
	}
	if got.Instrs[2].Op != HLT || got.Instrs[3].Op != ADD_W {
		t.Errorf("expected HLT then ADD_W, got %s", got.String())
	}
}

// ── isPow2 / log2 helpers ──

func TestIsPow2(t *testing.T) {
	tests := []struct {
		n    uint16
		want bool
	}{
		{0, false},
		{1, true},
		{2, true},
		{3, false},
		{4, true},
		{255, false},
		{256, true},
		{32768, true},
	}
	for _, tc := range tests {
		if got := isPow2(tc.n); got != tc.want {
			t.Errorf("isPow2(%d) = %v, want %v", tc.n, got, tc.want)
		}
	}
}

func TestLog2(t *testing.T) {
	tests := []struct {
		n    uint16
		want uint16
	}{
		{1, 0},
		{2, 1},
		{4, 2},
		{8, 3},
		{16, 4},
		{256, 8},
		{32768, 15},
	}
	for _, tc := range tests {
		if got := log2(tc.n); got != tc.want {
			t.Errorf("log2(%d) = %d, want %d", tc.n, got, tc.want)
		}
	}
}

// ── Helpers ──
// Note: these use lowercase short names to avoid conflicting with pir.go
// constants (e.g. PUSH_B, ADD_W) and gen6502_test.go's prog helper.

func mkprog(instrs ...Instr) *Program {
	return &Program{Instrs: instrs}
}

func pushW(n uint16) Instr {
	return Instr{Op: PUSH_W, Operand: Operand{Type: OpNumber, Num: n}}
}

func pushB(n uint16) Instr {
	return Instr{Op: PUSH_B, Operand: Operand{Type: OpNumber, Num: n}}
}

func getB(name string) Instr {
	return Instr{Op: GET_B, Operand: Operand{Type: OpName, Name: name}}
}

func getW(name string) Instr {
	return Instr{Op: GET_W, Operand: Operand{Type: OpName, Name: name}}
}

func addB() Instr { return Instr{Op: ADD_B} }
func addW() Instr { return Instr{Op: ADD_W} }
func subB() Instr { return Instr{Op: SUB_B} }
func subW() Instr { return Instr{Op: SUB_W} }
func mulB() Instr { return Instr{Op: MUL_B} }
func mulW() Instr { return Instr{Op: MUL_W} }
func divB() Instr { return Instr{Op: DIV_B} }
func divW() Instr { return Instr{Op: DIV_W} }
func modB() Instr { return Instr{Op: MOD_B} }
func modW() Instr { return Instr{Op: MOD_W} }
func andB() Instr { return Instr{Op: AND_B} }
func andW() Instr { return Instr{Op: AND_W} }
func orB() Instr  { return Instr{Op: OR_B} }
func orW() Instr  { return Instr{Op: OR_W} }
func xorB() Instr { return Instr{Op: XOR_B} }
func xorW() Instr { return Instr{Op: XOR_W} }
func shlB() Instr { return Instr{Op: SHL_B} }
func shlW() Instr { return Instr{Op: SHL_W} }
func shrB() Instr { return Instr{Op: SHR_B} }
func shrW() Instr { return Instr{Op: SHR_W} }
func negB() Instr { return Instr{Op: NEG_B} }
func negW() Instr { return Instr{Op: NEG_W} }
func notB() Instr { return Instr{Op: NOT_B} }
func notW() Instr { return Instr{Op: NOT_W} }
func castB() Instr { return Instr{Op: CAST_B} }
func castW() Instr { return Instr{Op: CAST_W} }
