package plz_test

import "testing"

func TestIntegrationLoopComplexCallFor(t *testing.T) {
	io := compileAndRun(t, `

PROCEDURE write_rep(b BYTE, rep BYTE)
	DECLARE ii BYTE
	FOR ii = 0 TO rep DO
		OUTPUT 0 b
	END
END

PROCEDURE abc()
	DECLARE cnt WORD
	DECLARE cnt2 WORD
	FOR cnt = 'A' TO 'Y' DO
		LET cnt2 = cnt + 1
		CALL write_rep(cnt2, 2)
	END
END

CALL abc()
HALT
`)
	expect := 25
	if len(io.OutBytes[0]) != expect {
		t.Fatalf("expected %d outputs, got %d: %v", expect, len(io.OutBytes[0]), io.OutBytes[0])
	}
	outs := string(io.OutBytes[0][0:expect])
	t.Logf("outs: %s", outs)
	if outs != "BCDEFGHIJKLMNOPQRSTUVWXYZ" {
		t.Errorf("expected BCDEF, got %s", outs)
	}
}
