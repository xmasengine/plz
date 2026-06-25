package plz_test

import "testing"

func TestIntegrationDataParamLength(t *testing.T) {
	testArchs(t, `
DATA my_data 10, 20, 30
PROCEDURE foo(d DATA)
  OUTPUT 0 LENGTH(d)
END
CALL foo(my_data)
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected 1 output")
		}
		if res.OutBytes[0][0] != 3 {
			t.Errorf("expected LENGTH=3, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationDataParamLengthLet(t *testing.T) {
	testArchs(t, `
DATA my_data 10, 20, 30
DECLARE len BYTE
PROCEDURE foo(d DATA)
  LET len = LENGTH(d)
END
CALL foo(my_data)
OUTPUT 0 len
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected 1 output")
		}
		if res.OutBytes[0][0] != 3 {
			t.Errorf("expected LENGTH=3, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationDataParamLengthPassthrough(t *testing.T) {
	testArchs(t, `
DATA my_data 10, 20, 30
PROCEDURE bar(d DATA)
  OUTPUT 0 LENGTH(d)
END
PROCEDURE foo(d DATA)
  CALL bar(d)
END
CALL foo(my_data)
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected 1 output")
		}
		if res.OutBytes[0][0] != 3 {
			t.Errorf("expected LENGTH=3, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationDataParamLengthReentrant(t *testing.T) {
	testArchs(t, `
DATA my_data 10, 20, 30
PROCEDURE foo(d DATA) REENTRANT
  OUTPUT 0 LENGTH(d)
END
CALL foo(my_data)
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected 1 output")
		}
		if res.OutBytes[0][0] != 3 {
			t.Errorf("expected LENGTH=3, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationDataParamLengthFourValues(t *testing.T) {
	testArchs(t, `
DATA my_data 10, 20, 30, 40
PROCEDURE foo(d DATA)
  OUTPUT 0 LENGTH(d)
END
CALL foo(my_data)
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected 1 output")
		}
		if res.OutBytes[0][0] != 4 {
			t.Errorf("expected LENGTH(my_data)=4, got %d", res.OutBytes[0][0])
		}
	})
}
