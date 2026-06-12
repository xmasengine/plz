package plz_test

import "testing"

func TestArchsQuickCheck(t *testing.T) {
	testArchs(t, `OUTPUT 0 42
HALT`, func(t *testing.T, res *RunResult) {
		if v := res.OutBytes[0][0]; v != 42 {
			t.Errorf("expected 42, got %d", v)
		}
	})
}
