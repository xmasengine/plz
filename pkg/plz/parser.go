package plz

import (
	"fmt"
	"path/filepath"
)

// Error represents a parse error with a source position and a descriptive
// message. It implements the error interface.
type Error struct {
	Position
	Message string
}

// Error returns the formatted error string in the form "filename:line:col: message".
func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Position, e.Message)
}

// Errorf creates an Error from the receiver Token's position and a formatted
// message. It is a convenience method for reporting parse errors at a token's
// source location.
func (t Token) Errorf(form string, args ...any) Error {
	return Error{Position: t.Position, Message: fmt.Sprintf(form, args...)}
}

// Parser for PL/Z.
//
// The design is a peek/parse parser with unlimited lookahead. Each AST node
// parses itself using the parser and a Parse method. For child nodes each AST
// node has to peek which child node it is. It is an error to call Parse on an
// AST node that is not currently available in the parser.
type Parser struct {
	Tokens      []Token
	Current     int
	TypeAliases map[string]Type
}

// NewParser creates a new Parser that reads from the given token slice. It
// initialises the TypeAliases map with the built-in type aliases (such as
// TEXT).
func NewParser(tokens []Token) *Parser {
	return &Parser{
		Tokens:      tokens,
		TypeAliases: builtinTypeAliases(),
	}
}

// builtinTypeAliases returns the map of pre-defined type aliases. Currently
// only "TEXT" is provided, which is a record with a length byte and a
// byte-array text field.
func builtinTypeAliases() map[string]Type {
	return map[string]Type{
		"TEXT": {
			Typ: &Record{
				Fields: []Field{
					{
						Identifier: "length",
						Type:       Type{Typ: &PredeclaredType{Kind: PredeclaredByte}},
					},
					{
						Identifier: "text",
						Type: Type{
							Typ: &Array{
								ElemType: Type{Typ: &PredeclaredType{Kind: PredeclaredByte}},
							},
						},
					},
				},
			},
		},
	}
}

// Peek returns the current token without consuming it. If the parser has
// reached the end of the token stream, it returns a Token with kind TokenEOF.
func (p Parser) Peek() Token {
	if p.Current >= len(p.Tokens) {
		return Token{TokenKind: TokenEOF}
	}

	return p.Tokens[p.Current]
}

// PeekAt returns the token at the given offset from the current position
// without consuming any tokens. If the offset goes beyond the end of the
// token stream, it returns a Token with kind TokenEOF.
func (p Parser) PeekAt(offset int) Token {
	if p.Current+offset >= len(p.Tokens) {
		return Token{TokenKind: TokenEOF}
	}
	return p.Tokens[p.Current+offset]
}

// Next consumes and returns the current token, advancing the parser position.
// If the parser has reached the end, it returns a Token with kind TokenEOF.
func (p *Parser) Next() Token {
	if p.Current >= len(p.Tokens) {
		return Token{TokenKind: TokenEOF}
	}

	res := p.Peek()
	p.Current++
	return res
}

// End reports whether the parser has consumed all tokens.
func (p *Parser) End() bool {
	return p.Current >= len(p.Tokens)
}

// Accept consumes the next token and returns it if its kind matches any of
// the given kinds. If none match, it returns an error describing the
// unexpected token.
func (p *Parser) Accept(kinds ...TokenKind) (*Token, error) {
	token := p.Next()
	for _, kind := range kinds {
		if kind == token.TokenKind {
			return &token, nil
		}
	}
	return nil, token.Errorf("Accept: unexpected token, not in %v", kinds)
}

// Have reports whether any of the given token kinds appear at the current
// position (or at subsequent offsets). It does not consume any tokens.
// Have reports whether the next len(kinds) tokens match the given kinds in
// sequence. All kinds must match at successive positions.
func (p Parser) Have(kinds ...TokenKind) bool {
	for offset, kind := range kinds {
		token := p.PeekAt(offset)
		if kind != token.TokenKind {
			return false
		}
	}
	return len(kinds) > 0
}

// Skip consumes and returns the next token only if its kind matches the given
// kind. Otherwise it returns nil without consuming anything.
func (p *Parser) Skip(kind TokenKind) *Token {
	t := p.Peek()
	if t.TokenKind == kind {
		p.Next()
		return &t
	}
	return nil
}

// ParseFile scans, parses and returns a PL/Z source file as a Program AST.
// The name parameter specifies the file path to read.
func ParseFile(name string) (*Program, error) {
	tokens, err := ScanFile(name)
	if err != nil {
		return nil, err
	}
	var res Program
	parser := NewParser(tokens)
	err = res.Parse(parser)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// Parse parses a complete PL/Z program from the token stream. It handles
// INCLUDE directives by recursively parsing the referenced file and
// appending its statements. All other top-level statements are parsed via
// Statement.Parse.
func (p *Program) Parse(parser *Parser) error {
	for !parser.End() {
		// Handle INCLUDE "filename"
		if parser.Peek().TokenKind == KeywordInclude {
			includeTok := parser.Next() // consume INCLUDE
			filenameTok, err := parser.Accept(TokenString)
			if err != nil {
				return err
			}
			parser.Skip(TokenKind(';')) // skip optional semicolon

			// Resolve path relative to the including file's directory.
			incPath := filenameTok.Text
			if dir := filepath.Dir(includeTok.Position.Filename); dir != "." && dir != "" {
				if !filepath.IsAbs(incPath) {
					incPath = filepath.Join(dir, incPath)
				}
			}

			incTokens, err := ScanFile(incPath)
			if err != nil {
				return filenameTok.Errorf("include: %v", err)
			}
			incParser := &Parser{
				Tokens:      incTokens,
				TypeAliases: parser.TypeAliases,
			}
			incProg := Program{}
			if err := incProg.Parse(incParser); err != nil {
				return err
			}
			p.Statements = append(p.Statements, incProg.Statements...)
			continue
		}

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

// Parse parses a single statement from the token stream. It first attempts
// to parse an optional named label (e.g. "loop:"), then dispatches on the
// leading keyword to parse the appropriate command type.
func (s *Statement) Parse(parser *Parser) error {
	var l Label
	if l.Have(*parser) {
		err := l.Parse(parser)
		if err != nil {
			return err
		}
		s.Label = &l
	}
	tok := parser.Peek()
	var err error

	switch tok.TokenKind {
	case KeywordCall:
		cmd := Call{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordConstant:
		cmd := Constant{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordData:
		if s.Label == nil {
			return tok.Errorf("No label for DATA")
		}
		cmd := Data{Name: s.Label.Name}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordDeclare:
		cmd := Declare{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordDefine:
		cmd := Define{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordDisable:
		cmd := Disable{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordEnable:
		cmd := Enable{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordGoTo:
		cmd := GoTo{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordHalt:
		cmd := Halt{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordIf:
		cmd := If{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordDo, KeywordWhile, KeywordFor, KeywordCase:
		cmd := Group{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordLet:
		cmd := Let{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordOutput:
		cmd := Output{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordInterrupt, KeywordNMI:
		if parser.Have(KeywordInterrupt, KeywordProc) || parser.Have(KeywordNMI, KeywordProc) {
			cmd := Procedure{}
			err = cmd.Parse(parser)
			s.Command = cmd
		} else {
			cmd := InterruptStmt{}
			err = cmd.Parse(parser)
			s.Command = cmd
		}
	case KeywordProc:
		cmd := Procedure{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordReturn:
		cmd := Return{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordTask:
		cmd := Task{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordSuspend:
		cmd := Suspend{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordResume:
		cmd := Resume{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordSleep:
		cmd := Sleep{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordYield:
		cmd := Yield{}
		err = cmd.Parse(parser)
		s.Command = cmd
	case KeywordAt:
		cmd := At{}
		err = cmd.Parse(parser)
		s.Command = cmd
	default:
		return tok.Errorf("Statement: unexpected token %v", tok)
	}

	return err
}

// Have reports whether the parser's current position contains a named label
// (an identifier followed by a colon). It peeks at tokens but does not
// consume them.
func (l *Label) Have(parser Parser) bool {
	return parser.Have(TokenIdent, ':')
}

// Parse parses a named label from the token stream. An identifier (e.g.
// "loop") followed by a colon is consumed, and the label's Name field is
// set to the identifier text.
func (l *Label) Parse(parser *Parser) error {
	tok, err := parser.Accept(TokenIdent)
	if err != nil {
		return err
	}
	l.Name = tok.Text
	_, err = parser.Accept(TokenKind(':'))
	return err
}

// Parse parses a HALT statement. It consumes the HALT keyword and returns.
func (g *Halt) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordHalt)
	return err
}

// Parse parses an INTERRUPT or NMI install statement. The keyword has already
// been consumed by Statement.Parse (the caller decides which keyword it is).
// This method consumes the procedure name identifier that follows.
func (s *InterruptStmt) Parse(parser *Parser) error {
	tok := parser.Next()
	if tok.TokenKind == KeywordNMI {
		s.NMI = true
	}
	ident, err := parser.Accept(TokenIdent)
	if err != nil {
		return err
	}
	s.Target = Identifier(ident.Text)
	return nil
}

// Parse parses an AT directive. The syntax is AT literal, where literal
// is a numeric value or a named constant. The AT directive sets the
// absolute memory address for subsequent data declarations.
func (a *At) Parse(parser *Parser) error {
	if _, err := parser.Accept(KeywordAt); err != nil {
		return err
	}
	return a.Address.Parse(parser)
}

// Parse parses a CONSTANT declaration. The syntax is:
//
//	CONSTANT name [=] literal
//
// The equals sign is optional, and the literal value is also optional.
func (g *Constant) Parse(parser *Parser) error {
	if _, err := parser.Accept(KeywordConstant); err != nil {
		return nil
	}

	name, err := parser.Accept(TokenIdent)
	if err != nil {
		return nil
	}
	g.Name = name.Text

	parser.Skip(TokenKind('=')) // Skip optional =

	// Optional literal value.
	if err := g.Literal.Parse(parser); err != nil {
		return err
	}
	return nil
}

// Parse parses a DATA statement. The syntax is:
//
//	name: DATA literal [, literal ...]
//
// Each literal may be a number, a string, or an identifier reference.
func (g *Data) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordData)
	if err != nil {
		return nil
	}
	for {
		var lit Literal
		if err := lit.Parse(parser); err != nil {
			return err
		}
		if lit.Lit == nil {
			break
		}
		g.Literals = append(g.Literals, lit)
		if parser.Peek().TokenKind != ',' {
			break
		}
		parser.Next() // consume ','
	}
	return nil
}

// Parse parses a literal value from the token stream. A literal is one of:
//   - an integer (TokenInt), stored as a NumberLit
//   - a string token (TokenString), stored as a TextLit
//   - an identifier (TokenIdent), stored as a ReferenceLit
func (g *Literal) Parse(parser *Parser) error {
	tok, err := parser.Accept(TokenInt, TokenString, TokenIdent)
	if err != nil {
		return err
	}

	if tok.TokenKind == TokenInt {
		g.Lit = &NumberLit{Value: tok.Number}
	} else if tok.TokenKind == TokenString {
		g.Lit = &TextLit{Value: tok.Text}
	} else if tok.TokenKind == TokenIdent {
		g.Lit = &ReferenceLit{Value: &Reference{Identifier: Identifier(tok.Text)}}
	}
	return nil
}

// Parse parses an identifier token from the stream and stores it in the
// receiver. The identifier text is captured as an Identifier value.
func (i *Identifier) Parse(parser *Parser) error {
	tok, err := parser.Accept(TokenIdent)
	if err != nil {
		return err
	}
	*i = Identifier(tok.Text)
	return nil
}

// Parse parses a reference from the token stream. A reference is an identifier
// optionally followed by array subscripts (e.g. "[index]") and/or field
// accesses (e.g. ".field"). Subscripts and fields are parsed greedily in a
// loop until neither is found.
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

// Parse parses a DISABLE statement. It consumes the DISABLE keyword and returns.
func (g *Disable) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordDisable)
	return err
}

// Parse parses an ENABLE statement. It consumes the ENABLE keyword and returns.
func (g *Enable) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordEnable)
	return err
}

// Parse parses a GOTO statement. The syntax is:
//
//	GOTO label
//
// where label is a named label (identifier).
func (g *GoTo) Parse(parser *Parser) error {
	if _, err := parser.Accept(KeywordGoTo); err != nil {
		return err
	}
	tok, err := parser.Accept(TokenIdent)
	if err != nil {
		return err
	}
	g.Name = tok.Text
	return nil
}

// Parse parses a CALL statement. The syntax is:
//
//	CALL name [(arg [, arg ...])]
//
// The argument list is optional. Each argument is parsed as an Expression.
func (g *Call) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordCall)
	if err != nil {
		return err
	}
	tok, err := parser.Accept(TokenIdent)
	if err != nil {
		return err
	}
	g.Reference = Reference{Identifier: Identifier(tok.Text)}

	// Optional argument list: (expr1, expr2, ...)
	if parser.Peek().TokenKind == '(' {
		parser.Next()
		for parser.Peek().TokenKind != ')' && !parser.End() {
			var arg Expression
			if err := arg.Parse(parser); err != nil {
				return err
			}
			g.Arguments = append(g.Arguments, arg)
			if parser.Peek().TokenKind != ',' {
				break
			}
			parser.Next()
		}
		if _, err := parser.Accept(TokenKind(')')); err != nil {
			return err
		}
	}
	return nil
}

// Parse parses a RETURN statement. The syntax is:
//
//	RETURN [expr [, expr ...]]
//
// The return expressions are optional and comma-separated. Parsing stops at
// EOF, semicolon, or the END keyword.
func (g *Return) Parse(parser *Parser) error {
	if _, err := parser.Accept(KeywordReturn); err != nil {
		return err
	}
	// Optional return expressions separated by commas.
	for {
		tok := parser.Peek()
		if tok.TokenKind == TokenEOF || tok.TokenKind == ';' || tok.TokenKind == KeywordEnd {
			break
		}
		// Check if next token looks like the start of an expression.
		switch tok.TokenKind {
		case TokenInt, TokenString, TokenChar, TokenIdent, KeywordInput, '(', '+', '-', '!':
			var expr Expression
			if err := expr.Parse(parser); err != nil {
				return err
			}
			g.Expressions = append(g.Expressions, expr)
		default:
			return tok.Errorf("Return: unexpected token %v", tok)
		}
		if parser.Peek().TokenKind != ',' {
			break
		}
		parser.Next()
	}
	return nil
}

// Parse parses an OUTPUT statement. The syntax is:
//
//	OUTPUT port expr
//
// where port is a numeric literal and expr is the value to write to the port.
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

// Parse parses a type expression from the token stream. It handles:
//   - RECORD fields END (inline record definition)
//   - TYPE name (type alias reference)
//   - ARRAY [size] elemtype (array type with optional size)
//   - BYTE, WORD, or DATA (predeclared types)
func (t *Type) Parse(parser *Parser) error {
	if parser.Peek().TokenKind == KeywordRecord {
		parser.Next()
		rec := &Record{}
		for parser.Peek().TokenKind == TokenIdent {
			var f Field
			if err := f.Identifier.Parse(parser); err != nil {
				return err
			}
			if err := f.Type.Parse(parser); err != nil {
				return err
			}
			rec.Fields = append(rec.Fields, f)
			if parser.Peek().TokenKind != TokenKind(',') {
				break
			}
			parser.Next()
		}
		if _, err := parser.Accept(KeywordEnd); err != nil {
			return err
		}
		t.Typ = rec
		return nil
	}
	if parser.Peek().TokenKind == KeywordType {
		parser.Next()
		nameTok, err := parser.Accept(TokenIdent)
		if err != nil {
			return err
		}
		aliased, ok := parser.TypeAliases[nameTok.Text]
		if !ok {
			return fmt.Errorf("undefined type alias %q", nameTok.Text)
		}
		*t = aliased
		return nil
	}
	if parser.Peek().TokenKind == KeywordArray {
		parser.Next()
		arr := &Array{}
		if parser.Peek().TokenKind == '[' {
			parser.Next()
			tok := parser.Peek()
			if tok.TokenKind == TokenInt {
				parser.Next()
				arr.Size = tok.Number
			}
			if _, err := parser.Accept(TokenKind(']')); err != nil {
				return err
			}
		}
		if err := arr.ElemType.Parse(parser); err != nil {
			return err
		}
		t.Typ = arr
		return nil
	}
	tok, err := parser.Accept(KeywordByte, KeywordWord, KeywordData)
	if err != nil {
		return err
	}
	switch tok.TokenKind {
	case KeywordByte:
		t.Typ = &PredeclaredType{Kind: PredeclaredByte}
	case KeywordWord:
		t.Typ = &PredeclaredType{Kind: PredeclaredWord}
	case KeywordData:
		t.Typ = &PredeclaredType{Kind: PredeclaredData}
	}
	return nil
}

// Parse parses a DEFINE statement. The syntax is:
//
//	DEFINE name type
//
// It creates a type alias, registering it in the parser's TypeAliases map so
// that subsequent code can refer to the type by name using the TYPE keyword.
func (d *Define) Parse(parser *Parser) error {
	if _, err := parser.Accept(KeywordDefine); err != nil {
		return err
	}
	nameTok, err := parser.Accept(TokenIdent)
	if err != nil {
		return err
	}
	d.Name = Identifier(nameTok.Text)
	if err := d.Type.Parse(parser); err != nil {
		return err
	}
	parser.TypeAliases[nameTok.Text] = d.Type
	return nil
}

// Parse parses a DECLARE statement. The syntax is:
//
//	DECLARE name type
//	DECLARE name ARRAY [OF] [size] type
//	DECLARE name type [= literal]
//	DECLARE name type AT literal
//
// For array declarations, the optional size may be given in brackets.
// If present, an optional initializer follows an equals sign. The AT
// suffix places the variable at an absolute address instead; no
// initializer is allowed in that case.
func (g *Declare) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordDeclare)
	if err != nil {
		return err
	}
	err = g.Identifier.Parse(parser)
	if err != nil {
		return err
	}

	// Optional ARRAY [size] OF type (single dimension only).
	if parser.Skip(KeywordArray) != nil {
		parser.Skip(KeywordOf)
		if parser.Peek().TokenKind == '[' {
			parser.Next()
			tok := parser.Peek()
			if tok.TokenKind == TokenInt {
				parser.Next()
				g.Size = tok.Number
			} else {
				g.Size = 0 // unbounded
			}
			if _, err := parser.Accept(TokenKind(']')); err != nil {
				return err
			}
		} else {
			g.Size = 0 // unbounded
		}
		return g.Type.Parse(parser)
	}

	err = g.Type.Parse(parser)
	if err != nil {
		return err
	}

	// Optional AT suffix for absolute address (no initializer allowed).
	if parser.Skip(KeywordAt) != nil {
		g.At = &Literal{}
		return g.At.Parse(parser)
	}

	if parser.Skip(TokenKind('=')) != nil {
		g.Initializer = &Initializer{}
		if err := g.Initializer.Literal.Parse(parser); err != nil {
			return err
		}
	}
	return nil
}

// Parse parses a LET assignment statement. The syntax is:
//
//	LET reference = expression
//
// The reference is parsed first (supporting subscripts and field access),
// followed by the equals sign and the value expression.
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

// PeekOperator looks at the next token(s) without consuming them and returns
// the corresponding Operator value. If the next token(s) do not form a
// recognised operator, it returns OperatorNone.
//
// Multi-character operators ('<=', '>=', '<>', '!=', '<<', '>>') are detected
// by inspecting the current and subsequent token kinds.
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
		if p.PeekAt(1).TokenKind == '<' {
			return OperatorShiftLeft
		}
		return OperatorLT
	case '>':
		if p.PeekAt(1).TokenKind == '=' {
			return OperatorGTE
		}
		if p.PeekAt(1).TokenKind == '>' {
			return OperatorShiftRight
		}
		return OperatorGT
	}
	return OperatorNone
}

// ReadOperator consumes tokens and returns the operator found. Multi-character
// operators (e.g. "==", "!=", "<=", ">=", "<<", ">>") consume two tokens.
// If no operator is present, it returns OperatorNone and consumes no tokens.
func (p *Parser) ReadOperator() (op Operator) {
	op = p.PeekOperator()
	switch op {
	case OperatorEQU, OperatorNEQ, OperatorGTE, OperatorLTE, OperatorShiftLeft, OperatorShiftRight:
		p.Next()
		p.Next()
	case OperatorNone:
		return OperatorNone
	default:
		p.Next()
	}
	return op
}

// ParseExpr is the core Pratt parser for expressions. It reads tokens
// starting from the current position and builds an Expression tree.
//
// The minBp (minimum binding power) parameter sets the minimum precedence
// that subsequent operators must have; operators with a lower priority than
// minBp cause the loop to stop and the current expression to be returned.
//
// The parse proceeds in two phases:
//
//  1. Prefix (nud) phase: dispatches on the current token to parse atomic
//     expressions (literals, identifiers, parenthesised sub-expressions) or
//     prefix operators (unary '-', unary '!', unary '+' as a no-op).
//
//  2. Infix/suffix (led) phase: loops while the next token is a postfix
//     operator ('[' for array indexing, '(' for function calls, '.' for field
//     access) or an infix operator. Each iteration wraps the current left-hand
//     expression into a larger expression node using the operator's precedence
//     to recursively parse the right-hand side.
func (left *Expression) ParseExpr(p *Parser, minBp int) error {
	tok := p.Peek()

	switch tok.TokenKind {
	case TokenInt:
		p.Next()
		left.Expr = &Operand{Op: &Literal{Lit: &NumberLit{Value: tok.Number}}}

	case TokenString:
		p.Next()
		left.Expr = &Operand{Op: &Literal{Lit: &TextLit{Value: tok.Text}}}

	case TokenChar:
		p.Next()
		left.Expr = &Operand{Op: &Literal{Lit: &NumberLit{Value: tok.Number}}}

	case TokenIdent:
		p.Next()
		left.Expr = &Operand{Op: &Reference{Identifier: Identifier(tok.Text)}}

	case KeywordInput:
		p.Next()
		if _, err := p.Accept(TokenKind('(')); err != nil {
			return err
		}
		var port Expression
		if err := port.Parse(p); err != nil {
			return err
		}
		if _, err := p.Accept(TokenKind(')')); err != nil {
			return err
		}
		left.Expr = &Operand{Op: &Input{Port: port}}

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
		left.Expr = &Prefix{
			Operator: OperatorNEG,
			Operand:  Operand{Op: &right},
		}

	case '!':
		p.Next()
		var right Expression
		err := right.ParseExpr(p, OperatorNOT.Priority())
		if err != nil {
			return err
		}
		left.Expr = &Prefix{
			Operator: OperatorNOT,
			Operand:  Operand{Op: &right},
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
				return tok.Errorf("empty array subscript")
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
			left.Expr = &Suffix{
				Operator: OperatorINDEX,
				Operands: []Operand{
					{Op: prev},
					{Op: indexCopy},
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
			operands = append(operands, Operand{Op: prev})
			for i := range args {
				ac := new(Expression)
				*ac = args[i]
				operands = append(operands, Operand{Op: ac})
			}
			*left = Expression{}
			left.Expr = &Suffix{
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
			left.Expr = &Suffix{
				Operator: OperatorFIELD,
				Operands: []Operand{
					{Op: prev},
					{Op: &Reference{Identifier: Identifier(fieldTok.Text)}},
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
		left.Expr = &Infix{
			Operator: op,
			Operands: [2]Operand{
				{Op: prev},
				{Op: rightCopy},
			},
		}
	}

	return nil
}

// Parse parses an IF statement. The syntax is:
//
//	IF condition THEN statement [ELSE statement]
//
// The ELSE branch is optional. The condition is parsed as an Expression and
// both branches are parsed as Statement values.
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

// Parse parses a grouped statement block. It handles four forms identified by
// the leading keyword:
//   - WHILE condition [DO] statements END
//   - FOR var = start TO end [BY step] [DO] statements END
//   - CASE expr [OF] [DO] statements END
//   - DO statements END (plain block)
//
// In each case, statements inside the block are parsed until END.
func (g *Group) Parse(parser *Parser) error {
	tok := parser.Peek()
	isCase := false
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
		parser.Skip(KeywordDo) // optional DO

		// Parse case branches. Each branch is introduced by OF followed
		// by a numeric value (or DEFAULT) and the statement to execute
		// when the selector matches.
		isCase = true
		for !parser.End() && parser.Peek().TokenKind != KeywordEnd {
			if _, err := parser.Accept(KeywordOf); err != nil {
				return parser.Peek().Errorf("expected OF or END in CASE")
			}
			if parser.Peek().TokenKind == KeywordDefault {
				parser.Next()
				var s Statement
				if err := s.Parse(parser); err != nil {
					return err
				}
				g.Case.Default = &s
			} else {
				var branch CaseBranch
				// Accept either a numeric literal or a constant identifier.
				tok, err := parser.Accept(TokenInt, TokenIdent)
				if err != nil {
					return tok.Errorf("expected case value or DEFAULT")
				}
				if tok.TokenKind == TokenInt {
					branch.Values = append(branch.Values, CaseVal{Value: tok.Number})
				} else {
					branch.Values = append(branch.Values, CaseVal{Name: tok.Text})
				}
				if err := branch.Statement.Parse(parser); err != nil {
					return err
				}
				g.Case.Branches = append(g.Case.Branches, branch)
			}
			parser.Skip(TokenKind(';'))
		}

	default:
		if _, err := parser.Accept(KeywordDo); err != nil {
			return err
		}
	}
	if !isCase {
		for !parser.End() && parser.Peek().TokenKind != KeywordEnd {
			var s Statement
			if err := s.Parse(parser); err != nil {
				return err
			}
			g.Statements = append(g.Statements, s)
			parser.Skip(TokenKind(';'))
		}
	}
	if _, err := parser.Accept(KeywordEnd); err != nil {
		return err
	}
	return nil
}

// Parse parses a PROCEDURE declaration. The syntax is:
//
//	[INTERRUPT | NMI] PROCEDURE name [(params)] [type] [REENTRANT]
//	  statements
//	END
//
// Parameters are parsed as a comma-separated list of identifier-type pairs.
// An optional return type (BYTE, WORD, or RECORD) follows the parameter list.
// If REENTRANT is present, the procedure uses stack-based frame allocation.
func (p *Procedure) Parse(parser *Parser) error {
	// Optional INTERRUPT or NMI modifier.
	if parser.Skip(KeywordInterrupt) != nil {
		p.Interrupt = &Interrupt{Interrupt: 1}
	} else if parser.Skip(KeywordNMI) != nil {
		p.Interrupt = &Interrupt{NMI: true}
	}
	if _, err := parser.Accept(KeywordProc); err != nil {
		return err
	}
	nameTok, err := parser.Accept(TokenIdent)
	if err != nil {
		return err
	}
	p.Name = Label{Name: nameTok.Text}

	// Optional typed parameter list: (param1 TYPE, param2 TYPE, ...)
	if parser.Peek().TokenKind == '(' {
		parser.Next()
		for parser.Peek().TokenKind != ')' && !parser.End() {
			var id Identifier
			if err := id.Parse(parser); err != nil {
				return err
			}
			var typ Type
			if err := typ.Parse(parser); err != nil {
				return err
			}
			if typ.Predeclared() == PredeclaredNone && typ.Record() == nil {
				return fmt.Errorf("expected type after parameter name")
			}
			p.Parameters = append(p.Parameters, id)
			p.ParamTypes = append(p.ParamTypes, typ)
			if parser.Peek().TokenKind != ',' {
				break
			}
			parser.Next()
		}
		if _, err := parser.Accept(TokenKind(')')); err != nil {
			return err
		}
	}

	// Optional return type (BYTE, WORD, or RECORD).
	if parser.Peek().TokenKind == KeywordByte || parser.Peek().TokenKind == KeywordWord || parser.Peek().TokenKind == KeywordRecord {
		if err := p.Type.Parse(parser); err != nil {
			return err
		}
	}

	// Optional REENTRANT.
	if parser.Skip(KeywordReentrant) != nil {
		p.Reentrant = true
	}

	// Parse body statements.
	for !parser.End() && parser.Peek().TokenKind != KeywordEnd {
		var s Statement
		if err := s.Parse(parser); err != nil {
			return err
		}
		p.Statements = append(p.Statements, s)
		parser.Skip(TokenKind(';'))
	}

	if _, err := parser.Accept(KeywordEnd); err != nil {
		return err
	}
	return nil
}

// Parse parses a TASK declaration. The syntax is:
//
//	TASK name [PRIORITY n]
//	  statements
//	END
//
// The priority is optional; if present, it is a numeric value. Body
// statements are parsed until END is reached.
func (t *Task) Parse(parser *Parser) error {
	if _, err := parser.Accept(KeywordTask); err != nil {
		return err
	}
	nameTok, err := parser.Accept(TokenIdent)
	if err != nil {
		return err
	}
	t.Name = Label{Name: nameTok.Text}
	if parser.Skip(KeywordPriority) != nil {
		tok, err := parser.Accept(TokenInt)
		if err != nil {
			return err
		}
		t.Priority = tok.Number
	}
	// Parse body statements.
	for !parser.End() && parser.Peek().TokenKind != KeywordEnd {
		var s Statement
		if err := s.Parse(parser); err != nil {
			return err
		}
		t.Body = append(t.Body, s)
		parser.Skip(TokenKind(';'))
	}
	if _, err := parser.Accept(KeywordEnd); err != nil {
		return err
	}
	return nil
}

// Parse parses a SUSPEND statement. The syntax is:
//
//	SUSPEND task_name
//
// The task name is parsed as an identifier.
func (s *Suspend) Parse(parser *Parser) error {
	if _, err := parser.Accept(KeywordSuspend); err != nil {
		return err
	}
	tok, err := parser.Accept(TokenIdent)
	if err != nil {
		return err
	}
	s.Name = Identifier(tok.Text)
	return nil
}

// Parse parses a RESUME statement. The syntax is:
//
//	RESUME task_name
//
// The task name is parsed as an identifier.
func (r *Resume) Parse(parser *Parser) error {
	if _, err := parser.Accept(KeywordResume); err != nil {
		return err
	}
	tok, err := parser.Accept(TokenIdent)
	if err != nil {
		return err
	}
	r.Name = Identifier(tok.Text)
	return nil
}

// Parse parses a SLEEP statement. The syntax is:
//
//	SLEEP duration
//
// The duration is parsed as an Expression.
func (s *Sleep) Parse(parser *Parser) error {
	if _, err := parser.Accept(KeywordSleep); err != nil {
		return err
	}
	return s.Duration.Parse(parser)
}

// Parse parses a YIELD statement. It consumes the YIELD keyword and returns.
func (y *Yield) Parse(parser *Parser) error {
	_, err := parser.Accept(KeywordYield)
	return err
}

// Parse parses the loop header of a FOR statement. The syntax is:
//
//	var = start TO end [BY step]
//
// The start, end, and optional step are parsed as Expression values.
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

// Parse parses an Expression from the token stream by delegating to
// ParseExpr with a minimum binding power of 0.
func (e *Expression) Parse(p *Parser) error {
	err := e.ParseExpr(p, 0)
	if err != nil {
		return err
	}
	return nil
}
