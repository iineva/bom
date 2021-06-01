package asset

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"os"
	"strings"

	"github.com/blacktop/go-lzfse"
	"github.com/iineva/bom/pkg/bom"
	"github.com/iineva/bom/pkg/helper"
)

// const typeMap = map[string]interface{}{
// "CARHEADER": CarHeader,
// "EXTENDED_METADATA": CarextendedMetadata,
// "KEYFORMAT": RenditionKeyFmt,
// "CARGLOBALS":
// "KEYFORMATWORKAROUND":
// "EXTERNAL_KEYS":

// // tree
// "FACETKEYS": Tree,
// "RENDITIONS": Tree,
// "APPEARANCEKEYS": Tree,
// "COLORS": Tree,
// "FONTS": Tree,
// "FONTSIZES": Tree,
// "GLYPHS": Tree,
// "BEZELS": Tree,
// "BITMAPKEYS": Tree,
// "ELEMENT_INFO": Tree,
// "PART_INFO": Tree,
// }

type AssetParser interface {
}

type asset struct {
	bom bom.BomParser
}

func New(b bom.BomParser) *asset {
	return &asset{bom: b}
}

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

func NewWithReadSeeker(r io.ReadSeeker) (*asset, error) {
	b := bom.New(r)
	if err := b.Parse(); err != nil {
		return nil, err
	}
	return &asset{bom: b}, nil
}

func (a *asset) read(name string, order binary.ByteOrder, p interface{}) error {
	d, err := a.bom.ReadBlock(name)
	if err != nil {
		return err
	}

	if err := binary.Read(bytes.NewBuffer(d), order, p); err != nil {
		return err
	}

	return nil
}

func (a *asset) CarHeader() (*CarHeader, error) {
	c := &CarHeader{}
	if err := a.read("CARHEADER", binary.LittleEndian, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (a *asset) KeyFormat() (*RenditionKeyFmt, error) {
	d, err := a.bom.ReadBlock("KEYFORMAT")
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(d)

	c := &RenditionKeyFmt{}
	if err := binary.Read(buf, binary.LittleEndian, &c.Tag); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &c.Version); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &c.MaximumRenditionKeyTokenCount); err != nil {
		return nil, err
	}
	// read key tokens
	c.RenditionKeyTokens = make([]RenditionAttributeType, c.MaximumRenditionKeyTokenCount)
	for i := uint32(0); i < c.MaximumRenditionKeyTokenCount; i++ {
		t := RenditionAttributeType(0)
		if err := binary.Read(buf, binary.LittleEndian, &t); err != nil {
			return nil, err
		}
		c.RenditionKeyTokens[i] = t
	}

	return c, nil
}

func (a *asset) ExtendedMetadata() (*CarextendedMetadata, error) {
	c := &CarextendedMetadata{}
	if err := a.read("EXTENDED_METADATA", binary.BigEndian, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (a *asset) AppearanceKeys() (map[string]uint16, error) {
	keys := map[string]uint16{}
	if err := a.bom.ReadTree("APPEARANCEKEYS", func(k *bytes.Buffer, d *bytes.Buffer) error {
		value := uint16(0)
		if err := binary.Read(d, binary.BigEndian, &value); err != nil {
			return err
		}
		keys[k.String()] = value
		return nil
	}); err != nil {
		return nil, err
	}
	return keys, nil
}

func (a *asset) FacetKeys() (map[string]map[RenditionAttributeType]uint16hex, error) {
	data := map[string]map[RenditionAttributeType]uint16hex{}
	if err := a.bom.ReadTree("FACETKEYS", func(k *bytes.Buffer, d *bytes.Buffer) error {
		attrs := map[RenditionAttributeType]uint16hex{}
		name := k.String()
		t := &Renditionkeytoken{}
		if err := binary.Read(d, binary.LittleEndian, &t.CursorHotSpot); err != nil {
			return err
		}
		if err := binary.Read(d, binary.LittleEndian, &t.NumberOfAttributes); err != nil {
			return err
		}
		t.Attributes = make([]RenditionAttribute, t.NumberOfAttributes)
		for i := uint16(0); i < t.NumberOfAttributes; i++ {
			a := RenditionAttribute{}
			if err := binary.Read(d, binary.LittleEndian, &a); err != nil {
				return err
			}
			t.Attributes[i] = a
			attrs[RenditionAttributeType(a.Name)] = a.Value
		}
		data[name] = attrs
		return nil
	}); err != nil {
		return nil, err
	}
	return data, nil
}

func (a *asset) Renditions() error {
	kf, err := a.KeyFormat()
	if err != nil {
		return err
	}
	i := 0
	if err := a.bom.ReadTree("RENDITIONS", func(k *bytes.Buffer, d *bytes.Buffer) error {
		attrs := map[RenditionAttributeType]uint16hex{}
		for i := 0; i < len(kf.RenditionKeyTokens); i++ {
			v := uint16hex(0)
			binary.Read(k, binary.LittleEndian, &v)
			attrs[kf.RenditionKeyTokens[i]] = v
		}

		c := csiheader{}
		if err := binary.Read(d, binary.LittleEndian, &c); err != nil {
			return err
		}

		// TODO: skip TLV for now
		tmp := make([]byte, c.Csibitmaplist.TvlLength)
		if _, err := d.Read(tmp); err != nil {
			return err
		}

		log.Printf("%s: %s: %s attrs: %+v TVL: %+v %v", c.Tag.String(), c.PixelFormat.String(), c.Csimetadata.Name.String(), attrs, c, len(tmp))
		// string value reverse
		switch strings.TrimSpace(string(helper.Reverse(c.PixelFormat[:]))) {
		case "DATA":
			// TODO:
		case "JPEG", "HEIF":
			// TODO:
		case "GA8", "RGB5", "RGBW", "GA16":
			// TODO:

		case "ARGB":
			// TODO:
			p := CUIThemePixelRendition{}
			if err := binary.Read(d, binary.LittleEndian, &p.Tag); err != nil {
				return err
			}
			if err := binary.Read(d, binary.LittleEndian, &p.Version); err != nil {
				return err
			}
			if err := binary.Read(d, binary.LittleEndian, &p.CompressionType); err != nil {
				return err
			}

			if p.CompressionType == kRenditionCompressionType_lzfse {

				var rawData []byte

				if p.Version != 3 {
					if err := binary.Read(d, binary.LittleEndian, &p.RawDataLength); err != nil {
						return err
					}
					p.RawData = make([]byte, p.RawDataLength)
					if _, err := d.Read(p.RawData); err != nil {
						return err
					}
					rawData = p.RawData
				} else {
					v3 := CUIThemePixelRenditionV3{}
					if err := binary.Read(d, binary.LittleEndian, &v3); err != nil {
						return err
					}
					rawData = d.Bytes()

					count := 0
					rawLen := uint32(0)
					if err := binary.Read(bytes.NewBuffer(rawData[4:]), binary.LittleEndian, &rawLen); err != nil {
						return err
					}
					for i, _ := range rawData {
						if i >= 3 {
							if rawData[i-1] == 120 && rawData[i-2] == 118 && rawData[i-3] == 98 {
								count++
								log.Print(count)
							}
						}
					}

				}

				// buf, _ := seekbuf.Open(d, seekbuf.MemoryMode)
				// defer buf.Close()
				decoded := lzfse.DecodeBuffer(rawData)
				fileName := fmt.Sprintf("%v-%v", i, c.Csimetadata.Name.String())
				// fileName = strings.Trim(fileName, string([]byte{0}))
				outf, _ := os.Create(fileName)
				i++

				rect := image.Rectangle{
					Min: image.Point{0, 0},
					Max: image.Point{X: int(c.Width) + 4, Y: int(c.Height)},
				}
				rgba := BGRA{image.RGBA{
					Pix:    decoded,
					Stride: rect.Dx() * 4,
					Rect:   rect,
				}}
				// aaa := helper.NewString8(string(decoded[:8]))
				// log.Print(aaa)
				if err := png.Encode(outf, &rgba); err != nil {
					log.Print(err)
				}
				// img, t, err := image.Decode(bytes.NewBuffer(d))
				// log.Printf("%+v, %v, %v", img, t, err)
				// _, err = io.Copy(outf, bytes.NewBuffer(d))
				// if err != nil {
				// log.Print(err.Error())
				// }
			}
			// a := uint32(0)

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
			// TODO: to image.Image
		case string([]byte{0, 0, 0, 0}):
			switch c.Csimetadata.Layout {
			case kRenditionLayoutType_Color:
				// TODO:
			case kRenditionLayoutType_MultisizeImage:
				// _CUIThemeMultisizeImageSetRendition
				// TODO:
				p := CUIThemeMultisizeImageSetRendition{}
				// l := d.Bytes()
				// log.Print(l)
				// return nil
				if err := binary.Read(d, binary.LittleEndian, &p); err != nil {
					return err
				}
				// if err := binary.Read(d, binary.LittleEndian, &p.Version); err != nil {
				// 	return err
				// }
				// if err := binary.Read(d, binary.LittleEndian, &p.CompressionType); err != nil {
				// 	return err
				// }
				// if err := binary.Read(d, binary.LittleEndian, &p.RawDataLength); err != nil {
				// 	return err
				// }
				log.Printf("%+v", p)
			}
		default:
			return fmt.Errorf("unknown rendition with pixel format: %v", c.PixelFormat.String())
		}

		// log.Printf("%+v", c)
		log.Print("")
		log.Print("")
		log.Print("")
		return nil
	}); err != nil {
		return err
	}
	return nil
}
