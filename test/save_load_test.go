package plz_test

import (
	"testing"
)

func TestIntegrationSave(t *testing.T) {
	io := compileAndRun(t, `
DECLARE src ARRAY [4] BYTE
LET src[0] = 222
LET src[1] = 173
LET src[2] = 190
LET src[3] = 239
DECLARE dst ARRAY [4] BYTE AT 0x8000
SAVE AT 0x8000 src
OUTPUT 0 dst[0]
OUTPUT 0 dst[1]
OUTPUT 0 dst[2]
OUTPUT 0 dst[3]
HALT`)
	if len(io.OutBytes[0]) < 4 {
		t.Fatal("expected 4 outputs")
	}
	if io.OutBytes[0][0] != 222 {
		t.Errorf("expected 222, got %d", io.OutBytes[0][0])
	}
	if io.OutBytes[0][1] != 173 {
		t.Errorf("expected 173, got %d", io.OutBytes[0][1])
	}
	if io.OutBytes[0][2] != 190 {
		t.Errorf("expected 190, got %d", io.OutBytes[0][2])
	}
	if io.OutBytes[0][3] != 239 {
		t.Errorf("expected 239, got %d", io.OutBytes[0][3])
	}
}

func TestIntegrationSaveWithData(t *testing.T) {
	io := compileAndRun(t, `
DATA my_data 10, 20, 30
DECLARE dst ARRAY [3] BYTE AT 0x8000
SAVE AT 0x8000 my_data
OUTPUT 0 dst[0]
OUTPUT 0 dst[1]
OUTPUT 0 dst[2]
HALT`)
	if len(io.OutBytes[0]) < 3 {
		t.Fatal("expected 3 outputs")
	}
	if io.OutBytes[0][0] != 10 {
		t.Errorf("expected 10, got %d", io.OutBytes[0][0])
	}
	if io.OutBytes[0][1] != 20 {
		t.Errorf("expected 20, got %d", io.OutBytes[0][1])
	}
	if io.OutBytes[0][2] != 30 {
		t.Errorf("expected 30, got %d", io.OutBytes[0][2])
	}
}

func TestIntegrationSavePersist(t *testing.T) {
	io := compileAndRun(t, `
DECLARE src ARRAY [2] BYTE
LET src[0] = 1
LET src[1] = 2
DECLARE dst ARRAY [2] BYTE AT 0x8000
SAVE AT 0x8000 src
LET src[0] = 99
LET src[1] = 100
OUTPUT 0 dst[0]
OUTPUT 0 dst[1]
HALT`)
	if len(io.OutBytes[0]) < 2 {
		t.Fatal("expected 2 outputs")
	}
	if io.OutBytes[0][0] != 1 {
		t.Errorf("expected 1 (saved value), got %d", io.OutBytes[0][0])
	}
	if io.OutBytes[0][1] != 2 {
		t.Errorf("expected 2 (saved value), got %d", io.OutBytes[0][1])
	}
}

func TestIntegrationLoad(t *testing.T) {
	io := compileAndRun(t, `
DECLARE src ARRAY [4] BYTE
LET src[0] = 10
LET src[1] = 20
LET src[2] = 30
LET src[3] = 40
SAVE AT 0x8000 src
LET src[0] = 0
LET src[1] = 0
LET src[2] = 0
LET src[3] = 0
LOAD AT 0x8000 src
OUTPUT 0 src[0]
OUTPUT 0 src[1]
OUTPUT 0 src[2]
OUTPUT 0 src[3]
HALT`)
	if len(io.OutBytes[0]) < 4 {
		t.Fatal("expected 4 outputs")
	}
	if io.OutBytes[0][0] != 10 {
		t.Errorf("expected 10, got %d", io.OutBytes[0][0])
	}
	if io.OutBytes[0][1] != 20 {
		t.Errorf("expected 20, got %d", io.OutBytes[0][1])
	}
	if io.OutBytes[0][2] != 30 {
		t.Errorf("expected 30, got %d", io.OutBytes[0][2])
	}
	if io.OutBytes[0][3] != 40 {
		t.Errorf("expected 40, got %d", io.OutBytes[0][3])
	}
}

func TestIntegrationLoadRoundTrip(t *testing.T) {
	io := compileAndRun(t, `
DECLARE src ARRAY [4] BYTE
DECLARE dst ARRAY [4] BYTE
LET src[0] = 100
LET src[1] = 101
LET src[2] = 102
LET src[3] = 103
SAVE AT 0x8000 src
LOAD AT 0x8000 dst
OUTPUT 0 dst[0]
OUTPUT 0 dst[1]
OUTPUT 0 dst[2]
OUTPUT 0 dst[3]
HALT`)
	if len(io.OutBytes[0]) < 4 {
		t.Fatal("expected 4 outputs")
	}
	if io.OutBytes[0][0] != 100 {
		t.Errorf("expected 100, got %d", io.OutBytes[0][0])
	}
	if io.OutBytes[0][1] != 101 {
		t.Errorf("expected 101, got %d", io.OutBytes[0][1])
	}
	if io.OutBytes[0][2] != 102 {
		t.Errorf("expected 102, got %d", io.OutBytes[0][2])
	}
	if io.OutBytes[0][3] != 103 {
		t.Errorf("expected 103, got %d", io.OutBytes[0][3])
	}
}

func TestIntegrationLoadWithData(t *testing.T) {
	io := compileAndRun(t, `
DATA my_data 100, 200, 128, 255
DECLARE buf ARRAY [4] BYTE
SAVE AT 0x8000 my_data
LOAD AT 0x8000 buf
OUTPUT 0 buf[0]
OUTPUT 0 buf[1]
OUTPUT 0 buf[2]
OUTPUT 0 buf[3]
HALT`)
	if len(io.OutBytes[0]) < 4 {
		t.Fatal("expected 4 outputs")
	}
	if io.OutBytes[0][0] != 100 {
		t.Errorf("expected 100, got %d", io.OutBytes[0][0])
	}
	if io.OutBytes[0][1] != 200 {
		t.Errorf("expected 200, got %d", io.OutBytes[0][1])
	}
	if io.OutBytes[0][2] != 128 {
		t.Errorf("expected 128, got %d", io.OutBytes[0][2])
	}
	if io.OutBytes[0][3] != 255 {
		t.Errorf("expected 255, got %d", io.OutBytes[0][3])
	}
}
