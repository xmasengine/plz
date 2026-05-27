package plz

import (
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
	if s.Let == nil {
		panic("expected Let statement")
	}
	return &s.Let.Expression, nil
}

func TestParseSimpleNumber(t *testing.T) {
	expr, err := parseLetExpr("LET x = 42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Operand == nil || expr.Operand.Literal == nil || expr.Operand.Literal.Number == nil {
		t.Fatal("expected literal number")
	}
	if *expr.Operand.Literal.Number != 42 {
		t.Errorf("expected 42, got %d", *expr.Operand.Literal.Number)
	}
}

func TestParseIdentifier(t *testing.T) {
	expr, err := parseLetExpr("LET x = foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Operand == nil || expr.Operand.Reference == nil {
		t.Fatal("expected reference")
	}
	if expr.Operand.Reference.Identifier != Identifier("foo") {
		t.Errorf("expected foo, got %s", expr.Operand.Reference.Identifier)
	}
}

func TestParseBinaryExpr(t *testing.T) {
	expr, err := parseLetExpr("LET x = 1 + 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Infix == nil {
		t.Fatal("expected Infix expression")
	}
	if expr.Infix.Operator != OperatorADD {
		t.Errorf("expected ADD, got %v", expr.Infix.Operator)
	}
	// left: 1
	left := expr.Infix.Operands[0].Expression
	if left == nil || left.Operand == nil {
		t.Fatal("expected left operand")
	}
	// right: 2
	right := expr.Infix.Operands[1].Expression
	if right == nil || right.Operand == nil {
		t.Fatal("expected right operand")
	}
}

func TestParseNested(t *testing.T) {
	expr, err := parseLetExpr("LET x = 1 + 2 * 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Top is ADD (since + has lower priority than *)
	if expr.Infix == nil {
		t.Fatal("expected Infix")
	}
	if expr.Infix.Operator != OperatorADD {
		t.Errorf("expected ADD at top, got %v", expr.Infix.Operator)
	}
	// Right side should be MUL (2 * 3)
	right := expr.Infix.Operands[1].Expression
	if right == nil || right.Infix == nil {
		t.Fatal("expected Infix on right")
	}
	if right.Infix.Operator != OperatorMUL {
		t.Errorf("expected MUL on right, got %v", right.Infix.Operator)
	}
}

func TestParseParens(t *testing.T) {
	expr, err := parseLetExpr("LET x = (1 + 2) * 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Top is MUL
	if expr.Infix == nil || expr.Infix.Operator != OperatorMUL {
		t.Fatalf("expected MUL at top, got %c", expr.Infix.Operator)
	}
	// Left should be ADD (from parentheses)
	left := expr.Infix.Operands[0].Expression
	if left == nil || left.Infix == nil || left.Infix.Operator != OperatorADD {
		t.Errorf("expected ADD in parens, got %c", left.Infix.Operator)
	}
}

func TestParseUnaryNeg(t *testing.T) {
	expr, err := parseLetExpr("LET x = -5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Prefix == nil {
		t.Fatal("expected Prefix expression")
	}
	if expr.Prefix.Operator != OperatorNEG {
		t.Errorf("expected NEG, got %v", expr.Prefix.Operator)
	}
}

func TestParseUnaryNot(t *testing.T) {
	expr, err := parseLetExpr("LET x = !flag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Prefix == nil || expr.Prefix.Operator != OperatorNOT {
		t.Errorf("expected NOT, got %v", expr.Prefix.Operator)
	}
}

func TestParseComparison(t *testing.T) {
	expr, err := parseLetExpr("LET x = a > b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Infix == nil || expr.Infix.Operator != OperatorGT {
		t.Errorf("expected GT, got %v", expr.Infix.Operator)
	}
}

func TestParseEquality(t *testing.T) {
	expr, err := parseLetExpr("LET x = a == b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Infix == nil || expr.Infix.Operator != OperatorEQU {
		t.Errorf("expected EQU, got %v", expr.Infix.Operator)
	}
}

func TestParseNotEqual(t *testing.T) {
	expr, err := parseLetExpr("LET x = a != b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Infix == nil || expr.Infix.Operator != OperatorNEQ {
		t.Errorf("expected NEQ, got %v", expr.Infix.Operator)
	}
}

func TestParseSubscript(t *testing.T) {
	expr, err := parseLetExpr("LET x = arr[i]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Suffix == nil || expr.Suffix.Operator != OperatorINDEX {
		t.Fatalf("expected INDEX suffix, got %v", expr.Suffix)
	}
	if len(expr.Suffix.Operands) != 2 {
		t.Fatalf("expected 2 operands (array + index), got %d", len(expr.Suffix.Operands))
	}
	// First operand: the array expression (arr)
	arr := expr.Suffix.Operands[0].Expression
	if arr == nil || arr.Operand == nil || arr.Operand.Reference == nil {
		t.Fatal("expected array reference")
	}
	if arr.Operand.Reference.Identifier != Identifier("arr") {
		t.Errorf("expected arr, got %s", arr.Operand.Reference.Identifier)
	}
	// Second operand: the index (i)
	idx := expr.Suffix.Operands[1].Expression
	if idx == nil || idx.Operand == nil || idx.Operand.Reference == nil {
		t.Fatal("expected index reference")
	}
	if idx.Operand.Reference.Identifier != Identifier("i") {
		t.Errorf("expected i, got %s", idx.Operand.Reference.Identifier)
	}
}

func TestParseSubscriptExpr(t *testing.T) {
	expr, err := parseLetExpr("LET x = arr[i + 1]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Suffix == nil || expr.Suffix.Operator != OperatorINDEX {
		t.Fatalf("expected INDEX suffix")
	}
	if len(expr.Suffix.Operands) != 2 {
		t.Fatalf("expected 2 operands, got %d", len(expr.Suffix.Operands))
	}
	// Index should be ADD (i + 1)
	idx := expr.Suffix.Operands[1].Expression
	if idx == nil || idx.Infix == nil || idx.Infix.Operator != OperatorADD {
		t.Errorf("expected ADD in index, got %v", idx.Infix)
	}
}

func TestParseFuncCall(t *testing.T) {
	expr, err := parseLetExpr("LET x = foo(a, b)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Suffix == nil || expr.Suffix.Operator != OperatorCALL {
		t.Fatalf("expected CALL suffix, got %v", expr.Suffix)
	}
	if len(expr.Suffix.Operands) != 3 {
		t.Fatalf("expected 3 operands (func + 2 args), got %d", len(expr.Suffix.Operands))
	}
	// First operand: the function expression (foo)
	fn := expr.Suffix.Operands[0].Expression
	if fn == nil || fn.Operand == nil || fn.Operand.Reference == nil {
		t.Fatal("expected function reference")
	}
	if fn.Operand.Reference.Identifier != Identifier("foo") {
		t.Errorf("expected foo, got %s", fn.Operand.Reference.Identifier)
	}
}

func TestParseFuncCallNoArgs(t *testing.T) {
	expr, err := parseLetExpr("LET x = foo()")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if expr.Suffix == nil || expr.Suffix.Operator != OperatorCALL {
		t.Fatalf("expected CALL suffix")
	}
	if len(expr.Suffix.Operands) != 1 {
		t.Fatalf("expected 1 operand (func only), got %d", len(expr.Suffix.Operands))
	}
}

func TestParseChainedSubscript(t *testing.T) {
	expr, err := parseLetExpr("LET x = a[i][j]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Outer is INDEX with a[i] as first operand
	if expr.Suffix == nil || expr.Suffix.Operator != OperatorINDEX {
		t.Fatalf("expected INDEX suffix")
	}
	// First operand of outer should be another INDEX
	inner := expr.Suffix.Operands[0].Expression
	if inner == nil || inner.Suffix == nil || inner.Suffix.Operator != OperatorINDEX {
		t.Fatal("expected nested INDEX")
	}
}

func TestParseCallThenSubscript(t *testing.T) {
	expr, err := parseLetExpr("LET x = f(a)[i]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Outer is INDEX; first operand is CALL
	if expr.Suffix == nil || expr.Suffix.Operator != OperatorINDEX {
		t.Fatalf("expected INDEX suffix")
	}
	inner := expr.Suffix.Operands[0].Expression
	if inner == nil || inner.Suffix == nil || inner.Suffix.Operator != OperatorCALL {
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
	if s.Let == nil {
		return letResult{err: Error{Message: "expected Let statement"}}
	}
	return letResult{Let: *s.Let}
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
	if sub.Operand == nil || sub.Operand.Reference == nil {
		t.Fatal("expected reference subscript")
	}
	if sub.Operand.Reference.Identifier != Identifier("i") {
		t.Errorf("expected i, got %s", sub.Operand.Reference.Identifier)
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
	if sub.Infix == nil || sub.Infix.Operator != OperatorADD {
		t.Errorf("expected ADD in index, got %v", sub.Infix)
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
	if r.Expression.Suffix == nil || r.Expression.Suffix.Operator != OperatorINDEX {
		t.Errorf("expected INDEX on RHS, got %v", r.Expression.Suffix)
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
	if s.If == nil {
		t.Fatal("expected If statement")
	}
	if s.If.Condition.Operand == nil || s.If.Condition.Operand.Reference == nil {
		t.Fatal("expected reference condition")
	}
	if s.If.Then.Let == nil {
		t.Fatal("expected Then to be a Let")
	}
	if s.If.Else != nil {
		t.Fatal("expected no Else")
	}
}

func TestParseIfThenElse(t *testing.T) {
	s, err := parseStmt("IF x THEN LET y = 1 ELSE LET z = 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.If == nil {
		t.Fatal("expected If statement")
	}
	if s.If.Else == nil {
		t.Fatal("expected Else")
	}
	if s.If.Else.Let == nil {
		t.Fatal("expected Else to be a Let")
	}
	if s.If.Else.Let.Identifier != Identifier("z") {
		t.Errorf("expected z, got %s", s.If.Else.Let.Identifier)
	}
}

func TestParseGroupDo(t *testing.T) {
	s, err := parseStmt("DO LET x = 1 END")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Group == nil {
		t.Fatal("expected Group")
	}
	if s.Group.While != nil || s.Group.For != nil || s.Group.Case != nil {
		t.Fatal("expected bare DO group")
	}
	if len(s.Group.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(s.Group.Statements))
	}
}

func TestParseGroupWhile(t *testing.T) {
	s, err := parseStmt("WHILE a DO LET x = 1 END")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Group == nil {
		t.Fatal("expected Group")
	}
	if s.Group.While == nil {
		t.Fatal("expected While")
	}
	if len(s.Group.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(s.Group.Statements))
	}
}

func TestParseGroupFor(t *testing.T) {
	s, err := parseStmt("FOR i = 1 TO 10 DO LET x = i END")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Group == nil {
		t.Fatal("expected Group")
	}
	if s.Group.For == nil {
		t.Fatal("expected For")
	}
	if s.Group.For.Reference.Identifier != Identifier("i") {
		t.Errorf("expected i, got %s", s.Group.For.Reference.Identifier)
	}
	if s.Group.For.By != nil {
		t.Fatal("expected no BY clause")
	}
}

func TestParseGroupForBy(t *testing.T) {
	s, err := parseStmt("FOR i = 1 TO 10 BY 2 DO LET x = i END")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Group == nil || s.Group.For == nil {
		t.Fatal("expected For")
	}
	if s.Group.For.By == nil {
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
	if s.Declare == nil {
		t.Fatal("expected Declare statement")
	}
	if len(s.Declare.Dims) != 1 || s.Declare.Dims[0] != 0 {
		t.Errorf("expected 1 unbounded dim, got %v", s.Declare.Dims)
	}
	if s.Declare.Size != 0 {
		t.Errorf("expected unbounded size 0, got %d", s.Declare.Size)
	}
	if s.Declare.Type.Predeclared != PredeclaredWord {
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
	if s.Declare == nil {
		t.Fatal("expected Declare statement")
	}
	if len(s.Declare.Dims) != 1 || s.Declare.Dims[0] != 10 {
		t.Errorf("expected [10], got %v", s.Declare.Dims)
	}
	if s.Declare.Size != 10 {
		t.Errorf("expected size 10, got %d", s.Declare.Size)
	}
	if s.Declare.Type.Predeclared != PredeclaredByte {
		t.Error("expected BYTE type")
	}
}

func TestParseArrayDeclareMultiDim(t *testing.T) {
	tokens, err := Scan(strings.NewReader("DECLARE arr ARRAY [3, 4] WORD"))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	var s Statement
	p := NewParser(tokens)
	if err := s.Parse(p); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.Declare == nil {
		t.Fatal("expected Declare statement")
	}
	if len(s.Declare.Dims) != 2 || s.Declare.Dims[0] != 3 || s.Declare.Dims[1] != 4 {
		t.Errorf("expected [3, 4], got %v", s.Declare.Dims)
	}
	if s.Declare.Size != 12 {
		t.Errorf("expected size 12, got %d", s.Declare.Size)
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
	if s.Procedure == nil {
		t.Fatal("expected Procedure statement")
	}
	if s.Procedure.Name.Name != "foo" {
		t.Errorf("expected name foo, got %s", s.Procedure.Name.Name)
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
	if s.Procedure == nil {
		t.Fatal("expected Procedure statement")
	}
	if len(s.Procedure.Parameters) != 2 {
		t.Errorf("expected 2 params, got %d", len(s.Procedure.Parameters))
	}
	if s.Procedure.Type.Predeclared != PredeclaredWord {
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
	if s.Procedure == nil {
		t.Fatal("expected Procedure statement")
	}
	if !s.Procedure.Reentrant {
		t.Error("expected REENTRANT")
	}
	if len(s.Procedure.Parameters) != 3 {
		t.Errorf("expected 3 params, got %d", len(s.Procedure.Parameters))
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
	if s.Call == nil {
		t.Fatal("expected Call statement")
	}
	if s.Call.Name != "foo" {
		t.Errorf("expected name foo, got %s", s.Call.Name)
	}
	if len(s.Call.Arguments) != 2 {
		t.Errorf("expected 2 arguments, got %d", len(s.Call.Arguments))
	}
}

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
