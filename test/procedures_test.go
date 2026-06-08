package plz_test

import (
	"testing"
)

func TestIntegrationProcCall(t *testing.T) {
	io := compileAndRun(t, `DECLARE result WORD
PROCEDURE add (x WORD, y WORD) WORD
  RETURN x + y
END
CALL add(2, 3)
LET result = add(2, 3)
OUTPUT 0 result
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 5 {
		t.Errorf("expected 5, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationProcLocalDeclare(t *testing.T) {
	io := compileAndRun(t, `DECLARE result WORD
PROCEDURE double (x WORD) WORD
  DECLARE t WORD
  LET t = x + x
  RETURN t
END
LET result = double(21)
OUTPUT 0 result
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationProcByteParam(t *testing.T) {
	io := compileAndRun(t, `DECLARE result WORD
PROCEDURE double (x BYTE) WORD
  RETURN x + x
END
LET result = double(21)
OUTPUT 0 result
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationProcByteReturn(t *testing.T) {
	io := compileAndRun(t, `DECLARE result WORD
PROCEDURE getByte BYTE
  RETURN 42
END
LET result = getByte()
OUTPUT 0 result
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationProcRecordParam(t *testing.T) {
	io := compileAndRun(t, `DECLARE s RECORD x BYTE, y WORD END
DECLARE val WORD
PROCEDURE getX (rv RECORD x BYTE, y WORD END) WORD
  RETURN rv.x
END
LET s.x = 42
LET s.y = 100
LET val = getX(s)
OUTPUT 0 val
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 42 {
		t.Errorf("expected 42, got %d", io.OutBytes[0][0])
	}
}

func TestIntegrationTemplateProcCall(t *testing.T) {
	io := compileAndRun(t, `
DECLARE result WORD
DECLARE register WORD

PROCEDURE set_register (x WORD, y WORD)
	LET register = x + y
END

TEMPLATE REG "CALL set_register($1, $2)"

REG(2, 3)
OUTPUT 0 register
HALT`)
	if len(io.OutBytes[0]) < 1 {
		t.Fatal("expected output")
	}
	if io.OutBytes[0][0] != 5 {
		t.Errorf("expected 5, got %d", io.OutBytes[0][0])
	}
}
