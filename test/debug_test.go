package plz_test

import (
	"fmt"
	"testing"
)

func TestDebugRun(t *testing.T) {
	src := `DECLARE s RECORD x WORD, y BYTE END
LET s.x = 300
LET s.y = 42
OUTPUT 0 s.x
OUTPUT 0 s.y
HALT`
	res := compileAndRunArch(t, src, "z80")
	fmt.Printf("result: %v\n", res.OutBytes)
}
