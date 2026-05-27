package plz

import "text/scanner"
import "errors"
import "strconv"
import "io"
import "os"

type Position = scanner.Position

type TokenKind rune

const (
	TokenEOF TokenKind = -(iota + 1)
	TokenIdent
	TokenInt
	TokenFloat // to match Scanner token kinds
	TokenChar
	TokenString
	TokenRawString // to match Scanner token kinds
	TokenComment
	KeywordArray
	KeywordByte
	KeywordCall
	KeywordConstant
	KeywordData
	KeywordDeclare
	KeywordDisable
	KeywordDo
	KeywordEnable
	KeywordEnd
	KeywordHalt
	KeywordGoTo
	KeywordIf
	KeywordThen
	KeywordElse
	KeywordInput
	KeywordLet
	KeywordReturn
	KeywordStruct
	KeywordOutput
	KeywordWord
	KeywordWhile
	KeywordFor
	KeywordTo
	KeywordBy
	KeywordCase
	KeywordOf
)

func (t TokenKind) String() string {
	for k, v := range Keywords {
		if t == v {
			return k
		}
	}
	return scanner.TokenString(rune(t))
}

var Keywords = map[string]TokenKind{
	"ARRAY":   KeywordArray,
	"BYTE":    KeywordByte,
	"CALL":     KeywordCall,
	"CONSTANT": KeywordConstant,
	"DATA":     KeywordData,
	"DECLARE":  KeywordDeclare,
	"DISABLE":  KeywordDisable,
	"DO":       KeywordDo,
	"ENABLE":   KeywordEnable,
	"GOTO":     KeywordGoTo,
	"END":      KeywordEnd,
	"IF":       KeywordIf,
	"THEN":     KeywordThen,
	"ELSE":     KeywordElse,
	"HALT":     KeywordHalt,
	"INPUT":    KeywordInput,
	"LET":      KeywordLet,
	"RETURN":   KeywordReturn,
	"OUTPUT":   KeywordOutput,
	"STRUCT":   KeywordStruct,
	"WHILE":    KeywordWhile,
	"FOR":      KeywordFor,
	"TO":       KeywordTo,
	"BY":       KeywordBy,
	"CASE":     KeywordCase,
	"OF":       KeywordOf,
	"WORD":     KeywordWord,
}

type Token struct {
	TokenKind TokenKind
	Position  Position
	Text      string
	Number    int
}

func (t Token) String() string {
	return t.Position.String() + " " + t.TokenKind.String() + " " + t.Text
}

func Scan(rd io.Reader) ([]Token, error) {
	var err error
	var res []Token
	s := &scanner.Scanner{}
	s.Init(rd)
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
		tok := Token{TokenKind: kind, Text: s.TokenText(), Position: s.Pos()}
		if kind == TokenInt {
			num, err := strconv.ParseInt(tok.Text, 0, 0)
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

func ScanFile(name string) ([]Token, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Scan(f)
}
