package plz_test

import "testing"

func TestIntegrationArrayRead(t *testing.T) {
	testArchs(t, `DECLARE arr WORD
DECLARE tmp WORD
LET arr[0] = 42
LET tmp = arr[0]
OUTPUT 0 tmp
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationArrayDeclared(t *testing.T) {
	testArchs(t, `DECLARE arr ARRAY [10] WORD
LET arr[0] = 42
LET arr[1] = 99
OUTPUT 0 arr[0]
OUTPUT 0 arr[1]
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 2 {
			t.Fatal("expected 2 outputs")
		}
		if res.OutBytes[0][0] != 42 {
			t.Errorf("expected 42, got %d", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 99 {
			t.Errorf("expected 99, got %d", res.OutBytes[0][1])
		}
	})
}

func TestIntegrationByteArray(t *testing.T) {
	testArchs(t, `DECLARE arr BYTE
DECLARE idx WORD
LET arr[0] = 10
LET arr[1] = 20
LET idx = 0
LET idx = arr[idx]
OUTPUT 0 idx
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 10 {
			t.Errorf("expected 10, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationRecord(t *testing.T) {
	testArchs(t, `DECLARE s RECORD x WORD, y BYTE END
LET s.x = 300
LET s.y = 42
OUTPUT 0 s.x
OUTPUT 0 s.y
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 2 {
			t.Fatal("expected 2 outputs")
		}
		if res.OutBytes[0][0] != 44 {
			t.Errorf("expected 44 for s.x low byte, got %d", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 42 {
			t.Errorf("expected 42 for s.y, got %d", res.OutBytes[0][1])
		}
	})
}

func TestIntegrationRecordWithArrayField(t *testing.T) {
	testArchs(t, `DECLARE s RECORD x WORD, arr ARRAY [4] BYTE END
DECLARE idx WORD
DECLARE val WORD
LET s.x = 999
LET s.arr[0] = 10
LET s.arr[1] = 20
LET s.arr[2] = 30
LET s.arr[3] = 40
LET idx = 2
LET val = s.arr[idx]
OUTPUT 0 val
LET val = s.x
OUTPUT 0 val
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 2 {
			t.Fatal("expected 2 outputs")
		}
		if res.OutBytes[0][0] != 30 {
			t.Errorf("expected 30 for s.arr[2], got %d", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 231 {
			t.Errorf("expected 231 for s.x low byte, got %d", res.OutBytes[0][1])
		}
	})
}

func TestIntegrationArrayOfRecordsFieldRead(t *testing.T) {
	testArchs(t, `DECLARE arr ARRAY [4] RECORD x WORD, y BYTE END
DECLARE idx WORD
DECLARE val WORD
LET arr[0].x = 100
LET arr[0].y = 50
LET arr[1].x = 200
LET arr[1].y = 77
LET idx = 1
LET val = arr[idx].x
OUTPUT 0 val
LET val = arr[idx].y
OUTPUT 0 val
LET val = arr[0].x
OUTPUT 0 val
LET val = arr[0].y
OUTPUT 0 val
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 4 {
			t.Fatal("expected 4 outputs")
		}
		if res.OutBytes[0][0] != 200 {
			t.Errorf("expected 200 for arr[1].x, got %d", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 77 {
			t.Errorf("expected 77 for arr[1].y, got %d", res.OutBytes[0][1])
		}
		if res.OutBytes[0][2] != 100 {
			t.Errorf("expected 100 for arr[0].x, got %d", res.OutBytes[0][2])
		}
		if res.OutBytes[0][3] != 50 {
			t.Errorf("expected 50 for arr[0].y, got %d", res.OutBytes[0][3])
		}
	})
}

func TestIntegrationDefineRecordAlias(t *testing.T) {
	testArchs(t, `DEFINE Point RECORD x WORD, y BYTE END
DECLARE pt TYPE Point
DECLARE val WORD
LET pt.x = 300
LET pt.y = 42
LET val = pt.x
OUTPUT 0 val
OUTPUT 0 pt.y
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 2 {
			t.Fatal("expected 2 outputs")
		}
		if res.OutBytes[0][0] != 44 {
			t.Errorf("expected 44 for p.x low byte, got %d", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 42 {
			t.Errorf("expected 42 for p.y, got %d", res.OutBytes[0][1])
		}
	})
}

func TestIntegrationDefineByteAlias(t *testing.T) {
	testArchs(t, `DEFINE MyByte BYTE
DECLARE bv TYPE MyByte
DECLARE val WORD
LET bv = 255
LET val = bv
OUTPUT 0 val
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 255 {
			t.Errorf("expected 255, got %d", res.OutBytes[0][0])
		}
	})
}

func TestIntegrationDataIndexBoundCheck(t *testing.T) {
	testArchs(t, `
PRAGMA BOUNDCHECK
DATA my_data 10, 20, 30
OUTPUT 0 my_data[0]
OUTPUT 0 my_data[1]
OUTPUT 0 my_data[2]
HALT`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 3 {
			t.Fatal("expected 3 outputs")
		}
		if res.OutBytes[0][0] != 10 {
			t.Errorf("expected 10, got %d", res.OutBytes[0][0])
		}
		if res.OutBytes[0][1] != 20 {
			t.Errorf("expected 20, got %d", res.OutBytes[0][1])
		}
		if res.OutBytes[0][2] != 30 {
			t.Errorf("expected 30, got %d", res.OutBytes[0][2])
		}
	})
}
