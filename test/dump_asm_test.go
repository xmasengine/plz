package plz_test

import (
	"testing"
	"fmt"
	"github.com/xmasengine/plz/pkg/pir"
)

func TestDumpReentrantAsm(t *testing.T) {
	src := `DATA my_data 10, 20, 30
PROCEDURE foo(d DATA) REENTRANT
  OUTPUT 0 LENGTH(d)
END
CALL foo(my_data)
HALT`
	prog := compilePIR(t, src)
	asm := pir.NewZ80Gen(pir.DefaultConfig()).Gen(prog)
	fmt.Println(asm)
	t.Log(asm)
}
