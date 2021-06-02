package asset

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"io/ioutil"

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

// gray alpha 8 bit
type GA8 struct {
	Pix    []uint8
	Stride int
	Rect   image.Rectangle
}

func (p *GA8) ColorModel() color.Model { return color.RGBAModel }

func (p *GA8) Bounds() image.Rectangle { return p.Rect }

func (p *GA8) At(x, y int) color.Color {
	return p.GA8At(x, y)
}

func (p *GA8) GA8At(x, y int) color.RGBA {
	if !(image.Point{x, y}.In(p.Rect)) {
		return color.RGBA{}
	}
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+2 : i+2] // Small cap improves performance, see https://golang.org/issue/27857
	return color.RGBA{s[0], s[0], s[0], s[1]}
}

func (p *GA8) PixOffset(x, y int) int {
	return (y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*2
}

// format: "ARGB", "GA8", "RGB5", "RGBW", "GA16"
func (a *asset) decodeImage(format string, d io.Reader, c *csiheader) (image.Image, error) {
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

	var rawData []byte
	var err error

	width := c.Width
	height := c.Height

	// f := c.PixelFormat.String()
	// n := c.Csimetadata.Name.String()
	// log.Print(f, ",", n)

	// decode header
	switch p.Version {
	case 0, 2:
		if err := binary.Read(d, binary.LittleEndian, &p.RawDataLength); err != nil {
			return nil, err
		}
		p.RawData = make([]byte, p.RawDataLength)
		if _, err := d.Read(p.RawData); err != nil {
			return nil, err
		}
		rawData = p.RawData
	case 1, 3: // maybe version 2
		// NOTE: unknow this bolck, skip
		// TODO: handle this block
		v3 := CUIThemePixelRenditionV3{}
		if err := binary.Read(d, binary.LittleEndian, &v3); err != nil {
			return nil, err
		}

		height = v3.Height
		rawData, err = ioutil.ReadAll(d)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupport version: %v", p.Version)
	}

	// upcompression raw data
	switch p.CompressionType {
	case kRenditionCompressionType_lzfse:
		rawData = lzfse.DecodeBuffer(rawData)
	case kRenditionCompressionType_uncompressed:
		// NOTE: do nothing
	// TODO
	// case kRenditionCompressionType_deepmap_2:
	default:
		return nil, fmt.Errorf("unsupport compression type: %v", p.CompressionType)
	}

	offset := 0
	switch format {
	case "ARGB":
		if v := len(rawData) - int(width*height*4); v != 0 {
			offset = v / int(height*4)
		}
		if offset < 0 {
			return nil, errors.New("error image content")
		}
		rect := image.Rectangle{
			Min: image.Point{0, 0},
			Max: image.Point{
				X: int(width),
				Y: int(height),
			},
		}
		bgra := &BGRA{image.RGBA{
			Pix:    rawData,
			Stride: (rect.Dx() + offset) * 4,
			Rect:   rect,
		}}
		return bgra, nil
	case "GA8":

		if v := len(rawData) - int(width*height*2); v != 0 {
			offset = v / int(height*2)
		}
		if offset < 0 {
			return nil, errors.New("error image content")
		}

		rect := image.Rectangle{
			Min: image.Point{0, 0},
			Max: image.Point{
				X: int(width),
				Y: int(height),
			},
		}
		bgra := &GA8{
			Pix:    rawData,
			Stride: (rect.Dx() + offset) * 2,
			Rect:   rect,
		}
		return bgra, nil
	case "RGB5":
	case "RGBW":
	case "GA16":
	}
	return nil, fmt.Errorf("unsupport image format: %v", format)
}
