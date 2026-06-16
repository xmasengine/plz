// Package sms implements the Sega Master System Video Display Processor (VDP)
// and related hardware. Written from scratch based on:
//   - Charles MacDonald's SMS VDP documentation (doc/sms-vdp.txt)
//   - SMS Power! hardware reference (sms-technical.txt, smsgg-technical.txt)
//   - Reference implementations (all MIT-licensed, used for format/behavior
//     understanding only, not code copying):
//   - github.com/koron-go/vdp  (MURAOKA Taro)
//   - github.com/mrcook/smstilemap  (Michael R. Cook)
//   - github.com/remogatto/sms  (Andrea Fazzi)
//   - github.com/user-none/go-chip-sn76489  (John Schember)
package sms

const (
	VRAMSize    = 16 * 1024 // VRAMSize is the 16K of video RAM the SMS has.
	CRAMSize    = 64        // CRAMSize is the size of the SMS color ram.
	ScreenWidth = 256       // ScreenWidth is the maximum screen width for the VDP.
	MaxHeight   = 240       // MaxHeight is the maximum screen heigth for the VDP.

	HblankCycles = 342 // HblankCycles is the amount of cycles until the horizontal blank.
	LinesNTSC    = 262 // LinesNTSC is the amount of lines in NTSC mode.
	LinesPAL     = 313 // LinesPAL is the amount of lines in PAL mode.

	ScreenHeightNTSC = 192 // ScreenHeightNTSC screen heigth in NTSC mode.
)

// VDM is ane mulation of the SMS video Display processor.
type VDP struct {
	VRAM [VRAMSize]byte // VRAM is the video RAM.
	CRAM [CRAMSize]byte // CRAM is the color RAM.

	addrReg    uint16
	codeReg    byte
	secondByte bool
	readBuf    byte

	reg [11]byte

	displayEnable    bool
	frameIntEnable   bool
	lineIntEnable    bool
	mode4            bool
	largeSprites     bool
	stretchedSprites bool
	shiftLeft        bool
	maskCol0         bool
	disableHScroll   bool
	disableVScroll   bool

	nameTableAddr       uint16
	spriteAttrTableAddr uint16
	spriteTileTableAddr uint16

	scrollX int16
	scrollY int16

	backdropColor byte
	backdropIdx   byte

	lineIntCounter byte
	lineIntReload  byte

	totalLines  uint16
	frameHeight uint16
	scanline    uint16
	dot         uint16
	cycleCount  uint64

	vblankPending bool
	framePending  bool
	linePending   bool
	sprOverflow   bool
	sprCollision  bool

	framebuffer [ScreenWidth * MaxHeight * 4]byte
	frameReady  bool

	IsGameGear bool
	vCounter   byte
	hCounter   byte
}

// New allocates a new VDP.
func New(isGG bool) *VDP {
	v := &VDP{IsGameGear: isGG}
	v.Reset()
	return v
}

// Reset restes the VDP to the initial state.
func (v *VDP) Reset() {
	v.scanline = 0
	v.dot = 0
	v.cycleCount = 0
	v.frameHeight = ScreenHeightNTSC
	v.totalLines = LinesNTSC
	v.secondByte = false
	v.addrReg = 0
	v.codeReg = 0
	v.readBuf = 0
	v.frameReady = false
	v.vblankPending = false
	v.framePending = false
	v.linePending = false
	v.sprOverflow = false
	v.sprCollision = false
	v.reg = [11]byte{}
	v.displayEnable = true
	v.mode4 = true
	v.lineIntEnable = false
	v.frameIntEnable = true
	v.scrollX = 0
	v.scrollY = 0
	v.backdropColor = 0
	v.backdropIdx = 0
	v.lineIntReload = 0
	v.lineIntCounter = 0
	v.vCounter = 0
	v.hCounter = 0

	for i := range v.VRAM {
		v.VRAM[i] = 0
	}
	for i := range v.CRAM {
		v.CRAM[i] = 0
	}
	for i := range v.framebuffer {
		v.framebuffer[i] = 0
	}
}

func (v *VDP) WriteControl(val byte) {
	if !v.secondByte {
		v.addrReg = (v.addrReg & 0xFF00) | uint16(val)
		v.secondByte = true
		return
	}
	v.secondByte = false

	if val&0x80 != 0 && val&0x40 == 0 {
		regNum := val & 0x0F
		data := byte(v.addrReg & 0xFF)
		v.setReg(regNum, data)
		return
	}

	code := (val >> 6) & 0x03
	addr := (uint16(val&0x3F) << 8) | (v.addrReg & 0x00FF)

	v.addrReg = addr & 0x3FFF
	v.codeReg = code

	if code == 0 {
		v.readBuf = v.VRAM[v.addrReg]
		v.addrReg = (v.addrReg + 1) & 0x3FFF
	}
}

func (v *VDP) ReadControl() byte {
	s := byte(0)
	if v.framePending {
		s |= 0x80
	}
	if v.sprOverflow {
		s |= 0x40
	}
	if v.sprCollision {
		s |= 0x20
	}
	v.framePending = false
	v.vblankPending = false
	v.sprOverflow = false
	v.sprCollision = false
	v.linePending = false
	v.secondByte = false
	return s
}

func (v *VDP) WriteData(val byte) {
	v.secondByte = false
	switch v.codeReg {
	case 0, 1, 2:
		v.VRAM[v.addrReg] = val
	case 3:
		v.writeCRAM(v.addrReg, val)
	}
	v.readBuf = val
	v.addrReg = (v.addrReg + 1) & 0x3FFF
}

func (v *VDP) ReadData() byte {
	v.secondByte = false
	val := v.readBuf
	v.readBuf = v.VRAM[v.addrReg]
	v.addrReg = (v.addrReg + 1) & 0x3FFF
	return val
}

func (v *VDP) writeCRAM(addr uint16, val byte) {
	if v.IsGameGear {
		cramAddr := addr & 0x3F
		if cramAddr&1 == 0 {
			GGColorLatch = val
		} else {
			wordAddr := (cramAddr - 1) & 0x3E
			v.CRAM[wordAddr] = GGColorLatch
			v.CRAM[wordAddr+1] = val & 0x0F
		}
	} else {
		v.CRAM[addr&0x1F] = val
	}
}

var GGColorLatch byte

func (v *VDP) setReg(num byte, val byte) {
	if num > 10 {
		return
	}
	v.reg[num] = val
	switch num {
	case 0:
		v.disableVScroll = val&0x80 != 0
		v.disableHScroll = val&0x40 != 0
		v.maskCol0 = val&0x20 != 0
		v.lineIntEnable = val&0x10 != 0
		v.shiftLeft = val&0x08 != 0
		v.mode4 = val&0x04 != 0
		v.updateHeight()
	case 1:
		v.displayEnable = val&0x40 != 0
		v.frameIntEnable = val&0x20 != 0
		v.largeSprites = val&0x02 != 0
		v.stretchedSprites = val&0x01 != 0
		v.updateHeight()
	case 2:
		v.nameTableAddr = ((uint16(val) >> 1) & 0x07) << 11
	case 5:
		v.spriteAttrTableAddr = ((uint16(val) >> 1) & 0x3F) << 8
	case 6:
		v.spriteTileTableAddr = ((uint16(val) >> 2) & 0x01) << 13
	case 7:
		v.backdropIdx = val & 0x0F
		v.backdropColor = 16 + v.backdropIdx
	case 8:
		v.scrollX = int16(val)
	case 9:
		v.scrollY = int16(val)
	case 10:
		v.lineIntReload = val
		v.lineIntCounter = val
	}
}

func (v *VDP) updateHeight() {
	m4 := v.reg[0]&0x04 != 0
	m2 := v.reg[0]&0x02 != 0
	m1 := v.reg[1]&0x10 != 0
	m3 := v.reg[1]&0x08 != 0

	if m4 && m2 && !m1 && !m3 {
		v.frameHeight = ScreenHeightNTSC
	} else if m4 && !m2 && m1 {
		v.frameHeight = 224
	} else if m4 && !m2 && !m1 {
		v.frameHeight = 240
	} else {
		v.frameHeight = ScreenHeightNTSC
	}
}

func (v *VDP) Tick(cpuCycles uint32) {
	for i := uint32(0); i < cpuCycles; i++ {
		v.dot++
		if v.dot >= HblankCycles {
			v.endScanline()
		}
	}
}

func (v *VDP) endScanline() {
	v.dot = 0

	if v.scanline < v.frameHeight && v.displayEnable {
		v.renderScanline(v.scanline)
	}

	v.scanline++
	v.vCounter++

	if v.lineIntEnable {
		if v.lineIntCounter == 0 {
			v.lineIntCounter = v.lineIntReload
			v.linePending = true
		} else {
			v.lineIntCounter--
		}
	}

	if v.scanline == v.frameHeight {
		v.framePending = true
		if v.frameIntEnable {
			v.vblankPending = true
		}
	}

	if v.scanline >= v.totalLines {
		v.scanline = 0
		v.vCounter = 0
		v.frameReady = true
	}
}

func (v *VDP) FrameReady() bool {
	r := v.frameReady
	v.frameReady = false
	return r
}

func (v *VDP) Framebuffer() []byte {
	return v.framebuffer[:int(v.frameHeight)*ScreenWidth*4]
}

func (v *VDP) VCounter() byte   { return v.vCounter }
func (v *VDP) HCounter() byte   { return v.hCounter }
func (v *VDP) IntPending() bool { return v.vblankPending || v.linePending }

func (v *VDP) renderScanline(y uint16) {
	if !v.displayEnable {
		off := int(y) * ScreenWidth * 4
		for x := 0; x < ScreenWidth*4; x++ {
			v.framebuffer[off+x] = 0
		}
		return
	}

	for x := uint16(0); x < ScreenWidth; x++ {
		c := v.backdropColor
		r, g, b := smsColorToRGB(v.CRAM[c])
		v.setPixel(x, y, r, g, b)
	}

	scrollY := v.scrollY
	fh := int16(v.frameHeight)
	for scrollY < 0 {
		scrollY += fh
	}
	effectiveY := (int16(y) + scrollY) % fh
	if effectiveY < 0 {
		effectiveY += fh
	}
	row := uint16(effectiveY) / 8
	py := uint16(effectiveY) % 8

	scrollX := v.scrollX
	for scrollX < 0 {
		scrollX += 256
	}

	for col := uint16(0); col < 32; col++ {
		pat, hFlip, vFlip, pri, pal := v.getNameTableEntry(col, row)
		tileData := v.getTileRow(pat, py, vFlip)
		xStart := (ScreenWidth - uint16(scrollX)%256 + col*8) % ScreenWidth

		for px := uint16(0); px < 8; px++ {
			sx := xStart + px
			if sx >= ScreenWidth {
				sx -= ScreenWidth
			}
			if v.maskCol0 && sx < 8 {
				c := 16 + v.backdropIdx
				r, g, b := smsColorToRGB(v.CRAM[c])
				v.setPixel(sx, y, r, g, b)
				continue
			}
			bit := px
			if hFlip {
				bit = 7 - px
			}
			idx := ((tileData[0] >> bit) & 1) | (((tileData[1] >> bit) & 1) << 1) |
				(((tileData[2] >> bit) & 1) << 2) | (((tileData[3] >> bit) & 1) << 3)
			if idx == 0 && pri == 0 {
				continue
			}
			c := uint16(pal)*16 + uint16(idx)
			r, g, b := smsColorToRGB(v.CRAM[c])
			v.setPixel(sx, y, r, g, b)
		}
	}

	v.renderSprites(y)
}

func (v *VDP) getNameTableEntry(col, row uint16) (pat uint16, hFlip, vFlip bool, pri byte, pal byte) {
	addr := v.nameTableAddr + row*64 + col*2
	lo := uint16(v.VRAM[addr])
	hi := uint16(v.VRAM[addr+1])
	pat = lo | ((hi & 0x03) << 8)
	hFlip = hi&0x04 != 0
	vFlip = hi&0x08 != 0
	pri = byte((hi >> 5) & 1)
	pal = byte((hi >> 3) & 1)
	return
}

func (v *VDP) getTileRow(pat uint16, py uint16, vFlip bool) [4]byte {
	if vFlip {
		py = 7 - py
	}
	base := (pat * 32) & (VRAMSize - 1)
	return [4]byte{
		v.VRAM[(base+py)&(VRAMSize-1)],
		v.VRAM[(base+py+8)&(VRAMSize-1)],
		v.VRAM[(base+py+16)&(VRAMSize-1)],
		v.VRAM[(base+py+24)&(VRAMSize-1)],
	}
}

func (v *VDP) renderSprites(y uint16) {
	sprHeight := uint16(8)
	if v.largeSprites {
		sprHeight = 16
	}
	if v.stretchedSprites {
		sprHeight *= 2
	}

	type sp struct{ x, pat, sy uint16 }

	var list [8]sp
	n := 0

	for si := uint16(0); si < 64; si++ {
		sy := uint16(v.VRAM[v.spriteAttrTableAddr+si])
		if sy == 0xD0 {
			break
		}
		if sy >= 224 {
			sy -= 256
		}
		sy++

		if y < sy || y >= sy+sprHeight {
			continue
		}

		attrAddr := v.spriteAttrTableAddr + 0x80 + si*2
		sx := uint16(v.VRAM[attrAddr])
		pat := uint16(v.VRAM[attrAddr+1])

		if n < 8 {
			list[n] = sp{sx, pat, sy}
			n++
		} else {
			v.sprOverflow = true
			break
		}
	}

	for i := 0; i < n; i++ {
		sl := list[i]
		var sprPat uint16
		sprY := y - sl.sy
		if v.largeSprites && sprY >= 8 {
			sprPat = 1
			sprY -= 8
		}
		pat := sl.pat | sprPat
		tile := v.getTileRow(pat, sprY, false)
		for px := uint16(0); px < 8; px++ {
			sx := sl.x + px
			if v.stretchedSprites {
				sx = sl.x + px*2
			}
			if sx >= ScreenWidth {
				continue
			}
			bit := 7 - px
			idx := ((tile[0] >> bit) & 1) | (((tile[1] >> bit) & 1) << 1) |
				(((tile[2] >> bit) & 1) << 2) | (((tile[3] >> bit) & 1) << 3)
			if idx == 0 {
				continue
			}
			c := 16 + idx
			r, g, b := smsColorToRGB(v.CRAM[c])
			v.setPixel(sx, y, r, g, b)
		}
	}
}

func (v *VDP) setPixel(x, y uint16, r, g, b byte) {
	if x >= ScreenWidth || y >= v.frameHeight {
		return
	}
	off := int(y)*ScreenWidth*4 + int(x)*4
	v.framebuffer[off] = r
	v.framebuffer[off+1] = g
	v.framebuffer[off+2] = b
	v.framebuffer[off+3] = 255
}

func (v *VDP) Reg(i int) byte          { return v.reg[i] }
func (v *VDP) AddrReg() uint16         { return v.addrReg }
func (v *VDP) CodeReg() byte           { return v.codeReg }
func (v *VDP) SecondByte() bool        { return v.secondByte }
func (v *VDP) VRAMAt(addr uint16) byte { return v.VRAM[addr] }
func (v *VDP) CRAMAt(addr uint16) byte { return v.CRAM[addr] }
func (v *VDP) Scanline() uint16        { return v.scanline }
func (v *VDP) Dot() uint16             { return v.dot }

func smsColorToRGB(c byte) (r, g, b byte) {
	r = (c & 0x03) * 85
	g = ((c >> 2) & 0x03) * 85
	b = ((c >> 4) & 0x03) * 85
	return
}
