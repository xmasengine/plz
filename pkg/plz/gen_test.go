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
	if !strings.Contains(asm, "sbc hl, de") {
		t.Error("expected sbc hl, de for comparison")
	}
	if !strings.Contains(asm, "ld hl, 1") {
		t.Error("expected ld hl, 1 for true result")
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
	if !strings.Contains(asm, "jmp z, _else_") {
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
	if !strings.Contains(asm, "jmp z, _else_") {
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
	if !strings.Contains(asm, "jmp z, _end_") {
		t.Error("expected conditional exit")
	}
	if !strings.Contains(asm, "jmp _while_") {
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

func TestGenDeclareRecord(t *testing.T) {
	asm := genTest(t, "DECLARE s RECORD x WORD, y BYTE END")
	if !strings.Contains(asm, "db 0, 0, 0, 0") {
		t.Error("expected 4 zero bytes for 2+1=3 rounded up to 4")
	}
	if !strings.Contains(asm, "s:") {
		t.Error("expected struct label")
	}
}

func TestGenRecordFieldRead(t *testing.T) {
	asm := genTest(t, "DECLARE s RECORD x WORD, y BYTE END\nLET v = s.y")
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

func TestGenArrayDeclareFixed(t *testing.T) {
	asm := genTest(t, "DECLARE arr ARRAY [10] BYTE")
	if !strings.Contains(asm, "db 0, 0, 0, 0, 0, 0, 0, 0, 0, 0") {
		t.Error("expected 10 zero bytes")
	}
}

func TestGenArrayDeclareWord(t *testing.T) {
	asm := genTest(t, "DECLARE arr ARRAY [5] WORD")
	// 5 * 2 = 10 bytes
	if !strings.Contains(asm, "db 0, 0, 0, 0, 0, 0, 0, 0, 0, 0") {
		t.Error("expected 10 zero bytes for 5 words")
	}
}

func TestGenArrayDeclareMultiDim(t *testing.T) {
	asm := genTest(t, "DECLARE arr ARRAY [12] BYTE")
	// 12 bytes
	if !strings.Contains(asm, "db 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0") {
		t.Error("expected 12 zero bytes for 12-element byte array")
	}
}

func TestGenProcNoArgs(t *testing.T) {
	asm := genTest(t, "PROCEDURE foo\nRETURN 99\nEND")
	if !strings.Contains(asm, "_plz_foo:") {
		t.Error("expected _plz_foo: label")
	}
	if !strings.Contains(asm, "ld hl, 99") {
		t.Error("expected return value")
	}
	if !strings.Contains(asm, "ret") {
		t.Error("expected ret")
	}
}

func TestGenProcOneArg(t *testing.T) {
	asm := genTest(t, "PROCEDURE double (x WORD) WORD\nRETURN x + x\nEND\nCALL double(5)")
	if !strings.Contains(asm, "ld hl, 5") {
		t.Error("expected call with arg 5")
	}
	if !strings.Contains(asm, "call _plz_double") {
		t.Error("expected call _plz_double")
	}
}

func TestGenProcTwoArgs(t *testing.T) {
	asm := genTest(t, "PROCEDURE add (a WORD, b WORD) WORD\nRETURN a + b\nEND\nCALL add(3, 4)")
	if !strings.Contains(asm, "ld hl, 3") {
		t.Error("expected arg1 = 3")
	}
	if !strings.Contains(asm, "pop de") {
		t.Error("expected arg2 in DE")
	}
	if !strings.Contains(asm, "call _plz_add") {
		t.Error("expected call _plz_add")
	}
}

func TestGenProcLocalDeclare(t *testing.T) {
	asm := genTest(t, `PROCEDURE foo (x WORD) WORD
DECLARE t WORD
LET t = x + 1
RETURN t
END
CALL foo(5)`)
	if !strings.Contains(asm, "_plz_foo_x: db 0, 0") {
		t.Error("expected RAM allocation for param x")
	}
	if !strings.Contains(asm, "_plz_foo_t: db 0, 0") {
		t.Error("expected RAM allocation for local t")
	}
}

func TestGenProcByteParam(t *testing.T) {
	asm := genTest(t, "PROCEDURE double (x BYTE) WORD\nRETURN x + x\nEND\nCALL double(21)")
	if !strings.Contains(asm, "ld a, l") {
		t.Error("expected byte truncation for BYTE param save")
	}
	if !strings.Contains(asm, "_plz_double_x: db 0") {
		t.Error("expected RAM allocation for BYTE param x")
	}
}

func TestGenProcByteReturn(t *testing.T) {
	asm := genTest(t, "PROCEDURE getByte BYTE\nRETURN 42\nEND\nCALL getByte()")
	if !strings.Contains(asm, "ld h, 0") {
		t.Error("expected BYTE return zero-extension")
	}
}

func TestGenProcRecordParam(t *testing.T) {
	asm := genTest(t, "DECLARE s RECORD x BYTE, y WORD END\nPROCEDURE useRec (rv RECORD x BYTE, y WORD END) WORD\nRETURN rv.y\nEND\nCALL useRec(s)")
	if !strings.Contains(asm, "ld hl, s") {
		t.Error("expected address load for record arg at call site")
	}
	if !strings.Contains(asm, "ld hl, (_plz_useRec_rv)") {
		t.Error("expected dereference for record param field access")
	}
}

func TestGenProcRecordReturn(t *testing.T) {
	asm := genTest(t, "DECLARE data RECORD x BYTE, y WORD END\nPROCEDURE getPtr RECORD x BYTE, y WORD END\nRETURN data\nEND\nCALL getPtr()")
	if !strings.Contains(asm, "ld hl, data") {
		t.Error("expected address load for RECORD return")
	}
}

func TestGenStructFieldWrite(t *testing.T) {
	asm := genTest(t, "DECLARE s RECORD x WORD, y BYTE END\nLET s.x = 99")
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

func TestGenDefineTypeAlias(t *testing.T) {
	asm := genTest(t, "DEFINE Point RECORD x WORD, y BYTE END\nDECLARE p TYPE Point\nLET p.x = 42")
	if !strings.Contains(asm, "p:") {
		t.Error("expected label for declared record via type alias")
	}
	if !strings.Contains(asm, "ld hl, p") {
		t.Error("expected record base access")
	}
}

func TestGenDefineByteAlias(t *testing.T) {
	asm := genTest(t, "DEFINE MyByte BYTE\nDECLARE bv TYPE MyByte\nLET bv = 99")
	if !strings.Contains(asm, "bv:") {
		t.Error("expected label for byte variable via type alias")
	}
	if !strings.Contains(asm, "ld a, l") {
		t.Error("expected byte truncation for store")
	}
}

func TestGenDefineWordAlias(t *testing.T) {
	asm := genTest(t, "DEFINE MyWord WORD\nDECLARE w TYPE MyWord\nLET w = 42")
	if !strings.Contains(asm, "w:") {
		t.Error("expected label for word variable via type alias")
	}
	if !strings.Contains(asm, "ld (w), hl") {
		t.Error("expected word store")
	}
}

func TestGenArrayOfRecordsFieldRead(t *testing.T) {
	asm := genTest(t, "DECLARE arr ARRAY [10] RECORD x WORD, y BYTE END\nDECLARE i WORD\nLET v = arr[i].y")
	if !strings.Contains(asm, "ld hl, arr") {
		t.Error("expected array base")
	}
	if !strings.Contains(asm, "add hl, hl") {
		t.Error("expected index * 4 (record size)")
	}
	if !strings.Contains(asm, "ld de, 2") {
		t.Error("expected field offset 2")
	}
	if !strings.Contains(asm, "ld h, 0") {
		t.Error("expected zero-extension for byte field")
	}
}

func TestGenArrayOfRecordsFieldWrite(t *testing.T) {
	asm := genTest(t, "DECLARE arr ARRAY [10] RECORD x WORD, y BYTE END\nDECLARE i WORD\nLET arr[i].x = 42")
	if !strings.Contains(asm, "ld hl, arr") {
		t.Error("expected array base")
	}
	if !strings.Contains(asm, "add hl, hl") {
		t.Error("expected index * 4 for record")
	}
	if !strings.Contains(asm, "ld (hl), e") {
		t.Error("expected store first byte")
	}
	if !strings.Contains(asm, "ld (hl), d") {
		t.Error("expected store second byte for word field")
	}
}

func TestGenRecordWithArrayFieldRead(t *testing.T) {
	asm := genTest(t, "DECLARE s RECORD x WORD, arr ARRAY [5] BYTE END\nDECLARE i WORD\nLET v = s.arr[i]")
	if !strings.Contains(asm, "ld hl, s") {
		t.Error("expected record base")
	}
	if !strings.Contains(asm, "ld de, 2") {
		t.Error("expected field offset for arr")
	}
	if !strings.Contains(asm, "ld h, 0") {
		t.Error("expected zero-extension for byte")
	}
}

func TestGenRecordWithArrayFieldWrite(t *testing.T) {
	asm := genTest(t, "DECLARE s RECORD x WORD, arr ARRAY [5] BYTE END\nDECLARE i WORD\nLET s.arr[i] = 99")
	if !strings.Contains(asm, "ld hl, s") {
		t.Error("expected record base")
	}
	if !strings.Contains(asm, "ld de, 2") {
		t.Error("expected field offset for arr")
	}
	if !strings.Contains(asm, "ld (hl), e") {
		t.Error("expected byte store")
	}
}

func TestGenBank(t *testing.T) {
	asm := genTest(t, "BANK 3\n")
	if !strings.Contains(asm, "ld a, 3") {
		t.Errorf("expected ld a, 3 in output:\n%s", asm)
	}
	if !strings.Contains(asm, "ld (0xFFFD), a") {
		t.Errorf("expected ld (0xFFFD), a in output:\n%s", asm)
	}
}

func TestGenAtBank(t *testing.T) {
	asm := genTest(t, "AT BANK 2\nfont: DATA 1, 2, 3")
	if !strings.Contains(asm, "bank 2") {
		t.Errorf("expected bank 2 directive in output:\n%s", asm)
	}
	if !strings.Contains(asm, "font:") {
		t.Errorf("expected font: label in output:\n%s", asm)
	}
}

func TestGenAt(t *testing.T) {
	asm := genTest(t, "AT 0xC000\nDECLARE x BYTE")
	if !strings.Contains(asm, "org 0xc000") {
		t.Errorf("expected org directive for c000 in output:\n%s", asm)
	}
	if !strings.Contains(asm, "x: db 0") {
		t.Errorf("expected x: db 0 in output:\n%s", asm)
	}
}
