package plz_test

import (
	"bytes"
	"testing"
	"github.com/xmasengine/plz/pkg/asm6502"

	"github.com/xmasengine/plz/pkg/cpu6502"
)

func TestRaw6502_ArrayIndex(t *testing.T) {
	src := `
	.org $1000

	jmp _6502_main

_6502_main:
	sei
	cld
	ldx #$ff
	txs
	lda #<12288
	sta $00
	lda #>12288
	sta $01
	lda #<20480
	sta $0c
	lda #>20480
	sta $0d

	; Global init: src[0]=10, src[1]=20, src[2]=30
	lda #10
	ldx #0
	jsr spillW
	lda #<_v_src
	ldx #>_v_src
	jsr spillW
	jsr swap
	jsr writeB

	lda #20
	ldx #0
	jsr spillW
	lda #<_v_src
	ldx #>_v_src
	jsr spillW
	lda #1
	ldx #0
	jsr spillW
	jsr addW
	jsr swap
	jsr writeB

	lda #30
	ldx #0
	jsr spillW
	lda #<_v_src
	ldx #>_v_src
	jsr spillW
	lda #2
	ldx #0
	jsr spillW
	jsr addW
	jsr swap
	jsr writeB

	; idx = 1
	lda #1
	ldx #0
	jsr spillW
	jsr putB_idx

	; output src[idx]
	lda #<_v_src
	ldx #>_v_src
	jsr spillW
	lda _v_idx
	ldx #0
	jsr spillW
	jsr addW
	jsr readB
	jsr outB

	jmp _6502_all_done

spillW:
	sta ($00),y
	inc $00
	bne _spw_1
	inc $01
_spw_1:
	txa
	sta ($00),y
	inc $00
	bne _spw_2
	inc $01
_spw_2:
	rts

fillW:
	dec $00
	lda #$ff
	cmp $00
	bne _fw_1
	dec $01
_fw_1:
	ldy #0
	lda ($00),y
	tax
	dec $00
	lda #$ff
	cmp $00
	bne _fw_2
	dec $01
_fw_2:
	ldy #0
	lda ($00),y
	rts

swap:
	jsr fillW
	sta $02
	stx $03
	jsr fillW
	pha
	txa
	pha
	lda $02
	ldx $03
	jsr spillW
	pla
	tax
	pla
	jsr spillW
	rts

writeB:
	jsr fillW
	pha
	jsr fillW
	stx $05
	sta $04
	pla
	ldy #0
	sta ($04),y
	rts

addW:
	jsr fillW
	sta $02
	stx $03
	jsr fillW
	pha
	clc
	adc $02
	sta $02
	pla
	txa
	adc $03
	tax
	lda $02
	jsr spillW
	rts

readB:
	jsr fillW
	stx $05
	sta $04
	ldy #0
	lda ($04),y
	ldx #0
	jsr spillW
	rts

outB:
	jsr fillW
	ldy #0
	sta ($0c),y
	inc $0c
	bne _out_b_1
	inc $0d
_out_b_1:
	rts

putB_idx:
	jsr fillW
	sta _v_idx
	rts

_6502_all_done:
	brk

_v_src:
	.byte 0, 0, 0
_v_idx:
	.byte 0
`
	r := bytes.NewReader([]byte(src))
	assembly, _, err := asm.Assemble(r, "plz", 0x1000, nil, 0)
	if err != nil || len(assembly.Errors) > 0 {
		for _, e := range assembly.Errors {
			t.Logf("asm error: %s", e)
		}
		t.Fatalf("assemble: %v", err)
	}
	bin := assembly.Code
	t.Logf("Binary size: %d bytes", len(bin))
	t.Logf("Binary addresses: $1000 - $%04X", 0x1000+len(bin)-1)

	mem := cpu.NewFlatMemory()
	off := int(0x1000)
	for i, b := range bin {
		mem.StoreByte(uint16(off+i), b)
	}

	emu := cpu.NewCPU(cpu.NMOS, mem)
	emu.Reg.PC = 0x1000

	bd := &brkDone6502{}
	emu.AttachBrkHandler(bd)

	const maxSteps = 500000
	for i := 0; i < maxSteps && !bd.done; i++ {
		emu.Step()
	}
	if !bd.done {
		t.Fatalf("program did not complete after %d steps; PC=$%04X", maxSteps, emu.Reg.PC)
	}

	t.Logf("_v_src references:")
	for i := 0; i < len(bin)-3; i++ {
		if bin[i] == 0xA9 && bin[i+2] == 0xA2 {
			low := uint16(bin[i+1])
			high := uint16(bin[i+3]) << 8
			if high|low >= 0x1120 {
				t.Logf("  LDA #$%02X / LDX #$%02X at offset 0x%X (addr $%04X)", bin[i+1], bin[i+3], i, high|low)
			}
		}
	}
	t.Logf("STA absolute references:")
	for i := 0; i < len(bin)-3; i++ {
		if bin[i] == 0x8D {
			addr := uint16(bin[i+1]) | uint16(bin[i+2])<<8
			if addr >= 0x1120 && addr < 0x1140 {
				t.Logf("  STA $%04X at binary offset 0x%X", addr, i)
			}
		}
	}
	t.Logf("LDA absolute references:")
	for i := 0; i < len(bin)-3; i++ {
		if bin[i] == 0xAD {
			addr := uint16(bin[i+1]) | uint16(bin[i+2])<<8
			if addr >= 0x1120 && addr < 0x1140 {
				t.Logf("  LDA $%04X at binary offset 0x%X", addr, i)
			}
		}
	}

	t.Logf("Memory:")
	t.Logf("  $1129: %d", mem.LoadByte(0x1129))
	t.Logf("  $112A: %d", mem.LoadByte(0x112A))
	t.Logf("  $112B: %d", mem.LoadByte(0x112B))
	t.Logf("  $112C: %d", mem.LoadByte(0x112C))
	t.Logf("  $112D: %d", mem.LoadByte(0x112D))
	t.Logf("  $112E: %d", mem.LoadByte(0x112E))

	out := mem.LoadByte(0x5000)
	t.Logf("Output: %d (expected 20)", out)
	if out != 20 {
		t.Errorf("output = %d, expected 20", out)
	}
}
