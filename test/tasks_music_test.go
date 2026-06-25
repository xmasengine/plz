package plz_test

import (
	"testing"
)

func TestIntegrationMusic(t *testing.T) {
	io := compileAndRun(t, `
PROCEDURE psg_freq(channel BYTE, freq WORD)
  OUTPUT 0x7F 0x80 | (channel << 5) | (freq & 0x0F)
  OUTPUT 0x7F (freq >> 4) & 0x3F
END
PROCEDURE psg_vol(channel BYTE, vol BYTE)
  OUTPUT 0x7F 0x90 | (channel << 5) | vol
END
PROCEDURE psg_silence()
  CALL psg_vol(0, 15)
  CALL psg_vol(1, 15)
  CALL psg_vol(2, 15)
  OUTPUT 0x7F 0xFF
END
TASK music_test PRIORITY 4
  CALL psg_silence()
  CALL psg_freq(0, 256)
  CALL psg_vol(0, 8)
  SLEEP 1
  CALL psg_vol(0, 15)
  YIELD
END
`)
	expected := []byte{159, 191, 223, 255, 128, 16, 152, 159}
	if len(io.OutBytes[0x7F]) != len(expected) {
		t.Fatalf("expected %d bytes on port 0x7F, got %d: %v", len(expected), len(io.OutBytes[0x7F]), io.OutBytes[0x7F])
	}
	for i, b := range expected {
		if io.OutBytes[0x7F][i] != b {
			t.Errorf("byte %d: expected 0x%02X, got 0x%02X", i, b, io.OutBytes[0x7F][i])
		}
	}
}

func TestIntegrationMusicDataDriven(t *testing.T) {
	io := compileAndRun(t, `
PROCEDURE psg_freq(channel BYTE, freq WORD)
  OUTPUT 0x7F 0x80 | (channel << 5) | (freq & 0x0F)
  OUTPUT 0x7F (freq >> 4) & 0x3F
END
PROCEDURE psg_vol(channel BYTE, vol BYTE)
  OUTPUT 0x7F 0x90 | (channel << 5) | vol
END
PROCEDURE psg_silence()
  CALL psg_vol(0, 15)
  CALL psg_vol(1, 15)
  CALL psg_vol(2, 15)
  OUTPUT 0x7F 0xFF
END

PROCEDURE play_song(song DATA)
  DECLARE idx WORD
  DECLARE freq WORD
  LET idx = 0
  WHILE 1 DO
    IF song[idx+4] == 0xFF THEN RETURN

    LET freq = song[idx] | (song[idx+1] << 8)
    CALL psg_freq(song[idx+3], freq)
    CALL psg_vol(song[idx+3], song[idx+4])
    SLEEP song[idx+2]
    CALL psg_vol(song[idx+3], 15)
    LET idx = idx + 5
  END
END

DATA my_song 0x80, 0x00, 1, 0, 8, 0x40, 0x01, 1, 0, 8, 0, 0, 0, 0, 0xFF

TASK music_test PRIORITY 4
  CALL psg_silence()
  CALL play_song(my_song)
  YIELD
END
`)
	if len(io.OutBytes[0x7F]) < 2 {
		t.Fatalf("expected output on port 0x7F, got none")
	}
	if io.OutBytes[0x7F][0] != 0x9F {
		t.Errorf("expected PSG silence byte 0x9F, got 0x%02X", io.OutBytes[0x7F][0])
	}
	hasFreq := false
	for _, b := range io.OutBytes[0x7F] {
		if b == 0x80 {
			hasFreq = true
			break
		}
	}
	if !hasFreq {
		t.Error("expected PSG frequency byte 0x80 (ch0, low nibble 0)")
	}
	freqCount := 0
	for _, b := range io.OutBytes[0x7F] {
		if b == 0x80 {
			freqCount++
		}
	}
	if freqCount < 2 {
		t.Errorf("expected at least 2 note frequency writes, got %d", freqCount)
	}
}

func TestIntegrationPIR_Yield(t *testing.T) {
	testArchs(t, `
TASK demo PRIORITY 4
  DECLARE x WORD
  LET x = 1
  YIELD
  LET x = x + 1
  OUTPUT 0 x
  YIELD
END`, func(t *testing.T, res *RunResult) {
		if len(res.OutBytes[0]) < 1 {
			t.Fatal("expected output")
		}
		if res.OutBytes[0][0] != 2 {
			t.Errorf("expected 2, got %d", res.OutBytes[0][0])
		}
	})
}
