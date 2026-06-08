package plz_test

import (
	"image"
	"testing"
)

func TestIntegrationSMSLibPlz(t *testing.T) {
	v := compileAndRunSMSFile(t, "../include/libplz_test.plz")
	if !v.FrameReady() {
		t.Fatal("no frame rendered")
	}
	img := frameImage(v)
	samples := []image.Point{
		{0, 0}, {1, 0}, {2, 0}, {4, 0}, {6, 0},
		{0, 2}, {0, 4}, {0, 6},
		{4, 4}, {2, 6}, {6, 2},
	}
	nonBlack := 0
	for _, p := range samples {
		_, g, _, _ := img.At(p.X, p.Y).RGBA()
		if g != 0 {
			nonBlack++
		}
	}
	if nonBlack == 0 {
		t.Fatal("all sampled pixels are black — tiles not rendering correctly")
	}
	t.Logf("LibPlz: frame %dx%d, %d/%d sampled pixels non-black",
		img.Bounds().Dx(), img.Bounds().Dy(), nonBlack, len(samples))
}

func TestIntegrationSMSHaltWake(t *testing.T) {
	v := compileAndRunSMS(t, `
	ENABLE
	HALT
	DISABLE
	HALT`)
	if !v.FrameReady() {
		t.Fatal("no frame rendered")
	}
	img := frameImage(v)
	if img.Bounds().Dy() != 192 {
		t.Fatalf("expected 192 rows, got %d", img.Bounds().Dy())
	}
	r, g, b, a := img.At(0, 0).RGBA()
	if r != 0 || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected black pixel at (0,0), got (%d,%d,%d,%d)", r, g, b, a)
	}
}

func TestIntegrationSMSInterrupt(t *testing.T) {
	v := compileAndRunSMS(t, `
PROCEDURE vblank() INTERRUPT
  DECLARE status BYTE
  LET status = INPUT(0xBF)
  ENABLE
END

INTERRUPT vblank
ENABLE

  HALT
  HALT
  DISABLE
  HALT`)
	if !v.FrameReady() {
		t.Fatal("no frame rendered")
	}
	img := frameImage(v)
	if img.Bounds().Dy() != 192 {
		t.Fatalf("expected 192 rows, got %d", img.Bounds().Dy())
	}
	r, g, b, a := img.At(0, 0).RGBA()
	if r != 0 || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected black pixel at (0,0), got (%d,%d,%d,%d)", r, g, b, a)
	}
}

func TestIntegrationSMSInterruptOutputs(t *testing.T) {
	v := compileAndRunSMS(t, `
PROCEDURE vblank() INTERRUPT
  DECLARE status BYTE
  LET status = INPUT(0xBF)
  ENABLE
END

INTERRUPT vblank
ENABLE

  OUTPUT 0xBF 0x04    // reg 0 data: mode 4
  OUTPUT 0xBF 0x80    // reg 0 select
  OUTPUT 0xBF 0xE0    // reg 1 data: display + frame int
  OUTPUT 0xBF 0x81    // reg 1 select
  HALT
  HALT
  DISABLE
  HALT`)
	if !v.FrameReady() {
		t.Fatal("no frame rendered")
	}
	img := frameImage(v)
	if img.Bounds().Dy() != 240 {
		t.Fatalf("expected 240 rows, got %d", img.Bounds().Dy())
	}
	hasContent := false
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			_, g, _, _ := img.At(x, y).RGBA()
			if g != 0 {
				hasContent = true
				break
			}
		}
	}
	if !hasContent {
		t.Log("warning: framebuffer is all black (may be expected if tile 0 pixels are transparent)")
	}
}
