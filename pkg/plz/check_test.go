package plz

import (
	"strings"
	"testing"
)

func checkTest(t *testing.T, src string) error {
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
	c := NewChecker()
	return prog.Check(c)
}

func checkOK(t *testing.T, src string) {
	t.Helper()
	if err := checkTest(t, src); err != nil {
		t.Fatalf("unexpected check error: %v", err)
	}
}

func checkErr(t *testing.T, src, want string) {
	t.Helper()
	err := checkTest(t, src)
	if err == nil {
		t.Fatal("expected check error")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}

func TestCheckUndeclaredVariable(t *testing.T) {
	checkErr(t, "LET x = y", "undeclared")
}

func TestCheckDeclaredVariable(t *testing.T) {
	checkOK(t, "DECLARE y WORD\nLET x = y")
}

func TestCheckArrayBoundsValid(t *testing.T) {
	checkOK(t, "DECLARE arr ARRAY [5] BYTE\nLET x = arr[3]")
}

func TestCheckArrayBoundsValidWrite(t *testing.T) {
	checkOK(t, "DECLARE arr ARRAY [5] BYTE\nLET arr[0] = 42")
}

func TestCheckArrayBoundsOutOfRange(t *testing.T) {
	checkErr(t, "DECLARE arr ARRAY [5] BYTE\nLET x = arr[7]", "out of bounds")
}

func TestCheckArrayBoundsWriteOutOfRange(t *testing.T) {
	checkErr(t, "DECLARE arr ARRAY [5] BYTE\nLET arr[10] = 99", "out of bounds")
}

func TestCheckArrayBoundsNegativeIndex(t *testing.T) {
	checkErr(t, "DECLARE arr ARRAY [5] BYTE\nLET x = arr[-1]", "out of bounds")
}

func TestCheckArrayBoundsVariableIndex(t *testing.T) {
	checkOK(t, "DECLARE arr ARRAY [5] BYTE\nDECLARE i WORD\nLET arr[i] = 42")
}

func TestCheckArrayBoundsConstantExpr(t *testing.T) {
	checkOK(t, "DECLARE arr ARRAY [5] BYTE\nLET x = arr[2+2]")
}

func TestCheckArrayBoundsConstantExprOutOfRange(t *testing.T) {
	checkErr(t, "DECLARE arr ARRAY [5] BYTE\nLET x = arr[2+4]", "out of bounds")
}

func TestCheckPragmaBoundcheck(t *testing.T) {
	checkOK(t, "PRAGMA BOUNDCHECK")
}

func TestCheckPragmaUnknown(t *testing.T) {
	checkErr(t, "PRAGMA FOO", "unrecognized pragma")
}

func TestCheckDefine(t *testing.T) {
	checkOK(t, "DEFINE mytype BYTE")
}

func TestCheckAt(t *testing.T) {
	checkOK(t, "AT 0xC000")
}

func TestCheckAtBank(t *testing.T) {
	checkOK(t, "AT BANK 1")
}

func TestCheckBank(t *testing.T) {
	checkOK(t, "BANK 3")
}

func TestCheckConstant(t *testing.T) {
	checkOK(t, "CONSTANT FOO = 42\nLET x = FOO")
}

func TestCheckConstantNoExpr(t *testing.T) {
	checkOK(t, "CONSTANT FOO")
}

func TestCheckOutput(t *testing.T) {
	checkOK(t, "DECLARE x BYTE\nOUTPUT 0 x")
}

func TestCheckData(t *testing.T) {
	checkOK(t, "DATA myarr 1, 2, 3")
}

func TestCheckSaveNoAtUndeclared(t *testing.T) {
	checkErr(t, "SAVE var", "undeclared")
}

func TestCheckSaveWithoutAt(t *testing.T) {
	checkOK(t, "DECLARE var BYTE\nSAVE var")
}

func TestCheckSaveWithAt(t *testing.T) {
	checkOK(t, "DECLARE var BYTE AT 0x8100\nSAVE var")
}

func TestCheckSaveNotRef(t *testing.T) {
	checkErr(t, "DECLARE arr ARRAY [5] BYTE AT 0x8100\nSAVE arr[0]", "must be a variable")
}

func TestCheckLoadNoAtUndeclared(t *testing.T) {
	checkErr(t, "LOAD var", "undeclared")
}

func TestCheckLoadWithoutAt(t *testing.T) {
	checkOK(t, "DECLARE var BYTE\nLOAD var")
}

func TestCheckLoadWithAt(t *testing.T) {
	checkOK(t, "DECLARE var BYTE AT 0xC000\nLOAD var")
}

func TestCheckSuspendUndeclared(t *testing.T) {
	checkErr(t, "SUSPEND foo", "undeclared task")
}

func TestCheckResumeUndeclared(t *testing.T) {
	checkErr(t, "RESUME foo", "undeclared task")
}

func TestCheckSleep(t *testing.T) {
	checkOK(t, "SLEEP 10")
}

func TestCheckReturn(t *testing.T) {
	checkOK(t, "PROCEDURE foo() BYTE\nRETURN 42\nEND")
	checkErr(t, "RETURN 42", "outside procedure")
}

func TestCheckInput(t *testing.T) {
	checkOK(t, "OUTPUT 0 INPUT(0)")
}

func TestCheckIf(t *testing.T) {
	checkOK(t, "DECLARE x BYTE\nIF x THEN LET y = 1")
}

func TestCheckIfElse(t *testing.T) {
	checkOK(t, "DECLARE x BYTE\nIF x THEN LET y = 1 ELSE LET z = 2")
}

func TestCheckGroupDo(t *testing.T) {
	checkOK(t, "DO LET x = 1 END")
}

func TestCheckGroupWhile(t *testing.T) {
	checkOK(t, "DECLARE x BYTE\nWHILE x DO LET x = x - 1 END")
}

func TestCheckGroupFor(t *testing.T) {
	checkOK(t, "DECLARE i WORD\nFOR i = 1 TO 10 DO LET x = i END")
}

func TestCheckLengthVariable(t *testing.T) {
	checkOK(t, "DECLARE x BYTE\nLET y = LENGTH(x)")
}

func TestCheckLengthArray(t *testing.T) {
	checkOK(t, "DECLARE arr ARRAY [10] BYTE\nLET n = LENGTH(arr)")
}

func TestCheckLengthUnboundedArray(t *testing.T) {
	checkErr(t, "DECLARE arr ARRAY BYTE\nLET n = LENGTH(arr)", "cannot determine length")
}

func TestCheckLengthData(t *testing.T) {
	checkOK(t, "DATA myarr 1, 2, 3\nLET n = LENGTH(myarr)")
}

func TestCheckLengthUndeclared(t *testing.T) {
	checkErr(t, "LET n = LENGTH(x)", "undeclared")
}

func TestCheckInterruptUndeclared(t *testing.T) {
	checkErr(t, "INTERRUPT foo", "undefined procedure")
}

func TestCheckNMIUndeclared(t *testing.T) {
	checkErr(t, "NMI foo", "undefined procedure")
}

func TestCheckEvalConstExprPrefixNeg(t *testing.T) {
	checkOK(t, "CONSTANT V = -5\nLET x = V")
}

func TestCheckEvalConstExprPrefixNot(t *testing.T) {
	checkOK(t, "CONSTANT V = !0\nLET x = V")
}

func TestCheckEvalConstExprInfixSub(t *testing.T) {
	checkOK(t, "CONSTANT V = 5 - 3\nLET x = V")
}

func TestCheckEvalConstExprInfixMul(t *testing.T) {
	checkOK(t, "CONSTANT V = 2 * 3\nLET x = V")
}

func TestCheckEvalConstExprInfixDiv(t *testing.T) {
	checkOK(t, "CONSTANT V = 10 / 3\nLET x = V")
}

func TestCheckEvalConstExprInfixBinop(t *testing.T) {
	checkOK(t, "CONSTANT V = 1 + 2 * 3 - 4\nLET x = V")
}

func TestCheckEvalConstExprComparison(t *testing.T) {
	checkOK(t, "CONSTANT V = 5 > 3\nLET x = V")
}

func TestCheckEvalConstExprEquality(t *testing.T) {
	checkOK(t, "CONSTANT V = 5 == 5\nLET x = V")
}

func TestCheckEvalConstExprNotEqual(t *testing.T) {
	checkOK(t, "CONSTANT V = 5 != 3\nLET x = V")
}

func TestCheckEvalConstExprLessThan(t *testing.T) {
	checkOK(t, "CONSTANT V = 3 < 5\nLET x = V")
}

func TestCheckEvalConstExprLTE(t *testing.T) {
	checkOK(t, "CONSTANT V = 3 <= 5\nLET x = V")
}

func TestCheckEvalConstExprGTE(t *testing.T) {
	checkOK(t, "CONSTANT V = 5 >= 3\nLET x = V")
}

func TestCheckEvalConstExprShiftLeft(t *testing.T) {
	checkOK(t, "CONSTANT V = 1 << 3\nLET x = V")
}

func TestCheckEvalConstExprShiftRight(t *testing.T) {
	checkOK(t, "CONSTANT V = 16 >> 2\nLET x = V")
}

func TestCheckEvalConstExprAnd(t *testing.T) {
	checkOK(t, "CONSTANT V = 3 & 1\nLET x = V")
}

func TestCheckEvalConstExprOr(t *testing.T) {
	checkOK(t, "CONSTANT V = 2 | 1\nLET x = V")
}

func TestCheckEvalConstExprXor(t *testing.T) {
	checkOK(t, "CONSTANT V = 3 ^ 1\nLET x = V")
}

func TestCheckEvalConstExprParenthesized(t *testing.T) {
	checkOK(t, "CONSTANT V = (1 + 2) * 3\nLET x = V")
}

func TestCheckEvalConstExprDivisionByZero(t *testing.T) {
	checkErr(t, "CONSTANT V = 1 / 0", "division by zero")
}

func TestCheckEvalConstExprModuloByZero(t *testing.T) {
	checkErr(t, "CONSTANT V = 1 % 0", "modulo by zero")
}

func TestCheckReferenceField(t *testing.T) {
	checkOK(t, "DECLARE p RECORD x BYTE, y BYTE END\nLET z = p.x")
}

func TestCheckReferenceFieldNotFound(t *testing.T) {
	checkErr(t, "DECLARE p RECORD x BYTE, y BYTE END\nLET p.z = 5", "has no field")
}

func TestCheckArrayBoundsData(t *testing.T) {
	checkOK(t, "DATA myarr 1, 2, 3\nLET x = myarr[1]")
}

func TestCheckArrayBoundsDataOutOfRange(t *testing.T) {
	checkErr(t, "DATA myarr 1, 2, 3\nLET x = myarr[10]", "out of bounds")
}

func TestCheckTaskValid(t *testing.T) {
	checkOK(t, "TASK t\nYIELD\nEND")
}

func TestCheckTaskDuplicate(t *testing.T) {
	checkErr(t, "TASK t\nYIELD\nEND\nTASK t\nYIELD\nEND", "duplicate task")
}

func TestCheckEvalConstExprLength(t *testing.T) {
	checkOK(t, "DECLARE arr ARRAY [10] BYTE\nCONSTANT n = LENGTH(arr)\nLET x = n")
}

func TestCheckEvalConstExprLengthVar(t *testing.T) {
	checkOK(t, "DECLARE x BYTE\nCONSTANT n = LENGTH(x)")
}

func TestCheckEvalConstExprLengthData(t *testing.T) {
	checkOK(t, "DATA myarr 1, 2, 3\nCONSTANT n = LENGTH(myarr)")
}

func TestCheckEvalConstExprSuffixError(t *testing.T) {
	checkErr(t, "CONSTANT V = arr[0]", "cannot be used in constant expressions")
}

func TestCheckEvalConstExprSuffixIndex(t *testing.T) {
	checkErr(t, "CONSTANT V = FOO[0]", "cannot be used in constant expressions")
}

func TestCheckEvalConstExprSuffixField(t *testing.T) {
	checkErr(t, "CONSTANT V = FOO.x", "cannot be used in constant expressions")
}

func TestCheckEvalConstExprUndeclaredRef(t *testing.T) {
	checkErr(t, "CONSTANT V = FOO", "undefined identifier")
}
