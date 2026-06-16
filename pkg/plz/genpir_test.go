package plz

import (
	"strings"
	"testing"

	"github.com/xmasengine/plz/pkg/pir"
)

// genPIR is a test helper that scans, parses, checks, and generates PIR.
func genPIR(t *testing.T, src string) *pir.Program {
	t.Helper()
	tokens, err := Scan(strings.NewReader(src))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	prog := &Program{}
	parser := NewParser(tokens)
	if err := prog.Parse(parser); err != nil {
		t.Fatalf("parse: %v", err)
	}
	pirProg, err := prog.GenPIR()
	if err != nil {
		t.Fatalf("GenPIR: %v", err)
	}
	return pirProg
}

func TestGenPIR_SimpleLet(t *testing.T) {
	p := genPIR(t, "LET x = 42")
	want := "NOP\nPUSH_B 42\nPUT_W x\nHLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_LetVariableRef(t *testing.T) {
	p := genPIR(t, "DECLARE y WORD\nLET x = y")
	want := "NOP\nGET_W y\nPUT_W x\nHLT\nALLOC 2\nVAR y\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_LetBinaryAdd(t *testing.T) {
	p := genPIR(t, "LET x = 10 + 20")
	want := "NOP\nPUSH_B 10\nPUSH_B 20\nADD_W\nPUT_W x\nHLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_LetBinarySub(t *testing.T) {
	p := genPIR(t, "LET x = 100 - 30")
	want := "NOP\nPUSH_B 100\nPUSH_B 30\nSUB_W\nPUT_W x\nHLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_LetBinaryMul(t *testing.T) {
	p := genPIR(t, "LET x = 6 * 7")
	want := "NOP\nPUSH_B 6\nPUSH_B 7\nMUL_W\nPUT_W x\nHLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_LetBinaryDiv(t *testing.T) {
	p := genPIR(t, "LET x = 100 / 3")
	want := "NOP\nPUSH_B 100\nPUSH_B 3\nDIV_W\nPUT_W x\nHLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_LetBinaryMod(t *testing.T) {
	p := genPIR(t, "LET x = 100 % 3")
	want := "NOP\nPUSH_B 100\nPUSH_B 3\nMOD_W\nPUT_W x\nHLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_LetComparisonEQ(t *testing.T) {
	p := genPIR(t, "LET x = 10 == 20")
	want := "NOP\nPUSH_B 10\nPUSH_B 20\nIS_W EQ\nPUT_W x\nHLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_LetComparisonGT(t *testing.T) {
	p := genPIR(t, "LET x = 30 > 20")
	want := "NOP\nPUSH_B 30\nPUSH_B 20\nIS_W GT\nPUT_W x\nHLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_LetComparisonLT(t *testing.T) {
	p := genPIR(t, "LET x = 10 < 20")
	want := "NOP\nPUSH_B 10\nPUSH_B 20\nIS_W LT\nPUT_W x\nHLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_IfThen(t *testing.T) {
	p := genPIR(t, "DECLARE x WORD\nDECLARE y WORD\nIF x THEN LET y = 1")
	want := "" +
		"NOP\n" +
		"GET_W x\n" +
		"NOT_W\n" +
		"GO_IF _else_1\n" +
		"PUSH_B 1\n" +
		"PUT_W y\n" +
		"TAG _else_1\n" +
		"HLT\n" +
		"ALLOC 2\nVAR x\n" +
		"ALLOC 2\nVAR y\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_IfThenElse(t *testing.T) {
	p := genPIR(t, "DECLARE x WORD\nDECLARE y WORD\nDECLARE z WORD\nIF x THEN LET y = 1 ELSE LET z = 2")
	want := "" +
		"NOP\n" +
		"GET_W x\n" +
		"NOT_W\n" +
		"GO_IF _else_1\n" +
		"PUSH_B 1\n" +
		"PUT_W y\n" +
		"GO _end_1\n" +
		"TAG _else_1\n" +
		"PUSH_B 2\n" +
		"PUT_W z\n" +
		"TAG _end_1\n" +
		"HLT\n" +
		"ALLOC 2\nVAR x\n" +
		"ALLOC 2\nVAR y\n" +
		"ALLOC 2\nVAR z\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_WhileLoop(t *testing.T) {
	p := genPIR(t, "DECLARE x WORD\nWHILE x DO LET x = x - 1 END")
	want := "" +
		"NOP\n" +
		"TAG _while_1\n" +
		"GET_W x\n" +
		"NOT_W\n" +
		"GO_IF _end_1\n" +
		"GET_W x\n" +
		"PUSH_B 1\n" +
		"SUB_W\n" +
		"PUT_W x\n" +
		"GO _while_1\n" +
		"TAG _end_1\n" +
		"HLT\n" +
		"ALLOC 2\nVAR x\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_ForLoop(t *testing.T) {
	p := genPIR(t, "DECLARE i WORD\nFOR i = 1 TO 10 DO LET x = i END")
	want := "" +
		"NOP\n" +
		"PUSH_B 1\n" +
		"PUT_W i\n" +
		"PUSH_B 10\n" +
		"ALLOC 2\nVAR _plz_for_to_1\n" +
		"PUT_W _plz_for_to_1\n" +
		"PUSH_W 1\n" +
		"ALLOC 2\nVAR _plz_for_step_1\n" +
		"PUT_W _plz_for_step_1\n" +
		"TAG _for_1\n" +
		"GET_W i\n" +
		"GET_W _plz_for_to_1\n" +
		"IS_W GT\n" +
		"GO_IF _end_1\n" +
		"GET_W i\n" +
		"PUT_W x\n" +
		"TAG _for_cont_1\n" +
		"GET_W i\n" +
		"GET_W _plz_for_step_1\n" +
		"ADD_W\n" +
		"PUT_W i\n" +
		"GO _for_1\n" +
		"TAG _end_1\n" +
		"HLT\n" +
		"ALLOC 2\nVAR i\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_ProcedureNoArgs(t *testing.T) {
	p := genPIR(t, "PROCEDURE foo() BYTE\nRETURN 42\nEND")
	want := "" +
		"NOP\n" +
		"HLT\n" +
		"ROUTE foo\n" +
		"PUSH_B 42\n" +
		"DONE\n" +
		"DONE\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_ProcedureArgs(t *testing.T) {
	p := genPIR(t, "PROCEDURE foo(a BYTE, b WORD) BYTE\nRETURN a\nEND")
	want := "" +
		"NOP\n" +
		"HLT\n" +
		"ROUTE foo\n" +
		"VAR a\n" +
		"ALLOC 2\nVAR b\n" +
		"PUT_B a\n" +
		"PUT_W b\n" +
		"GET_B a\n" +
		"DONE\n" +
		"DONE\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_CallStmt(t *testing.T) {
	p := genPIR(t, "PROCEDURE foo() END\nCALL foo()")
	want := "" +
		"NOP\n" +
		"RUN foo\n" +
		"HLT\n" +
		"ROUTE foo\n" +
		"DONE\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_ReturnValue(t *testing.T) {
	p := genPIR(t, "PROCEDURE foo() BYTE\nRETURN 99\nEND")
	want := "" +
		"NOP\n" +
		"HLT\n" +
		"ROUTE foo\n" +
		"PUSH_B 99\n" +
		"DONE\n" +
		"DONE\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_Output(t *testing.T) {
	p := genPIR(t, "OUTPUT 0 42")
	want := "NOP\nPUSH_B 42\nOUT_B 0\nHLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_Halt(t *testing.T) {
	p := genPIR(t, "HALT")
	want := "NOP\nHLT\nHLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_EnableDisable(t *testing.T) {
	p := genPIR(t, "DISABLE\nENABLE")
	want := "NOP\nDII\nENI\nHLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_Constants(t *testing.T) {
	p := genPIR(t, "CONSTANT FOO = 42\nLET x = FOO")
	want := "NOP\nPUSH_W 42\nPUT_W x\nHLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_Data(t *testing.T) {
	p := genPIR(t, "DATA mydata 10, 20, 30")
	want := "" +
		"NOP\n" +
		"HLT\n" +
		"TAG mydata\n" +
		"DATA_B 10\n" +
		"DATA_B 20\n" +
		"DATA_B 30\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_NestedScope(t *testing.T) {
	p := genPIR(t, "PROCEDURE foo() BYTE\nDECLARE x BYTE\nLET x = 42\nRETURN x\nEND")
	want := "" +
		"NOP\n" +
		"HLT\n" +
		"ROUTE foo\n" +
		"VAR x\n" +
		"PUSH_B 42\n" +
		"PUT_B x\n" +
		"GET_B x\n" +
		"DONE\n" +
		"DONE\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_ByteVarLet(t *testing.T) {
	p := genPIR(t, "DECLARE b BYTE\nLET b = 100")
	want := "NOP\nPUSH_B 100\nPUT_B b\nHLT\nVAR b\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_IfExprCondition(t *testing.T) {
	p := genPIR(t, "DECLARE x WORD\nIF x == 0 THEN LET y = 1")
	// IF with comparison: IS_W EQ pushes 1 if x==0, NOT_W inverts, GO_IF jumps if non-zero (= original was false)
	want := "" +
		"NOP\n" +
		"GET_W x\n" +
		"PUSH_B 0\n" +
		"IS_W EQ\n" +
		"NOT_W\n" +
		"GO_IF _else_1\n" +
		"PUSH_B 1\n" +
		"PUT_W y\n" +
		"TAG _else_1\n" +
		"HLT\n" +
		"ALLOC 2\nVAR x\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_ForLoopWithStep(t *testing.T) {
	p := genPIR(t, "DECLARE i WORD\nFOR i = 0 TO 10 BY 2 DO LET x = i END")
	want := "" +
		"NOP\n" +
		"PUSH_B 0\n" +
		"PUT_W i\n" +
		"PUSH_B 10\n" +
		"ALLOC 2\nVAR _plz_for_to_1\n" +
		"PUT_W _plz_for_to_1\n" +
		"PUSH_B 2\n" +
		"ALLOC 2\nVAR _plz_for_step_1\n" +
		"PUT_W _plz_for_step_1\n" +
		"TAG _for_1\n" +
		"GET_W i\n" +
		"GET_W _plz_for_to_1\n" +
		"IS_W GT\n" +
		"GO_IF _end_1\n" +
		"GET_W i\n" +
		"PUT_W x\n" +
		"TAG _for_cont_1\n" +
		"GET_W i\n" +
		"GET_W _plz_for_step_1\n" +
		"ADD_W\n" +
		"PUT_W i\n" +
		"GO _for_1\n" +
		"TAG _end_1\n" +
		"HLT\n" +
		"ALLOC 2\nVAR i\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_ProcedureCallWithArgs(t *testing.T) {
	p := genPIR(t, "PROCEDURE foo(a BYTE, b WORD) BYTE\nRETURN a\nEND\nCALL foo(10, 20)")
	want := "" +
		"NOP\n" +
		"PUSH_B 20\n" +
		"PUSH_B 10\n" +
		"RUN foo\n" +
		"HLT\n" +
		"ROUTE foo\n" +
		"VAR a\n" +
		"ALLOC 2\nVAR b\n" +
		"PUT_B a\n" +
		"PUT_W b\n" +
		"GET_B a\n" +
		"DONE\n" +
		"DONE\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_ReentrantProcedure(t *testing.T) {
	p := genPIR(t, "PROCEDURE foo(a BYTE) BYTE REENTRANT\nRETURN a\nEND")
	want := "" +
		"NOP\n" +
		"HLT\n" +
		"ROUTE foo\n" +
		"FRAME 1\n" +
		"LOCAL_B a\n" +
		"PUT_B a\n" +
		"GET_B a\n" +
		"DONE\n" +
		"DONE\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_MultipleProcedures(t *testing.T) {
	p := genPIR(t, ""+
		"PROCEDURE inc(x WORD) WORD\nRETURN x + 1\nEND\n"+
		"PROCEDURE dec(x WORD) WORD\nRETURN x - 1\nEND\n"+
		"LET a = 5\n")
	want := "" +
		"NOP\n" +
		"PUSH_B 5\n" +
		"PUT_W a\n" +
		"HLT\n" +
		"ROUTE inc\n" +
		"ALLOC 2\nVAR x\n" +
		"PUT_W x\n" +
		"GET_W x\n" +
		"PUSH_B 1\n" +
		"ADD_W\n" +
		"DONE\n" +
		"DONE\n" +
		"ROUTE dec\n" +
		"ALLOC 2\nVAR x\n" +
		"PUT_W x\n" +
		"GET_W x\n" +
		"PUSH_B 1\n" +
		"SUB_W\n" +
		"DONE\n" +
		"DONE\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_LabeledStatement(t *testing.T) {
	p := genPIR(t, "start:\nHALT\nloop:\nHALT")
	want := "" +
		"NOP\n" +
		"TAG start\n" +
		"HLT\n" +
		"TAG loop\n" +
		"HLT\n" +
		"HLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_DoBlock(t *testing.T) {
	p := genPIR(t, "DO LET x = 1 LET y = 2 END")
	want := "" +
		"NOP\n" +
		"PUSH_B 1\n" +
		"PUT_W x\n" +
		"PUSH_B 2\n" +
		"PUT_W y\n" +
		"HLT\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_MultipleStatements(t *testing.T) {
	p := genPIR(t, ""+
		"DECLARE x WORD\n"+
		"DECLARE y WORD\n"+
		"LET x = 10\n"+
		"LET y = 20\n"+
		"LET x = x + y\n")
	want := "" +
		"NOP\n" +
		"PUSH_B 10\n" +
		"PUT_W x\n" +
		"PUSH_B 20\n" +
		"PUT_W y\n" +
		"GET_W x\n" +
		"GET_W y\n" +
		"ADD_W\n" +
		"PUT_W x\n" +
		"HLT\n" +
		"ALLOC 2\nVAR x\n" +
		"ALLOC 2\nVAR y\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_DataString(t *testing.T) {
	p := genPIR(t, `DATA msg TEXT "hello"`)
	want := "" +
		"NOP\n" +
		"HLT\n" +
		"TAG msg\n" +
		`DATA_STR "hello"` + "\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_ProcedureWithLocalDeclares(t *testing.T) {
	p := genPIR(t, "PROCEDURE foo() WORD\nDECLARE a WORD\nDECLARE b BYTE\nLET a = 100\nLET b = 200\nRETURN a\nEND")
	want := "" +
		"NOP\n" +
		"HLT\n" +
		"ROUTE foo\n" +
		"ALLOC 2\nVAR a\n" +
		"VAR b\n" +
		"PUSH_B 100\n" +
		"PUT_W a\n" +
		"PUSH_B 200\n" +
		"PUT_B b\n" +
		"GET_W a\n" +
		"DONE\n" +
		"DONE\n"
	if got := p.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestGenPIR_StructureCheck(t *testing.T) {
	p := genPIR(t, "LET x = 42")

	if len(p.Instrs) < 4 {
		t.Fatalf("expected >= 4 instructions, got %d", len(p.Instrs))
	}

	// NOP
	if p.Instrs[0].Op != pir.NOP {
		t.Errorf("instr[0] = %v, want NOP", p.Instrs[0].Op)
	}

	// PUSH_B 42
	if p.Instrs[1].Op != pir.PUSH_B {
		t.Errorf("instr[1] = %v, want PUSH_B", p.Instrs[1].Op)
	}
	if p.Instrs[1].Operand.Type != pir.OpNumber || p.Instrs[1].Operand.Num != 42 {
		t.Errorf("instr[1] operand = %+v, want Num=42", p.Instrs[1].Operand)
	}

	// PUT_W x
	if p.Instrs[2].Op != pir.PUT_W {
		t.Errorf("instr[2] = %v, want PUT_W", p.Instrs[2].Op)
	}
	if p.Instrs[2].Operand.Type != pir.OpName || p.Instrs[2].Operand.Name != "x" {
		t.Errorf("instr[2] operand = %+v, want Name=x", p.Instrs[2].Operand)
	}

	// HLT
	if p.Instrs[3].Op != pir.HLT {
		t.Errorf("instr[3] = %v, want HLT", p.Instrs[3].Op)
	}
}
