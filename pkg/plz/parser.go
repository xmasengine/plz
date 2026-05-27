package plz

import "fmt"

type Error struct {
	Position
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Position, e.Message)
}

func (t Token) Errorf(form string, args ...any) Error {
	return Error{Position: t.Position, Message: fmt.Sprintf(form, args...)}
}

// Parser for PL?Z.
// The design is a peek/parse parser with unlimited lookahead.
// Each AST node parses itself using the parser and a Parse method.
// For child nodes each AST node has has to peek which child node it is,
// It is an error to call Parse on a AST node that is not currently available
// in the parser.
type Parser struct {
	Tokens  []Token
	Current int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{Tokens: tokens}
}

func (p Parser) Peek() Token {
	if p.Current >= len(p.Tokens) {
		return Token{TokenKind: TokenEOF}
	}

	return p.Tokens[p.Current]
}

func (p Parser) PeekAt(offset int) Token {
	if p.Current+offset >= len(p.Tokens) {
		return Token{TokenKind: TokenEOF}
	}
	return p.Tokens[p.Current+offset]
}

func (p *Parser) Next() Token {
	if p.Current >= len(p.Tokens) {
		return Token{TokenKind: TokenEOF}
	}

	res := p.Peek()
	p.Current++
	return res
}

func (p *Parser) End() bool {
	return p.Current >= len(p.Tokens)
}

func (p *Parser) Accept(kinds ...TokenKind) (*Token, error) {
	token := p.Next()
	for _, kind := range kinds {
		if kind == token.TokenKind {
			return &token, nil
		}
	}
	return nil, token.Errorf("Accept: unexpected token, not in %v", kinds)
}

func (p Parser) Have(kinds ...TokenKind) bool {
	for offset, kind := range kinds {
		token := p.PeekAt(offset)
		if kind == token.TokenKind {
			return true
		}
	}
	return false
}

func (p *Parser) Skip(kind TokenKind) *Token {
	t := p.Peek()
	if t.TokenKind == kind {
		p.Next()
		return &t
	}
	return nil
}

func ParseFile(name string) (*Program, error) {
	tokens, err := ScanFile(name)
	if err != nil {
		return nil, err
	}
	var res Program
	parser := &Parser{Tokens: tokens}
	err = res.Parse(parser)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (p *Program) Parse(parser *Parser) error {
	for !parser.End() {
		var s Statement
		err := s.Parse(parser)
		if err != nil {
			return err
		}
		p.Statements = append(p.Statements, s)
		parser.Skip(TokenKind(';'))
	}
	return nil
}

func (s *Statement) Parse(parser *Parser) error {
	var l Label
	// allow two labels , numeric and name
	for i := 0; i < 2; i++ {
		if l.Have(*parser) {
			err := l.Parse(parser)
			if err != nil {
				return nil
			}
			s.Label = &l
		}
	}
	tok := parser.Peek()
	switch tok.TokenKind {
	case KeywordCall:
		s.Call = &Call{}
		return s.Call.Parse(parser)
	case KeywordConstant:
		s.Constant = &Constant{}
		return s.Constant.Parse(parser)
	case KeywordData:
		s.Data = &Data{}
		return s.Data.Parse(parser)
	case KeywordDeclare:
		s.Declare = &Declare{}
		return s.Declare.Parse(parser)
	case KeywordDisable:
		s.Disable = &Disable{}
		return s.Disable.Parse(parser)
	case KeywordEnable:
		s.Enable = &Enable{}
		return s.Enable.Parse(parser)
	case KeywordGoTo:
		s.GoTo = &GoTo{}
		return s.GoTo.Parse(parser)
	case KeywordHalt:
		s.Halt = &Halt{}
		return s.Halt.Parse(parser)
	case KeywordIf:
		s.If = &If{}
		return s.If.Parse(parser)
	case KeywordDo, KeywordWhile, KeywordFor, KeywordCase:
		s.Group = &Group{}
		return s.Group.Parse(parser)
	case KeywordLet:
		s.Let = &Let{}
		return s.Let.Parse(parser)
	case KeywordOutput:
		s.Output = &Output{}
		return s.Output.Parse(parser)
	case KeywordReturn:
		s.Return = &Return{}
		return s.Return.Parse(parser)
	default:
		return tok.Errorf("Statement: unexpected token %v", tok)
	}

	return nil
}

// Returns true if we have a label false if not. May only peek, peekAt or Have.
func (l *Label) Have(parser Parser) bool {
	if parser.Have(TokenInt, ':') {
		return true
	}
	if parser.Have(TokenIdent, ':') {
		return true
	}
	return false
}

func (l *Label) Parse(parser *Parser) error {
	tok, err := parser.Accept(TokenInt, TokenIdent)
	if err != nil {
		return err
	} else if tok.TokenKind == TokenInt {
		l.Location = tok.Number
	} else if tok.TokenKind == TokenIdent {
		l.Name = tok.Text
	}
	_, err = parser.Accept(TokenKind(':'))
	return err
}

func (g *Halt) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordHalt)
	return err
}

func (g *Constant) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordConstant)
	if err != nil {
		return nil
	}

	name, err := parser.Accept(TokenIdent)
	if err != nil {
		return nil
	}
	g.Name = name.Text

	parser.Skip(TokenKind('=')) // Skip optional =

	return g.Literal.Parse(parser)
}

func (g *Data) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordData)
	if err != nil {
		return nil
	}
	return g.Literal.Parse(parser)
}

func (i *Identifier) Parse(parser *Parser) error {
	tok, err := parser.Accept(TokenIdent)
	if err != nil {
		return err
	}
	*i = Identifier(tok.Text)
	return nil
}

func (r *Reference) Parse(parser *Parser) error {
	err := r.Identifier.Parse(parser)
	if err != nil {
		return err
	}
	for {
		switch parser.Peek().TokenKind {
		case '[':
			parser.Next()
			var subscript Expression
			if err := subscript.Parse(parser); err != nil {
				return err
			}
			if _, err := parser.Accept(TokenKind(']')); err != nil {
				return err
			}
			r.Subscripts = append(r.Subscripts, subscript)
		case '.':
			parser.Next()
			fieldTok, err := parser.Accept(TokenIdent)
			if err != nil {
				return err
			}
			r.Fields = append(r.Fields, Identifier(fieldTok.Text))
		default:
			return nil
		}
	}
}

func (g *Literal) Parse(parser *Parser) error {
	tok, err := parser.Accept(TokenInt, TokenString, TokenIdent)
	if err != nil {
		return nil
	}

	if tok.TokenKind == TokenInt {
		g.Number = &tok.Number
	} else if tok.TokenKind == TokenString {
		g.Text = &tok.Text
		println("string literal", *g.Text)
	} else if tok.TokenKind == TokenIdent {
		ref := &Reference{Identifier: Identifier(tok.Text)}
		g.Reference = ref
	}
	return nil
}

func (g *Disable) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordDisable)
	return err
}

func (g *Enable) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordEnable)
	return err
}

func (g *GoTo) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordGoTo)
	if err != nil {
		return err
	}
	tok, err := parser.Accept(TokenInt, TokenIdent)
	if err != nil {
		return err
	} else if tok.TokenKind == TokenInt {
		g.Location = tok.Number
	} else if tok.TokenKind == TokenIdent {
		g.Name = tok.Text
	}
	return nil
}

func (g *Call) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordCall)
	if err != nil {
		return err
	}
	tok, err := parser.Accept(TokenInt, TokenIdent)
	if err != nil {
		return err
	} else if tok.TokenKind == TokenInt {
		g.Location = tok.Number
	} else if tok.TokenKind == TokenIdent {
		g.Name = tok.Text
	}
	return nil
}

func (g *Return) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordReturn)
	return err
}

func (g *Output) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordOutput)
	if err != nil {
		return err
	}
	tok, err := parser.Accept(TokenInt)
	if err != nil {
		return err
	}
	g.Port = tok.Number
	return g.Value.Parse(parser)
}

func (t *Type) Parse(parser *Parser) error {
	if parser.Peek().TokenKind == KeywordStruct {
		parser.Next()
		t.Predeclared = PredeclaredNone
		t.Struct = &Struct{}
		for parser.Peek().TokenKind == TokenIdent {
			var f Field
			if err := f.Identifier.Parse(parser); err != nil {
				return err
			}
			if err := f.Type.Parse(parser); err != nil {
				return err
			}
			t.Struct.Fields = append(t.Struct.Fields, f)
			if parser.Peek().TokenKind != TokenKind(',') {
				break
			}
			parser.Next()
		}
		return nil
	}
	tok, err := parser.Accept(KeywordByte, KeywordWord)
	if err != nil {
		return err
	}
	switch tok.TokenKind {
	case KeywordByte:
		t.Predeclared = PredeclaredByte
	case KeywordWord:
		t.Predeclared = PredeclaredWord
	}
	return nil
}

func (g *Declare) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordDeclare)
	if err != nil {
		return err
	}
	err = g.Identifier.Parse(parser)
	if err != nil {
		return err
	}
	err = g.Type.Parse(parser)
	if err != nil {
		return err
	}

	return nil
}

func (g *Let) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordLet)
	if err != nil {
		return err
	}
	err = g.Reference.Parse(parser)
	if err != nil {
		return err
	}
	_, err = parser.Accept(TokenKind('='))
	if err != nil {
		return err
	}
	return g.Expression.Parse(parser)
}

// PeekOperator looks at the next token(s) without consuming them.
// If not an operator returns OperatorNone.
func (p *Parser) PeekOperator() Operator {
	tok := p.Peek()
	switch tok.TokenKind {
	case '+':
		return OperatorADD
	case '-':
		return OperatorSUB
	case '*':
		return OperatorMUL
	case '/':
		return OperatorDIV
	case '%':
		return OperatorMOD
	case '&':
		return OperatorAND
	case '|':
		return OperatorOR
	case '^':
		return OperatorXOR
	case '=':
		return OperatorEQU
	case '!':
		if p.PeekAt(1).TokenKind == '=' {
			return OperatorNEQ
		}
		return OperatorNone
	case '<':
		if p.PeekAt(1).TokenKind == '=' {
			return OperatorLTE
		}
		return OperatorLT
	case '>':
		if p.PeekAt(1).TokenKind == '=' {
			return OperatorGTE
		}
		return OperatorGT
	}
	return OperatorNone
}

// ReadOperator consumestokens and returns the operator found.
// Multi-character operators consume two or three tokens.
// If no operator is present, returns OperatorNone and consumes no tokens.
func (p *Parser) ReadOperator() (op Operator) {
	op = p.PeekOperator()
	switch op {
	case OperatorEQU, OperatorNEQ, OperatorGTE, OperatorLTE:
		p.Next()
		p.Next()
	case OperatorNone:
		return OperatorNone
	default:
		p.Next()
	}
	return op
}

// ParseExpr is the core Pratt parser. It reads tokens starting from the
// current position and builds an Expression tree.  The minBp parameter sets
// the minimum binding-power that the expression must have; operators with
// lower priority cause the loop to stop.
func (left *Expression) ParseExpr(p *Parser, minBp int) error {
	tok := p.Peek()

	switch tok.TokenKind {
	case TokenInt:
		p.Next()
		num := tok.Number
		left.Operand = &Operand{Literal: &Literal{Number: &num}}

	case TokenString, TokenChar:
		p.Next()
		n := tok.Number
		left.Operand = &Operand{Literal: &Literal{Number: &n}}

	case TokenIdent:
		p.Next()
		left.Operand = &Operand{Reference: &Reference{Identifier: Identifier(tok.Text)}}

	case '(':
		p.Next()
		var sub Expression
		err := sub.ParseExpr(p, 0)
		if err != nil {
			return err
		}
		if _, err = p.Accept(TokenKind(')')); err != nil {
			return err
		}
		*left = sub

	case '+':
		// Unary plus – effectively a no-op, skip it.
		p.Next()
		err := left.ParseExpr(p, 0)
		if err != nil {
			return err
		}

	case '-':
		p.Next()
		var right Expression
		err := right.ParseExpr(p, OperatorNEG.Priority())
		if err != nil {
			return err
		}
		left.Prefix = &Prefix{
			Operator: OperatorNEG,
			Operand:  Operand{Expression: &right},
		}

	case '!':
		p.Next()
		var right Expression
		err := right.ParseExpr(p, OperatorNOT.Priority())
		if err != nil {
			return err
		}
		left.Prefix = &Prefix{
			Operator: OperatorNOT,
			Operand:  Operand{Expression: &right},
		}

	default:
		return tok.Errorf("unexpected token in expression: %v", tok)
	}

	for {
		tok := p.Peek()

		// Array subscript: expr[index]
		if tok.TokenKind == '[' {
			p.Next()
			if p.Peek().TokenKind == ']' {
				p.Next()
				prev := new(Expression)
				*prev = *left
				*left = Expression{}
				left.Suffix = &Suffix{
					Operator: OperatorINDEX,
					Operands: []Operand{{Expression: prev}},
				}
				continue
			}
			var index Expression
			if err := index.Parse(p); err != nil {
				return err
			}
			if _, err := p.Accept(TokenKind(']')); err != nil {
				return err
			}
			prev := new(Expression)
			*prev = *left
			indexCopy := new(Expression)
			*indexCopy = index
			*left = Expression{}
			left.Suffix = &Suffix{
				Operator: OperatorINDEX,
				Operands: []Operand{
					{Expression: prev},
					{Expression: indexCopy},
				},
			}
			continue
		}

		// Function call: expr(args...)
		if tok.TokenKind == '(' {
			p.Next()
			var args []Expression
			if p.Peek().TokenKind != ')' {
				for {
					var arg Expression
					if err := arg.Parse(p); err != nil {
						return err
					}
					args = append(args, arg)
					if p.Peek().TokenKind == ')' {
						break
					}
					if _, err := p.Accept(TokenKind(',')); err != nil {
						return err
					}
				}
			}
			if _, err := p.Accept(TokenKind(')')); err != nil {
				return err
			}
			prev := new(Expression)
			*prev = *left
			operands := make([]Operand, 0, 1+len(args))
			operands = append(operands, Operand{Expression: prev})
			for i := range args {
				ac := new(Expression)
				*ac = args[i]
				operands = append(operands, Operand{Expression: ac})
			}
			*left = Expression{}
			left.Suffix = &Suffix{
				Operator: OperatorCALL,
				Operands: operands,
			}
			continue
		}

		// Struct field access: expr.field
		if tok.TokenKind == '.' {
			p.Next()
			fieldTok, err := p.Accept(TokenIdent)
			if err != nil {
				return err
			}
			prev := new(Expression)
			*prev = *left
			*left = Expression{}
			left.Suffix = &Suffix{
				Operator: OperatorFIELD,
				Operands: []Operand{
					{Expression: prev},
					{Reference: &Reference{Identifier: Identifier(fieldTok.Text)}},
				},
			}
			continue
		}

		op := p.PeekOperator()
		if op == OperatorNone || op == OperatorNOT || op.Priority() < minBp {
			break
		}
		p.ReadOperator()

		prev := new(Expression)
		*prev = *left

		var right Expression
		if err := right.ParseExpr(p, op.Priority()); err != nil {
			return err
		}
		rightCopy := new(Expression)
		*rightCopy = right

		*left = Expression{}
		left.Infix = &Infix{
			Operator: op,
			Operands: [2]Operand{
				{Expression: prev},
				{Expression: rightCopy},
			},
		}
	}

	return nil
}

func (s *If) Parse(parser *Parser) error {
	if _, err := parser.Accept(KeywordIf); err != nil {
		return err
	}
	if err := s.Condition.Parse(parser); err != nil {
		return err
	}
	if _, err := parser.Accept(KeywordThen); err != nil {
		return err
	}
	if err := s.Then.Parse(parser); err != nil {
		return err
	}
	parser.Skip(TokenKind(';'))
	if parser.Skip(KeywordElse) != nil {
		s.Else = &Statement{}
		if err := s.Else.Parse(parser); err != nil {
			return err
		}
	}
	return nil
}

func (g *Group) Parse(parser *Parser) error {
	tok := parser.Peek()
	switch tok.TokenKind {
	case KeywordWhile:
		parser.Next()
		g.While = &While{}
		if err := g.While.Expression.Parse(parser); err != nil {
			return err
		}
		parser.Skip(KeywordDo)
	case KeywordFor:
		parser.Next()
		g.For = &For{}
		if err := g.For.Parse(parser); err != nil {
			return err
		}
		parser.Skip(KeywordDo)
	case KeywordCase:
		parser.Next()
		g.Case = &Case{}
		if err := g.Case.Expression.Parse(parser); err != nil {
			return err
		}
		parser.Skip(KeywordOf)
		parser.Skip(KeywordDo)
	default:
		if _, err := parser.Accept(KeywordDo); err != nil {
			return err
		}
	}
	for !parser.End() && parser.Peek().TokenKind != KeywordEnd {
		var s Statement
		if err := s.Parse(parser); err != nil {
			return err
		}
		g.Statements = append(g.Statements, s)
		parser.Skip(TokenKind(';'))
	}
	if _, err := parser.Accept(KeywordEnd); err != nil {
		return err
	}
	parser.Skip(TokenIdent) // optional label after END
	return nil
}

func (f *For) Parse(parser *Parser) error {
	if err := f.Reference.Parse(parser); err != nil {
		return err
	}
	if _, err := parser.Accept(TokenKind('=')); err != nil {
		return err
	}
	if err := f.Start.Parse(parser); err != nil {
		return err
	}
	if _, err := parser.Accept(KeywordTo); err != nil {
		return err
	}
	if err := f.To.Parse(parser); err != nil {
		return err
	}
	if parser.Skip(KeywordBy) != nil {
		f.By = &Expression{}
		if err := f.By.Parse(parser); err != nil {
			return err
		}
	}
	return nil
}

func (e *Expression) Parse(p *Parser) error {
	err := e.ParseExpr(p, 0)
	if err != nil {
		return err
	}
	return nil
}

/*


<PROGRAM> ::= <STATEMENT LIST>
<STATEMENT LIST> := <STATEMENT> | <STATEMENT LIST> <STATEMENT>
<STATEMENT> := <BASIC STATEMENT> | <IF STATEMENT>

<BASIC STATEMENT> := <ASSIGNMENT> ;
	| <GROUP> ;
	| <PROCEDURE DEFINITION> ;
	| <RETURN STATEMENT> ;
	| <CALL STATEMENT> ;
	| <GO TO STATEMENT> ;
	| <DECLARATION STATEMENT> ;
	| HALT i ;
	| ;
	| <LABEL DEFINITION> <BASIC STATEMENT>
s


<IF STATEMENT> := <IF CLAUSE> <STATEMENT>
	| <IF CLAUSE> <ELSE PART> <STATEMENT>
	| <LABEL DEFINITION> <BASIC STATEMENT>

<IF CLAUSE> := "IF" <EXPRESSION> "THEN"
<ELSE PART> := <BASIC STATEMENT> "ELSE"
<GROUP> := <GROUP HEAD> <ENDING>

<GROUP HEAD> := "DO"
	| "DO <STEP DEFINITION>
	| "DO" <WHILE CLAUSE> ;
	| "DO" <CASE SELECTOR> ;
	| <GROUP HEAD> <STATEM//ENT>

<STEP DEFINITION> := <VARIABLE> <REPLACE> <EXPRESSION> <ITERATION CONTROL>

<ITERATION CONTROL> := "TO" <EXPRESSION>
					| "TO" <EXPRESSION> "BY" <EXPRESSION>
<WHILE CLAUSE> := "WHILE" <EXPRESSION>
<CASE SELECTOR> :=  "CASE" <EXPRESSION>

<PROCEDURE DEFINITION> := < PROCEDURE HEAD> <STATEMENT LIST> <ENDING>
<PROCEDURE HEAD> ::= <PROCEDURE NAME> ;
	| <PROCEDURE NAME>
	| <PROCEDURE NAME> <TYPE>
	| <PROCEDURE NAME> <PARAMETER LIST>
	| <PROCEDURE NAME> <PARAMETER LIST> <TYPE>
	| <PROCEDURE NAME> <LABEL DEFINITION)

<PARAMETER LIST> := <PARAMETER HEAD> <IDENTIFIER>
<PARAMET ER HEAD) : :T (
<PARAMETER HEAD) < 10 ENT I FIE R) ,
<ENDING) END
END <IDENTIFIER)
<LABEL DEFINITION> <ENDING>
<LABEL DEFINITION)
<RETURN STATE~ENT>
<CALL STATEMENT)
<GO TO STATEMENT>
<GO TO) GO TO
GOTD
"T
< ID E NT I FIE R) :
<NUMBER) :
RETURN
RETURN <EXPRESSION>
CALL <VARIABLE>
<GO TO) <IDENTIFIER>
<GO TO) <NUMBER>
LIST>
;
<TYPE)
<DECLARATION STATEMENT> DECLARE <DECLARATION ELEMENT)
<ENDING>
<DECLARATION STATEMENT> , <CECLARATION ELEMENT>
<DECLARATION ELEMENT) ::= <TYPE DECLARATION> "
<IDENTIFIER> LIT2RALLY <STRING>
<IDENTIFIER> <DATA LIST>.
<DATA LIST> ::= <DATA HEAD> <CONSTANT> )
<DATA HEAD> DATA (
", <DATA HEAD> <CONSTANT> ,
<TYPE DECLARATION> ::=, <IDENTIFIER SPECIFICATION> <TYPE>
<BOUND HEAD> <NUMBER> ) <TYPE>
<TYPE DECLARATION> <INITIAL LIST>
49
64
65
66
67
68
69
70
71
72
73
74
75
76
71
78
79
<TVPE> :: =
\ BYTE
ADDRESS
LABEl
<BOUND HEAD> .. -.. - <IDENTIFIER SPECIFICATION> (
<IDENTIFIER SPECIFICATION> ::= <VARIABLE NAME>
I <IDENTIFIER LIST> <VARIAELE NAME> )
<IDENTIFIER LIST> .. -•• T (
<VARIABLE NAME>
<BASED VARIABLE>
<INITIAL LIST>
< INITIAL HEAD>
<A SSI GNf.1ENT>
: : T
.. -o .-
• 0_
• 0-
<IDENTIFIER LIST> <VARIABLE NAME> t
<IDENTIFIER>
<BASED VARIABLE> <IDENTIFIER>
<IDENTIFIER> BASED
<INITIAL HEAD> <CONSTANT>
I t\ITIAL (
<INITIAL HEAD> <CONSTANT> t
<VARIABLE> <REPLACE> <EXPRESSION>
<LEFT PART> <ASSIGNMENT>
80 <REPLACE> ::= =
81 <LEFT PART> ::= <VARIABLE>,
82 <EXPRESSION> <LOGICAL EXPRESSION>
83 •• , <VARIABLE>: = <LOGICAL EXPRESSION>
84 <LOGICAL EXPRESSION> •• - <LOGICAL FACTOR>
85 • ·-1 <LOGICAL EXPRESSION> OR <LOGICAL FACTOR>
86 <LOGICAL EXPRESSION> XOR <LOGICAL FACTOR>
<LOGICAL FACTOR> : : j <LOGICAL SECONDARY>
<LOGICAL FACTOR> AND <LOGICAL SECONDARY>
<LOGICAL SECONCARY> <LOGICAL PRIMARY>
NOT <LOGICAL PRIMARY>
<LOGICAL PRIMARY> <ARITHMETIC EXPRESSION>
<RELATION>
<ARITHMETIC
<TERM> .0_.. -
I<PRIMARY>
"I =<
>
< >
< => =
EXPRESSION>
<PRIMARY>
<ARI THME TI C EXPRE SS ION> <RELAT! ON> <ARITHMETIC EXPRESS ION)
:: = <T ERM>
<ARITHMETIC EXPRESSION> + <TERM>
<ARITHMETIC EXPRESSION> - <TERM)
<ARITHMETIC EXPRESSION> PLUS <TERM>
<ARITHMETIC EXPRESSION> MINUS <TERM>
- <TERM>
<TERM> * <PRIMARY>
<TERM> I <PRIMARY>
<TER~> MOD <PRIMARY>
00= <CONSTANT>
o 0.\ . <CONS TANT> •
~~~~n~~J>HEAD> <CONSl ANT> )
• <VARIABLE>
( <EXPRESSION> )
<COr--STANT HEAD> . (
<VARI ABL E> o 0,
<SUBSCRI PT HE AC>
<COt\ST ANT> : : =
I
<CONSTANT HEAD> <CONSTANT> t
< IOENT I FI ER>
<SUBSCRIPT HEAD> <EXPRESSION> )
00_ <IDENT IFIER> (
00, <SUBSCRI PT HEAD> <E:XPRESS ION> ,
<STRING>
<NUMBER>
123 <TO> ::. TO
124 <BY> ::= BY
125 <WHILE> ::= WHILE

*/
