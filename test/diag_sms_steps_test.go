package plz_test

import (
	"testing"
)

// Test what part of libplz_test.plz causes the hang
func TestDiagSMSLibPlzStep1PIR(t *testing.T) {
	// Just screen_mode_4 + halt_vsync
	v := compileAndRunSMSPIR(t, `
PROCEDURE int_vsync() INTERRUPT
  DECLARE status BYTE
  LET status = INPUT(0xBF)
  ENABLE
END

INTERRUPT int_vsync
DISABLE
OUTPUT 0xBF 0x04
OUTPUT 0xBF 0x80
OUTPUT 0xBF 0xE0
OUTPUT 0xBF 0x81
CALL halt_vsync
HALT

PROCEDURE halt_vsync()
  ENABLE
  HALT
END`)
	if !v.FrameReady() {
		t.Fatal("no frame rendered")
	}
}

func TestDiagSMSLibPlzStep2PIR(t *testing.T) {
	// Same but with ENABLE via write_vdp_reg;
	// R1 must have bit 5 set (0x20) to enable frame interrupts.
	v := compileAndRunSMSPIR(t, `
PROCEDURE int_vsync() INTERRUPT
  DECLARE status BYTE
  LET status = INPUT(0xBF)
  ENABLE
END

PROCEDURE write_vdp_reg(reg BYTE, value BYTE)
  OUTPUT WORD 0xBF value | ((reg | 0x80) << 8)
END

INTERRUPT int_vsync
DISABLE
CALL write_vdp_reg(0, 4)
CALL write_vdp_reg(1, 0xE2)
CALL halt_vsync
HALT

PROCEDURE halt_vsync()
  ENABLE
  HALT
END`)
	if !v.FrameReady() {
		t.Fatal("no frame rendered")
	}
}

// No interrupt handler at all, just direct ENABLE + procedure call
func TestDiagSMSProcCallHalt(t *testing.T) {
	v := compileAndRunSMSPIR(t, `
PROCEDURE halt_vsync()
  ENABLE
  HALT
END

DISABLE
OUTPUT 0xBF 0x04
OUTPUT 0xBF 0x80
OUTPUT 0xBF 0xE0
OUTPUT 0xBF 0x81
CALL halt_vsync
HALT`)
	if !v.FrameReady() {
		t.Fatal("no frame rendered")
	}
}

// Interrupt handler but no INT install
func TestDiagSMSIntNoInstall(t *testing.T) {
	v := compileAndRunSMSPIR(t, `
PROCEDURE int_vsync() INTERRUPT
  DECLARE status BYTE
  LET status = INPUT(0xBF)
  ENABLE
END

ENABLE
HALT
DISABLE
HALT`)
	if !v.FrameReady() {
		t.Fatal("no frame rendered")
	}
}

// Just CALL a procedure (no halt)
func TestDiagSMSSimpleCall(t *testing.T) {
	v := compileAndRunSMSPIR(t, `
PROCEDURE myproc()
  OUTPUT 0xBF 0x04
END

CALL myproc
HALT`)
	if !v.FrameReady() {
		t.Fatal("no frame rendered")
	}
}
