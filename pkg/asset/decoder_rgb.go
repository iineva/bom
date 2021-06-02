package asset

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"
	"io/ioutil"
	"log"

	"github.com/blacktop/go-lzfse"
)

// BGRA to RGBA
type BGRA struct {
	image.RGBA
}

func (p *BGRA) RGBAAt(x, y int) color.RGBA {
	c := p.RGBA.RGBAAt(x, y)
	return color.RGBA{R: c.B, G: c.G, B: c.R, A: c.A}
}

func (p *BGRA) At(x, y int) color.Color {
	return p.RGBAAt(x, y)
}

func (p *BGRA) SubImage(r image.Rectangle) image.Image {
	c := p.RGBA.SubImage(r).(*image.RGBA)
	return &BGRA{*c}
}

// "ARGB", "GA8", "RGB5", "RGBW", "GA16"
func (a *asset) decodeImage(d io.Reader, c *csiheader) (image.Image, error) {
	p := &CUIThemePixelRendition{}
	if err := binary.Read(d, binary.LittleEndian, &p.Tag); err != nil {
		return nil, err
	}
	if err := binary.Read(d, binary.LittleEndian, &p.Version); err != nil {
		return nil, err
	}
	if err := binary.Read(d, binary.LittleEndian, &p.CompressionType); err != nil {
		return nil, err
	}

	switch p.CompressionType {
	case kRenditionCompressionType_lzfse:
		return a.decode_lzfse(d, c, p)
	case kRenditionCompressionType_deepmap_2:
		// TODO
	}

	// /*
	// migic := helper.String4{}
	// if err := binary.Read(d, binary.LittleEndian, &migic); err != nil {
	// 	return err
	// }

	// rawLen := uint32(0)
	// if err := binary.Read(d, binary.LittleEndian, &rawLen); err != nil {
	// 	return err
	// }
	// log.Printf("%+v %+v", v3, l)
	// */

	// l := d.Bytes()
	// log.Printf("byte length: %v", len(l))
	// }
	log.Printf("%+v %+v", p, a)

	return nil, errors.New("decoder not support")
}

func (a *asset) decode_lzfse(d io.Reader, c *csiheader, p *CUIThemePixelRendition) (image.Image, error) {
	var rawData []byte
	var err error

	width := c.Width
	height := c.Height

	if p.Version != 3 {
		if err := binary.Read(d, binary.LittleEndian, &p.RawDataLength); err != nil {
			return nil, err
		}
		p.RawData = make([]byte, p.RawDataLength)
		if _, err := d.Read(p.RawData); err != nil {
			return nil, err
		}
		rawData = p.RawData
	} else {
		// NOTE: unknow this bolck, skip
		// TODO: handl this block
		v3 := CUIThemePixelRenditionV3{}
		if err := binary.Read(d, binary.LittleEndian, &v3); err != nil {
			return nil, err
		}

		height = v3.Height
		rawData, err = ioutil.ReadAll(d)
		if err != nil {
			return nil, err
		}
	}

	decoded := lzfse.DecodeBuffer(rawData)
	rect := image.Rectangle{
		Min: image.Point{0, 0},
		Max: image.Point{
			X: int(width) + 4, // I dont know why, but +4 will work correct
			Y: int(height),
		},
	}
	bgra := &BGRA{image.RGBA{
		Pix:    decoded,
		Stride: rect.Dx() * 4,
		Rect:   rect,
	}}
	// return bgra, nil
	// befor return, strip +4 pix
	return bgra.SubImage(image.Rectangle{
		Min: image.Point{0, 0},
		Max: image.Point{
			X: int(width),
			Y: int(height),
		},
	}), nil
}
