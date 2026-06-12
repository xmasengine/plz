// Package plz implements a compiler for the PL/Z programming language targeting
// the Z80 CPU. It provides a complete frontend including lexing, parsing,
// semantic analysis, and code generation.
package plz

import (
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"text/scanner"
)

// Position represents a source position in a file.
type Position = scanner.Position

// TokenKind identifies the type of a lexical token.
type TokenKind rune

// Token kinds for special and literal tokens.
const (
	TokenEOF       TokenKind = -(iota + 1) // End of input.
	TokenIdent                             // An identifier.
	TokenInt                               // An integer literal.
	TokenFloat                             // Placeholder: matches text/scanner Float token kind.
	TokenChar                              // A character literal.
	TokenString                            // A string literal.
	TokenRawString                         // A raw string literal (matching Scanner token kinds).
	TokenComment                           // A comment (retained when SkipComments is disabled).
)

// keywordBase is the offset for keyword TokenKind values, chosen to avoid
// collisions with token kind constants (TokenEOF, TokenIdent, etc.) and
// with character runes returned by the scanner (such as '(' and ')').
const keywordBase TokenKind = -200

// Keyword token kinds. Each constant represents a reserved word in the PL/Z language.
const (
	KeywordArray     TokenKind = keywordBase - (iota + 1) // ARRAY
	KeywordByte                                           // BYTE
	KeywordCall                                           // CALL
	KeywordConstant                                       // CONSTANT
	KeywordData                                           // DATA
	KeywordDeclare                                        // DECLARE
	KeywordDisable                                        // DISABLE
	KeywordDo                                             // DO
	KeywordEnable                                         // ENABLE
	KeywordEnd                                            // END
	KeywordHalt                                           // HALT
	KeywordGoTo                                           // GOTO
	KeywordIf                                             // IF
	KeywordThen                                           // THEN
	KeywordElse                                           // ELSE
	KeywordInput                                          // INPUT
	KeywordLength                                         // LENGTH
	KeywordLet                                            // LET
	KeywordReturn                                         // RETURN
	KeywordRecord                                         // RECORD
	KeywordOutput                                         // OUTPUT
	KeywordProc                                           // PROCEDURE
	KeywordReentrant                                      // REENTRANT
	KeywordWord                                           // WORD
	KeywordWhile                                          // WHILE
	KeywordFor                                            // FOR
	KeywordTo                                             // TO
	KeywordBy                                             // BY
	KeywordCase                                           // CASE
	KeywordOf                                             // OF
	KeywordDefine                                         // DEFINE
	KeywordType                                           // TYPE
	KeywordInclude                                        // INCLUDE
	KeywordInterrupt                                      // INTERRUPT
	KeywordNMI                                            // NMI
	KeywordTask                                           // TASK
	KeywordPriority                                       // PRIORITY
	KeywordSuspend                                        // SUSPEND
	KeywordSleep                                          // SLEEP
	KeywordYield                                          // YIELD
	KeywordResume                                         // RESUME
	KeywordAt                                             // AT
	KeywordBank                                           // BANK
	KeywordDefault                                        // DEFAULT
	KeywordTile                                           // TILE
	KeywordSave                                           // SAVE
	KeywordLoad                                           // LOAD
	KeywordPragma                                         // PRAGMA
	KeywordTemplate                                       // TEMPLATE
	KeywordBreak                                          // BREAK
	KeywordContinue                                       // CONTINUE
	KeywordMod                                            // MOD
	KeywordAnd                                            // AND (logical &&)
	KeywordOr                                             // OR (logical ||)
	KeywordNot                                            // NOT (unary !)
	KeywordBitAnd                                         // BITAND (bitwise &)
	KeywordBitOr                                          // BITOR (bitwise |)
	KeywordXor                                            // XOR (bitwise ^)
	KeywordShl                                            // SHL (<<)
	KeywordShr                                            // SHR (>>)
	KeywordPlus                                           // PLUS (+)
	KeywordMinus                                          // MINUS (-)
	KeywordTimes                                          // TIMES (*)
	KeywordDiv                                            // DIV (/)
	KeywordEq                                             // EQ (==)
	KeywordNe                                             // NE (!=)
	KeywordGt                                             // GT (>)
	KeywordLt                                             // LT (<)
	KeywordGe                                             // GE (>=)
	KeywordLe                                             // LE (<=)
)

// Keywords maps keyword strings to their corresponding TokenKind values.
var Keywords = map[string]TokenKind{
	"ARRAY":     KeywordArray,
	"BYTE":      KeywordByte,
	"CALL":      KeywordCall,
	"CONSTANT":  KeywordConstant,
	"DATA":      KeywordData,
	"DECLARE":   KeywordDeclare,
	"DISABLE":   KeywordDisable,
	"DO":        KeywordDo,
	"ENABLE":    KeywordEnable,
	"GOTO":      KeywordGoTo,
	"END":       KeywordEnd,
	"IF":        KeywordIf,
	"THEN":      KeywordThen,
	"ELSE":      KeywordElse,
	"HALT":      KeywordHalt,
	"INPUT":     KeywordInput,
	"LENGTH":    KeywordLength,
	"LET":       KeywordLet,
	"RETURN":    KeywordReturn,
	"OUTPUT":    KeywordOutput,
	"PROCEDURE": KeywordProc,
	"REENTRANT": KeywordReentrant,
	"RECORD":    KeywordRecord,
	"WHILE":     KeywordWhile,
	"FOR":       KeywordFor,
	"TO":        KeywordTo,
	"BY":        KeywordBy,
	"CASE":      KeywordCase,
	"OF":        KeywordOf,
	"DEFINE":    KeywordDefine,
	"TYPE":      KeywordType,
	"WORD":      KeywordWord,
	"INCLUDE":   KeywordInclude,
	"INTERRUPT": KeywordInterrupt,
	"NMI":       KeywordNMI,
	"TASK":      KeywordTask,
	"PRIORITY":  KeywordPriority,
	"SUSPEND":   KeywordSuspend,
	"SLEEP":     KeywordSleep,
	"YIELD":     KeywordYield,
	"RESUME":    KeywordResume,
	"AT":        KeywordAt,
	"BANK":      KeywordBank,
	"DEFAULT":   KeywordDefault,
	"TILE":      KeywordTile,
	"LOAD":      KeywordLoad,
	"SAVE":      KeywordSave,
	"PRAGMA":    KeywordPragma,
	"TEMPLATE":  KeywordTemplate,
	"BREAK":     KeywordBreak,
	"CONTINUE":  KeywordContinue,
	"MOD":       KeywordMod,
	"AND":       KeywordAnd,
	"OR":        KeywordOr,
	"NOT":       KeywordNot,
	"BITAND":    KeywordBitAnd,
	"BITOR":     KeywordBitOr,
	"XOR":       KeywordXor,
	"SHL":       KeywordShl,
	"SHR":       KeywordShr,
	"PLUS":      KeywordPlus,
	"MINUS":     KeywordMinus,
	"TIMES":     KeywordTimes,
	"DIV":       KeywordDiv,
	"EQ":        KeywordEq,
	"NE":        KeywordNe,
	"GT":        KeywordGt,
	"LT":        KeywordLt,
	"GE":        KeywordGe,
	"LE":        KeywordLe,
}

// String returns the human-readable name for a TokenKind.
func (t TokenKind) String() string {
	for k, v := range Keywords {
		if t == v {
			return k
		}
	}
	return scanner.TokenString(rune(t))
}

// Token represents a single lexical token with its kind, source position, text, and optional numeric value.
type Token struct {
	TokenKind TokenKind // The kind of token.
	Position  Position  // The source position of the token.
	Raw       string    // The original text of the token, may be quoted.
	Text      string    // The text of the token, may be unquoted.
	Number    int       // The numeric value, valid for TokenInt and TokenChar tokens.
}

// String returns a formatted representation of the token including position, kind, and text.
func (t Token) String() string {
	return t.Position.String() + " " + t.TokenKind.String() + " " + t.Text
}

// Scan tokenizes a stream of PL/Z source code read from r and returns the
// resulting token slice. It delegates to scan with an empty filename.
func Scan(rd io.Reader) ([]Token, error) {
	return scan(rd, "")
}

// ScanFile opens the named file, tokenizes its contents, and returns the
// resulting token slice. The filename is recorded in each token's Position.
func ScanFile(name string) ([]Token, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return scan(f, name)
}

// ScanString tokenizes a string of PL/Z source code read from s and returns the
// resulting token slice. It delegates to scan with an empty filename.
func ScanString(s string) ([]Token, error) {
	rd := strings.NewReader(s)
	return scan(rd, "")
}

// scan performs the actual scanning, converting a byte stream into a slice of
// Tokens. It configures the text/scanner to recognize identifiers, integers,
// characters, strings, raw strings, and comments. Identifiers matching a
// keyword string are converted to the corresponding keyword TokenKind. Integer
// and character tokens have their numeric value populated.
func scan(rd io.Reader, name string) ([]Token, error) {
	var err error
	var res []Token
	s := &scanner.Scanner{}
	s.Init(rd)
	s.Position.Filename = name
	s.Mode = scanner.ScanIdents |
		scanner.ScanInts |
		scanner.ScanChars |
		scanner.ScanStrings |
		scanner.ScanRawStrings |
		scanner.ScanComments |
		scanner.SkipComments

	s.Error = func(s2 *scanner.Scanner, msg string) {
		err = errors.New("scan error:" + s2.Pos().String() + ":" + msg)
	}

	for kind := TokenKind(s.Scan()); kind != TokenEOF; kind = TokenKind(s.Scan()) {
		if err != nil {
			return nil, err
		}
		raw := s.TokenText()
		tok := Token{
			TokenKind: kind, Raw: raw, Text: raw, Position: s.Pos(),
		}
		if kind == TokenInt {
			num, err := strconv.ParseUint(tok.Text, 0, 16)
			if err != nil {
				return nil, tok.Errorf("int %s %s", tok.Text, err.Error())
			}
			tok.Number = int(num)
		} else if kind == TokenChar {
			r, _, _, err := strconv.UnquoteChar(tok.Text[1:len(tok.Text)], '\'')
			if err != nil {
				return nil, tok.Errorf("char %s %s", tok.Text, err.Error())
			}
			tok.Number = int(r)
		} else if kind == TokenRawString {
			tok.TokenKind = TokenString
			r, err := strconv.Unquote(tok.Text)
			if err != nil {
				return nil, tok.Errorf("raw string %s %s", tok.Text, err.Error())
			}
			tok.Text = r
		} else if kind == TokenString {
			r, err := strconv.Unquote(tok.Text)
			if err != nil {
				return nil, tok.Errorf("string %s %s", tok.Text, err.Error())
			}
			tok.Text = r
		} else if kind == TokenIdent {
			if kw, ok := Keywords[tok.Text]; ok {
				tok.TokenKind = kw
			}
		}
		res = append(res, tok)
	}
	return res, nil
}
