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
	KeywordDisable
	KeywordDo
	KeywordEnable
	KeywordEnd
	KeywordHalt
	KeywordGoTo
	KeywordInput
	KeywordOutput
)

var Keywords = map[string]TokenKind{
	"DISABLE": KeywordDisable,
	"DO":      KeywordDo,
	"ENABLE":  KeywordEnable,
	"GOTO":    KeywordGoTo,
	"END":     KeywordEnd,
	"HALT":    KeywordHalt,
	"INPUT":   KeywordInput,
	"OUTPUT":  KeywordOutput,
}

type Token struct {
	TokenKind TokenKind
	Position  Position
	Text      string
	Number    int
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
			tok.Number, err = strconv.Atoi(tok.Text)
			if err != nil {
				return nil, tok.Errorf("int %s %s", tok.Text, err.Error())
			}
		} else if kind == TokenChar {
			r, _, _, err := strconv.UnquoteChar(tok.Text[1:len(tok.Text)], '\'')
			tok.Number = int(r)
			if err != nil {
				return nil, tok.Errorf("char %s %s", tok.Text, err.Error())
			}
		} else if kind == TokenRawString {
			tok.TokenKind = TokenString
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
