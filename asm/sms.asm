// SDSC tag and SMS rom header
// sdsctag 1.10,"Background Color","SMS Color Tutorial","Stan"

const vdp = 0xbf

// Boot section
org 0x0000
    di              // disable interrupts
    im 1            // Interrupt mode 1
    jp main         // jump to main program

org 0x0066
	// Pause button handler
	// Do nothing
    retn


// Main program
main:
    ld sp, 0xdff0

    // Set up VDP registers

    ld hl, VdpData
    ld b, VdpDataEnd-VdpData
    ld c, vdp
    otir

    // Clear VRAM

    // 1. Set VRAM write address to 0 by outputting 0x4000 ORed with 0x0000
    ld a,0x00
    out (vdp),a
    ld a, 0x40
    out (vdp),a
    // 2. Output 16KB of zeroes to clear the VRAM.
    ld bc, 0x4000    // Counter for 16KB of VRAM
    ld a, 0x00        // Value to write
    ClearVRAMLoop:
        out (0xbe),a // Output to VRAM address, which is auto-incremented after each write
        dec c
        jp nz,ClearVRAMLoop
        dec b
        jp nz,ClearVRAMLoop


    // Load palette

    // 1. Set VRAM write address to CRAM (palette) address 0 (for palette index 0)
    // by outputting 0xC000 ORed with 0x0000
    ld a, 0x00
    out (vdp), a
    ld a, 0xc0
    out (vdp), a
    // 2. Output color data
    ld hl, PaletteData
    ld b, PaletteDataEnd-PaletteData
    ld c, 0xbe
    otir

    // Background Color
    // THIS IS OUR ACTUAL BACKGROUND COLOR PROGRAM

Background:
    ld a,0x00
    out (vdp),a
    ld a,0x00
    out (vdp),a
    ld a,0xc4       // Turn on the screen
    out (vdp),a
    ld a,0x81
    out (vdp),a

// Data for our program to use


PaletteData:
db 0x19
PaletteDataEnd:

// VDP initialisation data
VdpData:
db 0x04,0x80,0x84,0x81,0xff,0x82,0xff,0x85,0xff,0x86,0xff,0x87,0x00,0x88,0x00,0x89,0xff,0x8a
VdpDataEnd:

// This background color program was built using information found
// in Maxim of SMS Power fame's original programming tutorial.
// All information concerning the setting up of the VDP before the
// actual background code is his creation and the author owes him a
// great deal of gratitude for writing it.
// Background Color Demo Copyright Stan 2009.
// THIS IS THE ABSOLUTE END OF OUR PROGRAM DEMO

