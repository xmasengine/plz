package plz

import (
	"os"
	"strings"
	"testing"
)

// genTest helper: parse a PL/Z program, generate assembly, return the text.
func genTest(t *testing.T, src string) string {
	t.Helper()
	tokens, err := Scan(strings.NewReader(src))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	prog := Program{}
	parser := NewParser(tokens)
	if err := prog.Parse(parser); err != nil {
		t.Fatalf("parse: %v", err)
	}

	f, err := os.CreateTemp("", "plz_gen_*.asm")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	// Reopen for writing
	fw, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	g := NewGen(fw)
	if err := prog.Gen(g); err != nil {
		t.Fatalf("gen: %v", err)
	}
	fw.Close()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(body)
}

func TestGenLetNumber(t *testing.T) {
	asm := genTest(t, "LET x = 42")
	if !strings.Contains(asm, "ld hl, 42") {
		t.Error("expected ld hl, 42")
	}
	if !strings.Contains(asm, "ld (x), hl") {
		t.Error("expected ld (x), hl")
	}
}

func TestGenLetVar(t *testing.T) {
	asm := genTest(t, "DECLARE y WORD\nLET x = y")
	if !strings.Contains(asm, "ld hl, (y)") {
		t.Error("expected ld hl, (y)")
	}
	if !strings.Contains(asm, "ld (x), hl") {
		t.Error("expected ld (x), hl")
	}
}

func TestGenLetAdd(t *testing.T) {
	asm := genTest(t, "DECLARE a WORD\nDECLARE b WORD\nLET x = a + b")
	if !strings.Contains(asm, "add hl, de") {
		t.Error("expected add hl, de")
	}
	if !strings.Contains(asm, "ld (x), hl") {
		t.Error("expected ld (x), hl")
	}
}

func TestGenRuntimeHeader(t *testing.T) {
	asm := genTest(t, "HALT")
	if !strings.Contains(asm, "_plz_mul:") {
		t.Error("expected _plz_mul runtime function")
	}
	if !strings.Contains(asm, "_plz_div:") {
		t.Error("expected _plz_div runtime function")
	}
	if !strings.Contains(asm, "_plz_eq:") {
		t.Error("expected _plz_eq runtime function")
	}
	if !strings.Contains(asm, "_plz_gt:") {
		t.Error("expected _plz_gt runtime function")
	}
	if !strings.Contains(asm, "_plz_start:") {
		t.Error("expected _plz_start label")
	}
}

func TestGenLetMul(t *testing.T) {
	asm := genTest(t, "DECLARE a WORD\nDECLARE b WORD\nLET x = a * b")
	if !strings.Contains(asm, "call _plz_mul") {
		t.Error("expected call _plz_mul")
	}
}

func TestGenLetDiv(t *testing.T) {
	asm := genTest(t, "DECLARE a WORD\nDECLARE b WORD\nLET x = a / b")
	if !strings.Contains(asm, "call _plz_div") {
		t.Error("expected call _plz_div")
	}
}

func TestGenLetMod(t *testing.T) {
	asm := genTest(t, "DECLARE a WORD\nDECLARE b WORD\nLET x = a % b")
	if !strings.Contains(asm, "call _plz_mod") {
		t.Error("expected call _plz_mod")
	}
}

func TestGenLetComparison(t *testing.T) {
	asm := genTest(t, "DECLARE a WORD\nDECLARE b WORD\nLET x = a > b")
	if !strings.Contains(asm, "call _plz_gt") {
		t.Error("expected call _plz_gt")
	}
}

func TestGenLetNeg(t *testing.T) {
	asm := genTest(t, "DECLARE y WORD\nLET x = -y")
	if !strings.Contains(asm, "sbc hl, de") {
		t.Error("expected sbc hl, de for negation")
	}
}

func TestGenIfThen(t *testing.T) {
	asm := genTest(t, "DECLARE x WORD\nIF x THEN LET y = 1")
	if !strings.Contains(asm, "ld hl, (x)") {
		t.Error("expected condition load")
	}
	if !strings.Contains(asm, "jr z, _else_") {
		t.Error("expected conditional jump")
	}
	if !strings.Contains(asm, "_else_") {
		t.Error("expected _else_ label")
	}
	if strings.Contains(asm, "_end_") {
		t.Error("expected no _end_ label without ELSE")
	}
}

func TestGenIfThenElse(t *testing.T) {
	asm := genTest(t, "DECLARE x WORD\nIF x THEN LET y = 1 ELSE LET z = 2")
	if !strings.Contains(asm, "ld hl, (x)") {
		t.Error("expected condition load")
	}
	if !strings.Contains(asm, "jr z, _else_") {
		t.Error("expected conditional jump")
	}
	if !strings.Contains(asm, "_else_") {
		t.Error("expected _else_ label")
	}
	if !strings.Contains(asm, "_end_") {
		t.Error("expected _end_ label with ELSE")
	}
}

func TestGenGroupDo(t *testing.T) {
	asm := genTest(t, "DO LET x = 1 END")
	// Bare DO doesn't loop, just emits the body inline
	if !strings.Contains(asm, "ld hl, 1") {
		t.Error("expected body to be emitted")
	}
	if !strings.Contains(asm, "ld (x), hl") {
		t.Error("expected store")
	}
}

func TestGenGroupWhile(t *testing.T) {
	asm := genTest(t, "DECLARE a WORD\nWHILE a DO LET x = 1 END")
	if !strings.Contains(asm, "_while_") {
		t.Error("expected _while_ label")
	}
	if !strings.Contains(asm, "jr z, _end_") {
		t.Error("expected conditional exit")
	}
	if !strings.Contains(asm, "jr _while_") {
		t.Error("expected jump back")
	}
	if !strings.Contains(asm, "_end_") {
		t.Error("expected _end_ label")
	}
}

func TestGenGroupFor(t *testing.T) {
	asm := genTest(t, "DECLARE i WORD\nFOR i = 1 TO 10 DO LET x = i END")
	if !strings.Contains(asm, "_for_") {
		t.Error("expected _for_ label")
	}
	if !strings.Contains(asm, "add hl, de") {
		t.Error("expected add hl, de for step")
	}
	if !strings.Contains(asm, "ld (i), hl") {
		t.Error("expected store to loop variable")
	}
}

func TestGenGroupForBy(t *testing.T) {
	asm := genTest(t, "DECLARE i WORD\nFOR i = 0 TO 100 BY 2 DO LET x = i END")
	if !strings.Contains(asm, "ld hl, 2") {
		t.Error("expected step value 2")
	}
}

func TestGenLetArraySet(t *testing.T) {
	asm := genTest(t, "DECLARE i WORD\nLET arr[i] = 42")
	if !strings.Contains(asm, "ld hl, 42") {
		t.Error("expected rhs value")
	}
	if !strings.Contains(asm, "ld hl, arr") {
		t.Error("expected array base address")
	}
	if !strings.Contains(asm, "ld (hl), e") {
		t.Error("expected store through hl")
	}
}

func TestGenLetArrayRead(t *testing.T) {
	asm := genTest(t, "DECLARE arr WORD\nDECLARE i WORD\nLET x = arr[i]")
	if !strings.Contains(asm, "ld hl, arr") {
		t.Error("expected array base")
	}
	if !strings.Contains(asm, "add hl, hl") {
		t.Error("expected index * 2")
	}
	if !strings.Contains(asm, "ld a, (hl)") {
		t.Error("expected read from computed address")
	}
}

func TestGenLetSub(t *testing.T) {
	asm := genTest(t, "DECLARE a WORD\nDECLARE b WORD\nLET x = a - b")
	if !strings.Contains(asm, "sbc hl, de") {
		t.Error("expected sbc hl, de for subtract")
	}
}

func TestGenLetAnd(t *testing.T) {
	asm := genTest(t, "DECLARE a WORD\nDECLARE b WORD\nLET x = a & b")
	if !strings.Contains(asm, "and d") {
		t.Error("expected and d")
	}
}

func TestGenLetOr(t *testing.T) {
	asm := genTest(t, "DECLARE a WORD\nDECLARE b WORD\nLET x = a | b")
	if !strings.Contains(asm, "or e") {
		t.Error("expected or e")
	}
}

func TestGenLetXor(t *testing.T) {
	asm := genTest(t, "DECLARE a WORD\nDECLARE b WORD\nLET x = a ^ b")
	if !strings.Contains(asm, "xor d") {
		t.Error("expected xor d")
	}
}

func TestGenLetNot(t *testing.T) {
	asm := genTest(t, "DECLARE flag WORD\nLET x = !flag")
	if !strings.Contains(asm, "ld hl, 0") {
		t.Error("expected ld hl, 0 for false case")
	}
	if !strings.Contains(asm, "inc l") {
		t.Error("expected inc l for true case")
	}
}

func TestGenLetByteLoad(t *testing.T) {
	asm := genTest(t, "DECLARE b BYTE\nLET x = b")
	if !strings.Contains(asm, "ld a, (b)") {
		t.Error("expected byte load via a")
	}
	if !strings.Contains(asm, "ld h, 0") {
		t.Error("expected zero-extension")
	}
}

func TestGenLetByteStore(t *testing.T) {
	asm := genTest(t, "DECLARE b BYTE\nLET b = 42")
	if !strings.Contains(asm, "ld a, l") {
		t.Error("expected ld a, l for byte store")
	}
	if !strings.Contains(asm, "ld (b), a") {
		t.Error("expected ld (b), a")
	}
}

func TestGenLetByteArrayNoScale(t *testing.T) {
	asm := genTest(t, "DECLARE arr BYTE\nDECLARE i WORD\nLET x = arr[i]")
	if !strings.Contains(asm, "ld hl, arr") {
		t.Error("expected array base")
	}
	if strings.Contains(asm, "add hl, hl") {
		t.Error("byte array should NOT scale index by 2")
	}
	if !strings.Contains(asm, "ld h, 0") {
		t.Error("expected zero-extension")
	}
}

func TestGenDeclareStruct(t *testing.T) {
	asm := genTest(t, "DECLARE s STRUCT x WORD, y BYTE")
	if !strings.Contains(asm, "db 0, 0, 0, 0") {
		t.Error("expected 4 zero bytes for 2+1=3 rounded up to 4")
	}
	if !strings.Contains(asm, "s:") {
		t.Error("expected struct label")
	}
}

func TestGenStructFieldRead(t *testing.T) {
	asm := genTest(t, "DECLARE s STRUCT x WORD, y BYTE\nLET v = s.y")
	if !strings.Contains(asm, "ld hl, s") {
		t.Error("expected struct base")
	}
	if !strings.Contains(asm, "ld de, 2") {
		t.Error("expected field offset 2 for second field")
	}
	if !strings.Contains(asm, "add hl, de") {
		t.Error("expected offset add")
	}
	if !strings.Contains(asm, "ld h, 0") {
		t.Error("expected zero-extension for byte field")
	}
}

func TestGenStructFieldWrite(t *testing.T) {
	asm := genTest(t, "DECLARE s STRUCT x WORD, y BYTE\nLET s.x = 99")
	if !strings.Contains(asm, "ld hl, s") {
		t.Error("expected struct base")
	}
	// offset 0 for first field — no ld de,0 expected (optimized out)
	if !strings.Contains(asm, "ld (hl), e") {
		t.Error("expected store first byte")
	}
	if !strings.Contains(asm, "ld (hl), d") {
		t.Error("expected store second byte for word field")
	}
}
