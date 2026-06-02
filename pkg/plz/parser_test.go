package plz

import (
	"os"
	"strings"
	"testing"
)

func parseLetExpr(src string) (*Expression, error) {
	tokens, err := Scan(strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	// We need to consume the expression from the token stream.
	// Wrap it in a LET statement so the scanner produces the right tokens,
	// then manually skip past "LET x =" to the expression.
	p := NewParser(tokens)
	var s Statement
	err = s.Parse(p)
	if err != nil {
		return nil, err
	}
	if let, ok := s.Command.(Let); !ok {
		panic("expected Let statement")
	} else {
		return &let.Expression, nil
	}
}

func TestParseSimpleNumber(t *testing.T) {
	expr, err := parseLetExpr("LET x = 42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Operand() == nil || expr.Operand().Literal() == nil || expr.Operand().Literal().Number() == nil {
		t.Fatal("expected literal number")
	}
	if expr.Operand().Literal().Number().Value != 42 {
		t.Errorf("expected 42, got %d", expr.Operand().Literal().Number().Value)
	}
}

func TestParseIdentifier(t *testing.T) {
	expr, err := parseLetExpr("LET x = foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Operand() == nil || expr.Ref() == nil {
		t.Fatal("expected reference")
	}
	if expr.Ref().Identifier != Identifier("foo") {
		t.Errorf("expected foo, got %s", expr.Ref().Identifier)
	}
}

func TestParseBinaryExpr(t *testing.T) {
	expr, err := parseLetExpr("LET x = 1 + 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Infix() == nil {
		t.Fatal("expected Infix expression")
	}
	if expr.Infix().Operator != OperatorADD {
		t.Errorf("expected ADD, got %v", expr.Infix().Operator)
	}
	// left: 1
	left := expr.Infix().Operands[0].Expr()
	if left == nil || left.Operand() == nil {
		t.Fatal("expected left operand")
	}
	// right: 2
	right := expr.Infix().Operands[1].Expr()
	if right == nil || right.Operand() == nil {
		t.Fatal("expected right operand")
	}
}

func TestParseNested(t *testing.T) {
	expr, err := parseLetExpr("LET x = 1 + 2 * 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Top is ADD (since + has lower priority than *)
	if expr.Infix() == nil {
		t.Fatal("expected Infix")
	}
	if expr.Infix().Operator != OperatorADD {
		t.Errorf("expected ADD at top, got %v", expr.Infix().Operator)
	}
	// Right side should be MUL (2 * 3)
	right := expr.Infix().Operands[1].Expr()
	if right == nil || right.Infix() == nil {
		t.Fatal("expected Infix on right")
	}
	if right.Infix().Operator != OperatorMUL {
		t.Errorf("expected MUL on right, got %v", right.Infix().Operator)
	}
}

func TestParseParens(t *testing.T) {
	expr, err := parseLetExpr("LET x = (1 + 2) * 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Top is MUL
	if expr.Infix() == nil || expr.Infix().Operator != OperatorMUL {
		t.Fatalf("expected MUL at top, got %c", expr.Infix().Operator)
	}
	// Left should be ADD (from parentheses)
	left := expr.Infix().Operands[0].Expr()
	if left == nil || left.Infix() == nil || left.Infix().Operator != OperatorADD {
		t.Errorf("expected ADD in parens, got %c", left.Infix().Operator)
	}
}

func TestParseUnaryNeg(t *testing.T) {
	expr, err := parseLetExpr("LET x = -5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Prefix() == nil {
		t.Fatal("expected Prefix expression")
	}
	if expr.Prefix().Operator != OperatorNEG {
		t.Errorf("expected NEG, got %v", expr.Prefix().Operator)
	}
}

func TestParseUnaryNot(t *testing.T) {
	expr, err := parseLetExpr("LET x = !flag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Prefix() == nil || expr.Prefix().Operator != OperatorNOT {
		t.Errorf("expected NOT, got %v", expr.Prefix().Operator)
	}
}

func TestParseComparison(t *testing.T) {
	expr, err := parseLetExpr("LET x = a > b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Infix() == nil || expr.Infix().Operator != OperatorGT {
		t.Errorf("expected GT, got %v", expr.Infix().Operator)
	}
}

func TestParseEquality(t *testing.T) {
	expr, err := parseLetExpr("LET x = a == b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Infix() == nil || expr.Infix().Operator != OperatorEQU {
		t.Errorf("expected EQU, got %v", expr.Infix().Operator)
	}
}

func TestParseNotEqual(t *testing.T) {
	expr, err := parseLetExpr("LET x = a != b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Infix() == nil || expr.Infix().Operator != OperatorNEQ {
		t.Errorf("expected NEQ, got %v", expr.Infix().Operator)
	}
}

func TestParseSubscript(t *testing.T) {
	expr, err := parseLetExpr("LET x = arr[i]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Suffix() == nil || expr.Suffix().Operator != OperatorINDEX {
		t.Fatalf("expected INDEX suffix, got %v", expr.Suffix())
	}
	if len(expr.Suffix().Operands) != 2 {
		t.Fatalf("expected 2 operands (array + index), got %d", len(expr.Suffix().Operands))
	}
	// First operand: the array expression (arr)
	arr := expr.Suffix().Operands[0].Expr()
	if arr == nil || arr.Operand() == nil || arr.Ref() == nil {
		t.Fatal("expected array reference")
	}
	if arr.Ref().Identifier != Identifier("arr") {
		t.Errorf("expected arr, got %s", arr.Ref().Identifier)
	}
	// Second operand: the index (i)
	idx := expr.Suffix().Operands[1].Expr()
	if idx == nil || idx.Operand() == nil || idx.Ref() == nil {
		t.Fatal("expected index reference")
	}
	if idx.Ref().Identifier != Identifier("i") {
		t.Errorf("expected i, got %s", idx.Ref().Identifier)
	}
}

func TestParseSubscriptExpr(t *testing.T) {
	expr, err := parseLetExpr("LET x = arr[i + 1]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Suffix() == nil || expr.Suffix().Operator != OperatorINDEX {
		t.Fatalf("expected INDEX suffix")
	}
	if len(expr.Suffix().Operands) != 2 {
		t.Fatalf("expected 2 operands, got %d", len(expr.Suffix().Operands))
	}
	// Index should be ADD (i + 1)
	idx := expr.Suffix().Operands[1].Expr()
	if idx == nil || idx.Infix() == nil || idx.Infix().Operator != OperatorADD {
		t.Errorf("expected ADD in index, got %v", idx.Infix())
	}
}

func TestParseFuncCall(t *testing.T) {
	expr, err := parseLetExpr("LET x = foo(a, b)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Suffix() == nil || expr.Suffix().Operator != OperatorCALL {
		t.Fatalf("expected CALL suffix, got %v", expr.Suffix())
	}
	if len(expr.Suffix().Operands) != 3 {
		t.Fatalf("expected 3 operands (func + 2 args), got %d", len(expr.Suffix().Operands))
	}
	// First operand: the function expression (foo)
	fn := expr.Suffix().Operands[0].Expr()
	if fn == nil || fn.Operand() == nil || fn.Ref() == nil {
		t.Fatal("expected function reference")
	}
	if fn.Ref().Identifier != Identifier("foo") {
		t.Errorf("expected foo, got %s", fn.Ref().Identifier)
	}
}

func TestParseFuncCallNoArgs(t *testing.T) {
	expr, err := parseLetExpr("LET x = foo()")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Suffix() == nil || expr.Suffix().Operator != OperatorCALL {
		t.Fatalf("expected CALL suffix")
	}
	if len(expr.Suffix().Operands) != 1 {
		t.Fatalf("expected 1 operand (func only), got %d", len(expr.Suffix().Operands))
	}
}

func TestParseChainedSubscript(t *testing.T) {
	expr, err := parseLetExpr("LET x = a[i][j]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Outer is INDEX with a[i] as first operand
	if expr.Suffix() == nil || expr.Suffix().Operator != OperatorINDEX {
		t.Fatalf("expected INDEX suffix")
	}
	// First operand of outer should be another INDEX
	inner := expr.Suffix().Operands[0].Expr()
	if inner == nil || inner.Suffix() == nil || inner.Suffix().Operator != OperatorINDEX {
		t.Fatal("expected nested INDEX")
	}
}

func TestParseCallThenSubscript(t *testing.T) {
	expr, err := parseLetExpr("LET x = f(a)[i]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Outer is INDEX; first operand is CALL
	if expr.Suffix() == nil || expr.Suffix().Operator != OperatorINDEX {
		t.Fatalf("expected INDEX suffix")
	}
	inner := expr.Suffix().Operands[0].Expr()
	if inner == nil || inner.Suffix() == nil || inner.Suffix().Operator != OperatorCALL {
		t.Fatal("expected CALL inside INDEX")
	}
}

type letResult struct {
	Let
	err error
}

func parseLetStmt(src string) letResult {
	tokens, err := Scan(strings.NewReader(src))
	if err != nil {
		return letResult{err: err}
	}
	p := NewParser(tokens)
	var s Statement
	err = s.Parse(p)
	if err != nil {
		return letResult{err: err}
	}

	if let, ok := s.Command.(Let); !ok {
		return letResult{err: Error{Message: "expected Let statement"}}
	} else {
		return letResult{Let: let}
	}
}

func TestLetSimpleVar(t *testing.T) {
	r := parseLetStmt("LET x = 42")
	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	if r.Identifier != Identifier("x") {
		t.Errorf("expected x, got %s", r.Identifier)
	}
	if len(r.Subscripts) != 0 {
		t.Errorf("expected no subscripts, got %d", len(r.Subscripts))
	}
}

func TestLetArraySet(t *testing.T) {
	r := parseLetStmt("LET arr[i] = 5")
	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	if r.Identifier != Identifier("arr") {
		t.Errorf("expected arr, got %s", r.Identifier)
	}
	if len(r.Subscripts) != 1 {
		t.Fatalf("expected 1 subscript, got %d", len(r.Subscripts))
	}
	sub := r.Subscripts[0]
	if sub.Operand() == nil || sub.Operand().Reference() == nil {
		t.Fatal("expected reference subscript")
	}
	if sub.Operand().Reference().Identifier != Identifier("i") {
		t.Errorf("expected i, got %s", sub.Operand().Reference().Identifier)
	}
}

func TestLetArraySetExprIndex(t *testing.T) {
	r := parseLetStmt("LET arr[i + 1] = x")
	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	if len(r.Subscripts) != 1 {
		t.Fatalf("expected 1 subscript, got %d", len(r.Subscripts))
	}
	sub := r.Subscripts[0]
	if sub.Infix() == nil || sub.Infix().Operator != OperatorADD {
		t.Errorf("expected ADD in index, got %v", sub.Infix())
	}
}

func TestLetArraySetChained(t *testing.T) {
	r := parseLetStmt("LET arr[i][j] = 42")
	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	if r.Identifier != Identifier("arr") {
		t.Errorf("expected arr, got %s", r.Identifier)
	}
	if len(r.Subscripts) != 2 {
		t.Fatalf("expected 2 subscripts, got %d", len(r.Subscripts))
	}
}

func TestLetArraySetRHSWithSubscript(t *testing.T) {
	r := parseLetStmt("LET arr[i] = other[j]")
	if r.err != nil {
		t.Fatalf("unexpected error: %v", r.err)
	}
	if len(r.Subscripts) != 1 {
		t.Fatalf("expected 1 subscript on LHS, got %d", len(r.Subscripts))
	}
	// RHS should be a subscript expression
	if r.Expression.Suffix() == nil || r.Expression.Suffix().Operator != OperatorINDEX {
		t.Errorf("expected INDEX on RHS, got %v", r.Expression.Suffix())
	}
}

// parseStmt parses a full statement (IF, GROUP, etc.)
func parseStmt(src string) (Statement, error) {
	tokens, err := Scan(strings.NewReader(src))
	if err != nil {
		return Statement{}, err
	}
	p := NewParser(tokens)
	var s Statement
	err = s.Parse(p)
	return s, err
}

func TestParseIfThen(t *testing.T) {
	s, err := parseStmt("IF x THEN LET y = 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sIf, ok := s.Command.(If)
	if !ok {
		t.Fatal("expected If statement")
	}
	if sIf.Condition.Operand() == nil || sIf.Condition.Ref() == nil {
		t.Fatal("expected reference condition")
	}
	if _, ok := sIf.Then.Command.(Let); !ok {
		t.Fatal("expected Then to be a Let")
	}
	if sIf.Else != nil {
		t.Fatal("expected no Else")
	}
}

func TestParseIfThenElse(t *testing.T) {
	s, err := parseStmt("IF x THEN LET y = 1 ELSE LET z = 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sIf, ok := s.Command.(If)
	if !ok {
		t.Fatal("expected If statement")
	}
	if sIf.Else == nil {
		t.Fatal("expected Else")
	}
	if _, ok := sIf.Then.Command.(Let); !ok {
		t.Fatal("expected Else to be a Let")
	}

	if let, ok := sIf.Else.Command.(Let); !ok {
		t.Fatal("expected Else to be a Let")

	} else {
		if let.Identifier != Identifier("z") {
			t.Errorf("expected z, got %s", let.Identifier)
		}
	}
}

func TestParseGroupDo(t *testing.T) {
	s, err := parseStmt("DO LET x = 1 END")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sGroup, ok := s.Command.(Group)
	if !ok {
		t.Fatal("expected Group")
	}
	if sGroup.While != nil || sGroup.For != nil || sGroup.Case != nil {
		t.Fatal("expected bare DO group")
	}
	if len(sGroup.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(sGroup.Statements))
	}
}

func TestParseGroupWhile(t *testing.T) {
	s, err := parseStmt("WHILE a DO LET x = 1 END")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sGroup, ok := s.Command.(Group)
	if !ok {
		t.Fatal("expected Group")
	}
	if sGroup.While == nil {
		t.Fatal("expected While")
	}
	if len(sGroup.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(sGroup.Statements))
	}
}

func TestParseGroupFor(t *testing.T) {
	s, err := parseStmt("FOR i = 1 TO 10 DO LET x = i END")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sGroup, ok := s.Command.(Group)
	if !ok {
		t.Fatal("expected Group")
	}

	if sGroup.For == nil {
		t.Fatal("expected For")
	}
	if sGroup.For.Reference.Identifier != Identifier("i") {
		t.Errorf("expected i, got %s", sGroup.For.Reference.Identifier)
	}
	if sGroup.For.By != nil {
		t.Fatal("expected no BY clause")
	}
}

func TestParseGroupForBy(t *testing.T) {
	s, err := parseStmt("FOR i = 1 TO 10 BY 2 DO LET x = i END")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sGroup, ok := s.Command.(Group)
	if !ok {
		t.Fatal("expected Group")
	}

	if sGroup.For == nil {
		t.Fatal("expected For")
	}

	if sGroup.For.By == nil {
		t.Fatal("expected BY clause")
	}
}

func TestParseArrayDeclareUnbounded(t *testing.T) {
	tokens, err := Scan(strings.NewReader("DECLARE arr ARRAY WORD"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var s Statement
	p := NewParser(tokens)
	if err := s.Parse(p); err != nil {
		t.Fatalf("parse: %v", err)
	}

	sDeclare, ok := s.Command.(Declare)
	if !ok {
		t.Fatal("expected Declare statement")
	}

	arr := sDeclare.Type.Array()
	if arr == nil {
		t.Fatal("expected array type")
	}
	if arr.Size != 0 {
		t.Errorf("expected unbounded size 0, got %d", arr.Size)
	}
	if arr.ElemType.Predeclared() != PredeclaredWord {
		t.Error("expected WORD type")
	}
}

func TestParseArrayDeclareFixed(t *testing.T) {
	tokens, err := Scan(strings.NewReader("DECLARE arr ARRAY [10] BYTE"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var s Statement
	p := NewParser(tokens)
	if err := s.Parse(p); err != nil {
		t.Fatalf("parse: %v", err)
	}
	sDeclare, ok := s.Command.(Declare)
	if !ok {
		t.Fatal("expected Declare statement")
	}

	arr := sDeclare.Type.Array()
	if arr == nil {
		t.Fatal("expected array type")
	}
	if arr.Size != 10 {
		t.Errorf("expected size 10, got %d", arr.Size)
	}
	if arr.ElemType.Predeclared() != PredeclaredByte {
		t.Error("expected BYTE type")
	}
}

func TestParseArrayDeclareMultiDim(t *testing.T) {
	tokens, err := Scan(strings.NewReader("DECLARE arr ARRAY [6] WORD"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var s Statement
	p := NewParser(tokens)
	if err := s.Parse(p); err != nil {
		t.Fatalf("parse: %v", err)
	}

	sDeclare, ok := s.Command.(Declare)
	if !ok {
		t.Fatal("expected Declare statement")
	}
	arr := sDeclare.Type.Array()
	if arr == nil {
		t.Fatal("expected array type")
	}
	if arr.Size != 6 {
		t.Errorf("expected size 6, got %d", arr.Size)
	}
}

func TestParseProcBasic(t *testing.T) {
	tokens, err := Scan(strings.NewReader("PROCEDURE foo\nRETURN\nEND"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var s Statement
	p := NewParser(tokens)
	if err := s.Parse(p); err != nil {
		t.Fatalf("parse: %v", err)
	}

	sProcedure, ok := s.Command.(Procedure)
	if !ok {
		t.Fatal("expected Procedure statement")
	}
	if sProcedure.Name.Name != "foo" {
		t.Errorf("expected name foo, got %s", sProcedure.Name.Name)
	}
}

func TestParseProcParams(t *testing.T) {
	tokens, err := Scan(strings.NewReader("PROCEDURE add (x WORD, y WORD) WORD\nRETURN x + y\nEND"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var s Statement
	p := NewParser(tokens)
	if err := s.Parse(p); err != nil {
		t.Fatalf("parse: %v", err)
	}
	sProcedure, ok := s.Command.(Procedure)
	if !ok {
		t.Fatal("expected Procedure statement")
	}
	if len(sProcedure.Parameters) != 2 {
		t.Errorf("expected 2 params, got %d", len(sProcedure.Parameters))
	}
	if sProcedure.Type.Predeclared() != PredeclaredWord {
		t.Error("expected WORD return type")
	}
}

func TestParseProcReentrant(t *testing.T) {
	tokens, err := Scan(strings.NewReader("PROCEDURE foo (a WORD, b WORD, c WORD) WORD REENTRANT RETURN a END"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var s Statement
	p := NewParser(tokens)
	if err := s.Parse(p); err != nil {
		t.Fatalf("parse: %v", err)
	}
	sProcedure, ok := s.Command.(Procedure)
	if !ok {
		t.Fatal("expected Procedure statement")
	}
	if !sProcedure.Reentrant {
		t.Error("expected REENTRANT")
	}
	if len(sProcedure.Parameters) != 3 {
		t.Errorf("expected 3 params, got %d", len(sProcedure.Parameters))
	}
}

func TestParseCallWithArgs(t *testing.T) {
	tokens, err := Scan(strings.NewReader("CALL foo(1, 2)"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var s Statement
	p := NewParser(tokens)
	if err := s.Parse(p); err != nil {
		t.Fatalf("parse: %v", err)
	}

	sCall, ok := s.Command.(Call)
	if !ok {
		t.Fatal("expected Call statement")
	}

	if string(sCall.Identifier) != "foo" {
		t.Errorf("expected name foo, got %s", string(sCall.Identifier))
	}
	if len(sCall.Arguments) != 2 {
		t.Errorf("expected 2 arguments, got %d", len(sCall.Arguments))
	}
}

/*
func TestParseReturnMulti(t *testing.T) {
	tokens, err := Scan(strings.NewReader("RETURN 1, 2"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var s Statement
	p := NewParser(tokens)
	if err := s.Parse(p); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.Return == nil {
		t.Fatal("expected Return statement")
	}
	if len(s.Return.Expressions) != 2 {
		t.Errorf("expected 2 return expressions, got %d", len(s.Return.Expressions))
	}
}
*/

func TestInclude(t *testing.T) {
	dir := t.TempDir()
	// Write the included file.
	incPath := dir + "/lib.plz"
	if err := os.WriteFile(incPath, []byte("DECLARE x BYTE\nCONSTANT FOO = 42\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Write the main file with an INCLUDE directive.
	mainPath := dir + "/main.plz"
	mainSrc := "INCLUDE \"lib.plz\"\nDECLARE y WORD\n"
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	prog, err := ParseFile(mainPath)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(prog.Statements) != 3 {
		t.Fatalf("expected 3 statements (DECLARE x, CONSTANT FOO, DECLARE y), got %d", len(prog.Statements))
	}
}

/*
func TestIncludeTypeAliases(t *testing.T) {
	// Verify that type aliases from DEFINE in the including file
	// are visible in the included file (shared TypeAliases map).
	dir := t.TempDir()
	incPath := dir + "/lib.plz"
	// This file uses TYPE TEXT (a built-in alias) and TYPE my_point from the main file.
	if err := os.WriteFile(incPath, []byte("DECLARE msg TYPE TEXT\nDECLARE point TYPE my_point\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mainPath := dir + "/main.plz"
	mainSrc := "DEFINE my_point RECORD x BYTE, y BYTE END\nINCLUDE \"lib.plz\"\n"
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}
	prog, err := ParseFile(mainPath)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(prog.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(prog.Statements))
	}
	// stmt[0] is DEFINE, stmt[1] is DECLARE msg TEXT, stmt[2] is DECLARE point my_point
	if prog.Statements[1].Declare == nil || prog.Statements[1].Declare.Identifier != Identifier("msg") {
		t.Error("expected DECLARE msg")
	}
	if prog.Statements[1].Declare.Type.Record == nil {
		t.Error("expected TEXT record type")
	}
	if prog.Statements[2].Declare == nil || prog.Statements[2].Declare.Identifier != Identifier("point") {
		t.Error("expected DECLARE point")
	}
}

func TestIncludeNested(t *testing.T) {
	// Verify nested includes work.
	dir := t.TempDir()
	// inner.plz — included by middle.plz
	if err := os.WriteFile(dir+"/inner.plz", []byte("DECLARE inner_var BYTE\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// middle.plz — includes inner.plz, included by main.plz
	if err := os.WriteFile(dir+"/middle.plz", []byte("INCLUDE \"inner.plz\"\nDECLARE middle_var BYTE\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// main.plz — includes middle.plz
	mainPath := dir + "/main.plz"
	if err := os.WriteFile(mainPath, []byte("INCLUDE \"middle.plz\"\nDECLARE main_var BYTE\n"), 0644); err != nil {
		t.Fatal(err)
	}
	prog, err := ParseFile(mainPath)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(prog.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(prog.Statements))
	}
	if prog.Statements[0].Declare == nil || prog.Statements[0].Declare.Identifier != Identifier("inner_var") {
		t.Error("expected DECLARE inner_var as first statement")
	}
	if prog.Statements[1].Declare == nil || prog.Statements[1].Declare.Identifier != Identifier("middle_var") {
		t.Error("expected DECLARE middle_var as second statement")
	}
	if prog.Statements[2].Declare == nil || prog.Statements[2].Declare.Identifier != Identifier("main_var") {
		t.Error("expected DECLARE main_var as third statement")
	}
}

func TestSubscriptNoIndex(t *testing.T) {
	// arr[] should parse with just the array as the only operand (no index)
	expr, err := parseLetExpr("LET x = arr[]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Suffix == nil || expr.Suffix.Operator != OperatorINDEX {
		t.Fatalf("expected INDEX suffix")
	}
	if len(expr.Suffix.Operands) != 1 {
		t.Fatalf("expected 1 operand for empty subscript, got %d", len(expr.Suffix.Operands))
	}
}

func TestParseInterruptProc(t *testing.T) {
	src := "INTERRUPT PROCEDURE my_isr\nEND"
	tokens, err := Scan(strings.NewReader(src))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var p Program
	parser := NewParser(tokens)
	if err := p.Parse(parser); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(p.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(p.Statements))
	}
	proc := p.Statements[0].Procedure
	if proc == nil {
		t.Fatal("expected PROCEDURE")
	}
	if proc.Interrupt == nil || proc.Interrupt.NMI || proc.Interrupt.Interrupt != 1 {
		t.Fatal("expected INTERRUPT procedure")
	}
	if proc.Name.Name != "my_isr" {
		t.Fatalf("expected name my_isr, got %s", proc.Name.Name)
	}
}

func TestParseNMIProc(t *testing.T) {
	src := "NMI PROCEDURE my_nmi\nEND"
	tokens, err := Scan(strings.NewReader(src))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var p Program
	parser := NewParser(tokens)
	if err := p.Parse(parser); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(p.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(p.Statements))
	}
	proc := p.Statements[0].Procedure
	if proc == nil {
		t.Fatal("expected PROCEDURE")
	}
	if proc.Interrupt == nil || !proc.Interrupt.NMI {
		t.Fatal("expected NMI procedure")
	}
}
*/
func TestSMSTileFromString(t *testing.T) {
	tile, err := SMSTileFromString("........" +
		"\n...FF..." +
		"\n..F..F.." +
		"\n.F....F." +
		"\n.FFFFFF." +
		"\n.F....F." +
		"\n.F....F." +
		"\n........")
	if err != nil {
		t.Fatalf("SMSTileFromString: %v", err)
	}
	check := func(x, y int, want byte) {
		got, _ := tile.PaletteIdAt(x, y)
		if byte(got) != want {
			t.Errorf("tile.PaletteIdAt(%d,%d) = %d, want %d", x, y, got, want)
		}
	}
	check(0, 0, 0)
	check(3, 1, 15)
	check(4, 1, 15)
	check(2, 2, 15)
	check(5, 2, 15)
	check(1, 3, 15)
	check(6, 3, 15)
	check(1, 4, 15)
	check(2, 4, 15)
	check(3, 4, 15)
	check(4, 4, 15)
	check(5, 4, 15)
	check(6, 4, 15)
	check(1, 5, 15)
	check(6, 5, 15)
	check(1, 6, 15)
	check(6, 6, 15)
	check(7, 7, 0)
	buf := tile.Bytes()
	if len(buf) != 32 {
		t.Errorf("tile.Bytes() returned %d bytes, want 32", len(buf))
	}
}

func TestSMSTileFromStringSimple(t *testing.T) {
	tile, err := SMSTileFromString("11111111")
	if err != nil {
		t.Fatalf("SMSTileFromString: %v", err)
	}
	for x := 0; x < 8; x++ {
		got, _ := tile.PaletteIdAt(x, 0)
		if byte(got) != 1 {
			t.Errorf("tile.PaletteIdAt(%d,0) = %d, want 1", x, got)
		}
	}
	for y := 1; y < 8; y++ {
		for x := 0; x < 8; x++ {
			got, _ := tile.PaletteIdAt(x, y)
			if byte(got) != 0 {
				t.Errorf("tile.PaletteIdAt(%d,%d) = %d, want 0", x, y, got)
			}
		}
	}
}

func TestParseBank(t *testing.T) {
	src := "BANK 3\n"
	tokens, err := Scan(strings.NewReader(src))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var s Statement
	p := NewParser(tokens)
	if err := s.Parse(p); err != nil {
		t.Fatalf("parse: %v", err)
	}
	bank, ok := s.Command.(BankStmt)
	if !ok {
		t.Fatalf("expected BankStmt, got %T", s.Command)
	}
	if bank.Number.Operand() == nil || bank.Number.Operand().Literal() == nil || bank.Number.Operand().Literal().Number() == nil {
		t.Fatal("expected literal number")
	}
	if bank.Number.Operand().Literal().Number().Value != 3 {
		t.Errorf("expected 3, got %d", bank.Number.Operand().Literal().Number().Value)
	}
}

func TestParseAtBank(t *testing.T) {
	src := "AT BANK 2\n"
	tokens, err := Scan(strings.NewReader(src))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var s Statement
	p := NewParser(tokens)
	if err := s.Parse(p); err != nil {
		t.Fatalf("parse: %v", err)
	}
	at, ok := s.Command.(At)
	if !ok {
		t.Fatalf("expected At command, got %T", s.Command)
	}
	if !at.HasBank {
		t.Fatal("expected HasBank to be true")
	}
	if at.BankNumber != 2 {
		t.Errorf("at.BankNumber = %d, want 2", at.BankNumber)
	}
}

func TestSMSTileFromStringNewlines(t *testing.T) {
	tile, err := SMSTileFromString("\n.A.B.C\n.D.E.F\n")
	if err != nil {
		t.Fatalf("SMSTileFromString: %v", err)
	}
	check := func(x, y int, want byte) {
		got, _ := tile.PaletteIdAt(x, y)
		if byte(got) != want {
			t.Errorf("tile.PaletteIdAt(%d,%d) = %d, want %d", x, y, got, want)
		}
	}
	check(1, 0, 10)
	check(3, 0, 11)
	check(5, 0, 12)
	check(1, 1, 13)
	check(3, 1, 14)
	check(5, 1, 15)
}

func TestParseDataTile(t *testing.T) {
	src := "myfont: DATA TILE\n" +
		"`\n" +
		"........\n" +
		"...FF...\n" +
		"........\n" +
		"`\n"
	tokens, err := Scan(strings.NewReader(src))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	p := NewParser(tokens)
	prog := Program{}
	if err := prog.Parse(p); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	data, ok := prog.Statements[0].Command.(Data)
	if !ok {
		t.Fatalf("expected Data command, got %T", prog.Statements[0].Command)
	}
	if data.Name != "myfont" {
		t.Errorf("data.Name = %q, want %q", data.Name, "myfont")
	}
	if data.Tile == nil {
		t.Fatal("data.Tile is nil, expected Tile data")
	}
	if len(data.Tile.Tiles) != 1 {
		t.Fatalf("expected 1 tile, got %d", len(data.Tile.Tiles))
	}
	got, _ := data.Tile.Tiles[0].PaletteIdAt(3, 1)
	if byte(got) != 15 {
		t.Errorf("pixel (3,1) = %d, want 15", got)
	}
}
