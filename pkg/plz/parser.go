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
	tok, err = parser.Accept(TokenInt, TokenChar)
	if err != nil {
		return err
	}
	g.Value = byte(tok.Number & 0xff)
	return nil
}

/*


<PROGRAM> ::= <STATEMENT LIST>
<STATMENT LIST> := <STATEMENT> | <STATEMENT LIST> <STATEMENT>
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
