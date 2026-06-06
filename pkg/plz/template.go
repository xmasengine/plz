package plz

import "strconv"
import "fmt"
import "os"
import "errors"

// Template is a very simple template engine that supports only
// shell expansions like $1 ${2} as provided by os.Expand. But it is enough.
type Template string

type tokenExpander struct {
	errors []error
	tokens []Token
}

// expandTokens is used with os.Expand to injec tokens into a template.
func (e *tokenExpander) expand(placeholder string) string {
	i, err := strconv.Atoi(placeholder)
	if err != nil {
		e.errors = append(e.errors, err)
		return ""
	}

	if i < 1 || i > len(e.tokens) {
		e.errors = append(e.errors,
			fmt.Errorf("template parameter %s out of range", placeholder),
		)
		return ""
	}
	// we do -1 to support $1 $2 in stead of $0 $1
	// Use the raw value of the token so we don't have to re-quote it.
	return e.tokens[i-1].Raw
}

func (t Template) ExpandTokens(args ...Token) (string, error) {
	expander := tokenExpander{tokens: args}

	expanded := os.Expand(string(t), expander.expand)

	if len(expander.errors) > 0 {
		return "", fmt.Errorf("template %s error: %s", string(t), errors.Join(expander.errors...))
	}
	println("template", expanded)
	return expanded, nil
}
