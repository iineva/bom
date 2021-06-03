package asset

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"io/ioutil"

	"github.com/blacktop/go-lzfse"
	"github.com/iineva/bom/pkg/mreader"
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
	if err := binary.Read(d, binary.LittleEndian, &p.RawDataLength); err != nil {
		return nil, err
	}

	rawData := mreader.New()

	// decode header
	switch p.Version {
	case 0, 2:
		buf := make([]byte, p.RawDataLength)
		if _, err := d.Read(buf); err != nil {
			return nil, err
		}
		r, err := umCompression(p.CompressionType, bytes.NewBuffer(buf))
		if err != nil {
			return nil, err
		}
		rawData.Add(r)
	case 1, 3:
		for i := 0; i < int(p.RawDataLength); i++ {
			v3 := &CUIThemePixelRenditionV3{}
			if err := binary.Read(d, binary.LittleEndian, v3); err != nil {
				return nil, err
			}
			buf := make([]byte, v3.RowDataLen)
			err := binary.Read(d, binary.LittleEndian, buf)
			if err != nil {
				return nil, err
			}
			r, err := umCompression(p.CompressionType, bytes.NewBuffer(buf))
			if err != nil {
				return nil, err
			}
			rawData.Add(r)
		}
	default:
		return nil, fmt.Errorf("unsupport version: %v", p.Version)
	}

	defer rawData.Close()
	return decodeImage(format, int(c.Width), int(c.Height), rawData)
}

func umCompression(t RenditionCompressionType, r io.Reader) (decoded io.ReadCloser, err error) {
	// upcompression raw data
	switch t {
	case kRenditionCompressionType_zip:
		return gzip.NewReader(r)
	case kRenditionCompressionType_lzfse:
		d, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		decoded = io.NopCloser(bytes.NewBuffer(lzfse.DecodeBuffer(d)))
	case kRenditionCompressionType_uncompressed:
		decoded = io.NopCloser(r)
	// NOTE: do nothing
	// TODO
	// case kRenditionCompressionType_deepmap_2:
	default:
		return nil, fmt.Errorf("unsupport compression type: %v", t)
	}
	return
}

func decodeImage(format string, width, height int, r io.Reader) (image.Image, error) {
	offset := 0
	rawData, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
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
