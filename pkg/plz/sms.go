package plz

import (
	"github.com/mrcook/smstilemap/sms"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/png"
	"os"
)

// SMS specific functionality.

type SMSTile = sms.Tile
type SMSPaletteID = sms.PaletteId

const tileWidth = 8
const tileHeight = 8

func SMSTileFromString(s string) (SMSTile, error) {
	res := SMSTile{}

	var x, y int
	for _, r := range s {
		col, skip, next, err := colorForRune(r)
		if err != nil {
			return res, err
		}
		if skip {
			continue
		}
		if next {
			if x > 0 {
				x = 0
				y++
				if y >= tileHeight {
					return res, nil
				}
			}
			continue
		}
		res.SetPaletteIdAt(x, y, SMSPaletteID(col))
		x++
		if x >= tileWidth {
			x = 0
			y++
			if y >= tileHeight { // Ignore extra characters, stop if we have all pixels.
				return res, nil
			}
		}
	}
	return res, nil
}

type PalettedImageWithSubImage interface {
	image.PalettedImage
	SubImage(r image.Rectangle) image.Image
}

func SMSLoadTileFromImage(img image.Image) (*SMSTile, error) {
	pimg, ok := img.(*image.Paletted)
	if !ok {
		return nil, Error{Message: "image has no palette"}
	}
	println("SMSLoadTileFromImage", pimg.Stride, pimg.PixOffset(0, 0))
	tile := &SMSTile{}
	for y := 0; y < pimg.Bounds().Dy() && y < tileHeight; y++ {
		for x := 0; x < pimg.Bounds().Dx() && x < tileWidth; x++ {
			col := pimg.ColorIndexAt(x, y)
			tile.SetPaletteIdAt(x, y, SMSPaletteID(col))
		}
	}
	return tile, nil
}

func IsBitmapEmpty(bitmap image.PalettedImage, cw, ch, cx, cy int) bool {
	for y := cy; y < cy+ch; y++ {
		for x := cx; x < cx+cw; x++ {
			idx := bitmap.ColorIndexAt(x, y)
			if idx != 0 {
				return false
			}
		}
	}
	return true
}

func SMSLoadTiles(fn string) ([]*SMSTile, error) {
	rd, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer rd.Close()
	img, _, err := image.Decode(rd)
	if err != nil {
		return nil, err
	}
	pimg, ok := img.(*image.Paletted)
	if !ok {
		return nil, Error{Message: "can only load paletted images with sub images"}
	}
	cm := pimg.ColorModel()
	palette, ok := cm.(color.Palette)
	if !ok {
		return nil, Error{Message: "Cannot get palette"}
	}
	if len(palette) > 16 {
		return nil, Error{Message: "Too many palette entries, can only have 16"}
	}
	if len(palette) < 1 {
		return nil, Error{Message: "Too few palette entries."}
	}
	println("SMSLoadTiles palette, stride", len(palette), pimg.Stride)
	res := []*SMSTile{}
	rect := image.Rect(0, 0, tileWidth, tileHeight)
	for y := 0; y < pimg.Bounds().Dy(); y += rect.Bounds().Dy() {
		for x := 0; x < pimg.Bounds().Dx(); x += rect.Bounds().Dx() {
			shift := rect.Add(image.Pt(x, y))
			println("SMSLoadTiles", shift.Min.X, shift.Min.Y,
				shift.Max.X, shift.Max.Y)

			sub := pimg.SubImage(shift)
			if IsBitmapEmpty(sub.(image.PalettedImage), 8, 8, 0, 0) {
				println("sub is empty")
			}

			tile, err := SMSLoadTileFromImage(sub)
			if err != nil {
				return nil, err
			}
			res = append(res, tile)
		}
	}
	if IsBitmapEmpty(pimg, 8, 8, 8, 8) {
		println("main is empty, should not be so")
	}
	return res, nil
}
