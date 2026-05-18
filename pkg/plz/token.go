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
	KeywordDo
	KeywordEnd
	KeywordInput
	KeywordOutput
)

var Keywords = map[string]TokenKind{
	"DO":     KeywordDo,
	"END":    KeywordEnd,
	"INPUT":  KeywordInput,
	"OUTPUT": KeywordOutput,
}

type Token struct {
	TokenKind
	Position
	Text   string
	Number int
}

func Scan(rd io.Reader) ([]Token, error) {
	var err error
	var res []Token
	s := &scanner.Scanner{}
	s.Init(rd)
	s.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanStrings | scanner.ScanRawStrings | scanner.ScanComments | scanner.SkipComments
	s.Error = func(s2 *scanner.Scanner, msg string) {
		err = errors.New(s2.Pos().String() + ":" + msg)
	}
	for kind := TokenKind(s.Next()); kind != TokenEOF; kind = TokenKind(s.Next()) {
		if err != nil {
			return nil, err
		}
		tok := Token{TokenKind: kind, Text: s.TokenText()}
		if kind == TokenInt {
			tok.Number, err = strconv.Atoi(tok.Text)
		} else if kind == TokenRawString {
			tok.TokenKind = TokenString
		} else if kw, ok := Keywords[tok.Text]; ok {
			tok.TokenKind = kw
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
