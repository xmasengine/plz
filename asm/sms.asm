// Define VDP control and data ports
const VDP_CONTROL = 0xBF
const VDP_DATA = 0xBE

// Define VDP register settings
const VDP_REG_0 = 0b00100000 // Enable display, sprites use first 256 tiles
const VDP_REG_1 = 0b11100000 // Enable display, 32x28 screen, sprites 8x8

// A 8x8 tile pattern (a simple white square)
TilePattern:
    db 0b11111111
    db 0b11111111
    db 0b11111111
    db 0b11111111
    db 0b11111111
    db 0b11111111
    db 0b11111111
    db 0b11111111

// Sprite Attribute Table (SAT)
SpriteTable:
    db 96        // Y position (centered)
    db 0         // Tile index (0)
    db 0         // Attributes (palette 0)
    db 96        // X position (centered)

org 0x0000
Init:
	// no interrupts
	di
    // Initialize stack
    ld sp, 0xDFF0
    // im 1
    im 1
	jmp Start

org 0x80
Start:
    // Turn off the screen
    call WaitVBlank
    ld a, VDP_REG_0
    out (VDP_CONTROL), a
    ld a, VDP_REG_1
    out (VDP_CONTROL), a

    // Load the tile into pattern table 0 (VRAM 0x0000-0x1FFF)
    ld hl, TilePattern
    ld de, 0x4000 // VDP write command for VRAM 0x0000
    ld bc, 8     // 8 bytes for one tile
    call LoadDataToVRAM

    // Set the Sprite Attribute Table (SAT) address
    ld de, SpriteTable
    ld hl, 0x8300 // VDP command to set SAT to 0x300
    ld a, l
    out (VDP_CONTROL), a
    ld a, h
    out (VDP_CONTROL), a

    // Turn on the screen
    ld a, VDP_REG_1 | 0b00010000 // Turn display on
    out (VDP_CONTROL), a

MainLoop:
    halt         // Wait for VBlank interrupt
    jr MainLoop

// Subroutine to load data from CPU memory to VRAM
LoadDataToVRAM:
	ld a, e
    out (VDP_CONTROL), a
    ld a, d
    out (VDP_CONTROL), a
LoadDataLoop:
    ld a, (hl)
    out (VDP_DATA), a
    inc hl
    dec bc
    ld a, b
    or c
    jr nz, LoadDataLoop
    ret

// Subroutine to wait for VBlank
WaitVBlank:
	ei
    in a, (VDP_CONTROL)
    bit 7, a
    jr nz, WaitVBlank
    ret