package plz_test

import "testing"

func TestIntegrationPIR_LoopComplexCallFor(t *testing.T) {
	testArchs(t, `
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
`, func(t *testing.T, res *RunResult) {
		// 25 letters (A-Y), each repeated 3 times (FOR ii=0 TO 2): 75 total
		expect := 75
		if len(res.OutBytes[0]) != expect {
			t.Fatalf("expected %d outputs, got %d", expect, len(res.OutBytes[0]))
		}
		outs := string(res.OutBytes[0])
		expectedOut := ""
		for ch := 'B'; ch <= 'Z'; ch++ {
			for j := 0; j < 3; j++ {
				expectedOut += string(ch)
			}
		}
		if outs != expectedOut {
			t.Errorf("expected %s, got %s", expectedOut, outs)
		}
	})
}
