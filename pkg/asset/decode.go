package asset

import (
	"encoding/binary"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"log"
	"strings"

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

	if err := binary.Read(d, order, p); err != nil {
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
	buf, err := a.bom.ReadBlock("KEYFORMAT")
	if err != nil {
		return nil, err
	}

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
	if err := a.bom.ReadTree("APPEARANCEKEYS", func(k io.Reader, d io.Reader) error {
		value := uint16(0)
		if err := binary.Read(d, binary.BigEndian, &value); err != nil {
			return err
		}
		key, err := ioutil.ReadAll(k)
		if err != nil {
			return err
		}
		keys[string(key)] = value
		return nil
	}); err != nil {
		return nil, err
	}
	return keys, nil
}

func (a *asset) FacetKeys() (map[string]RenditionAttrs, error) {
	data := map[string]RenditionAttrs{}
	if err := a.bom.ReadTree("FACETKEYS", func(k io.Reader, d io.Reader) error {
		attrs := map[RenditionAttributeType]uint16hex{}
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
		name, err := ioutil.ReadAll(k)
		if err != nil {
			return err
		}
		data[string(name)] = attrs
		return nil
	}); err != nil {
		return nil, err
	}
	return data, nil
}

func (a *asset) BitmapKeys() error {
	if err := a.bom.ReadTree("BITMAPKEYS", func(k io.Reader, d io.Reader) error {
		// TODO: handle bitmapKeys
		kk, err := ioutil.ReadAll(k)
		if err != nil {
			return err
		}
		dd, err := ioutil.ReadAll(d)
		if err != nil {
			return err
		}
		log.Printf("%v: %v", kk, dd)
		return nil
	}); err != nil {
		return err
	}
	return nil
}

type RenditionAttrs map[RenditionAttributeType]uint16hex

type RenditionType int

const (
	RenditionTypeImage = RenditionType(0)
	RenditionTypeData  = RenditionType(1)
	RenditionTypeColor = RenditionType(3)
)

type RenditionCallback struct {
	Attrs RenditionAttrs
	Type  RenditionType
	Err   error
	Image image.Image
	Name  string
}

func (a *asset) Renditions(loop func(cb *RenditionCallback) (stop bool)) error {
	kf, err := a.KeyFormat()
	if err != nil {
		return err
	}

	if err := a.bom.ReadTree("RENDITIONS", func(k io.Reader, d io.Reader) error {
		attrs := RenditionAttrs{}
		for i := 0; i < len(kf.RenditionKeyTokens); i++ {
			v := uint16hex(0)
			binary.Read(k, binary.LittleEndian, &v)
			attrs[kf.RenditionKeyTokens[i]] = v
		}

		c := &csiheader{}
		if err := binary.Read(d, binary.LittleEndian, c); err != nil {
			return err
		}

		// TODO: skip TLV for now
		tmp := make([]byte, c.Csibitmaplist.TvlLength)
		if _, err := d.Read(tmp); err != nil {
			return err
		}

		log.Printf("%s: %s: %s attrs: %+v TVL: %+v %v", c.Tag.String(), c.PixelFormat.String(), c.Csimetadata.Name.String(), attrs, c, len(tmp))
		// string value reverse
		format := strings.TrimSpace(string(helper.Reverse(c.PixelFormat[:])))
		switch format {
		case "DATA":
			// TODO:
			log.Print("TODO: handle DATA")
		case "JPEG", "HEIF":
			// TODO:
			log.Print("TODO: handle JPEG")
		case "ARGB", "GA8", "RGB5", "RGBW", "GA16":
			// TODO:
			cb := &RenditionCallback{
				Attrs: attrs,
				Type:  RenditionTypeImage,
				Name:  c.Csimetadata.Name.String(),
			}

			img, err := a.decodeImage(format, d, c)
			if err != nil {
				cb.Err = err
				stop := loop(cb)
				if stop {
					return err
				}
			}
			cb.Image = img
			stop := loop(cb)
			if stop {
				return nil
			}
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

		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (a *asset) ImageWalke(func(name string, img image.Image) (end bool)) error {
	c, err := a.FacetKeys()
	if err != nil {
		return err
	}
	for k, v := range c {
		log.Printf("%v: %v", k, v)
	}
	return nil
}

func (a *asset) GetImage(nam string) (image.Image, error) {

	return nil, nil
}
