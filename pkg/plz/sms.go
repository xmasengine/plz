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

func colorForRune(r rune) (col byte, skip bool, next bool, err error) {
	switch r {
	case '\n':
		return 0, false, true, nil
	case ' ':
		return 0, true, true, nil
	case '.':
		return 0, false, false, nil
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return byte(r - '0'), false, false, nil
	case 'A', 'B', 'C', 'D', 'E', 'F':
		return byte(r - 'A' + 10), false, false, nil
	case 'X':
		return 15, false, false, nil
	default:
		return 0, false, false, Error{Message: "Unknown character in Tile, expected .X0123456789ABCDEF"}
	}
}

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
	tile := &SMSTile{}
	bounds := pimg.Bounds()
	// Have to use the bounds because subimages may not have a rectangle
	// that has a Min of (0,0).
	for y := bounds.Min.Y; y < bounds.Max.Y && y-bounds.Min.Y < tileHeight; y++ {
		for x := bounds.Min.X; x < bounds.Max.X && x-bounds.Min.X < tileWidth; x++ {
			col := pimg.ColorIndexAt(x, y)
			tile.SetPaletteIdAt(x-bounds.Min.X, y-bounds.Min.Y, SMSPaletteID(col))
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
	res := []*SMSTile{}
	rect := image.Rect(0, 0, tileWidth, tileHeight)
	for y := 0; y < pimg.Bounds().Dy(); y += rect.Bounds().Dy() {
		for x := 0; x < pimg.Bounds().Dx(); x += rect.Bounds().Dx() {
			shift := rect.Add(image.Pt(x, y))
			sub := pimg.SubImage(shift)
			tile, err := SMSLoadTileFromImage(sub)
			if err != nil {
				return nil, err
			}
			res = append(res, tile)
		}
	}
	return res, nil
}
