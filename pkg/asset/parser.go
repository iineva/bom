package asset

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/iineva/bom/pkg/bom"
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
	c.RenditionKeyTokens = make([]string, c.MaximumRenditionKeyTokenCount)
	for i := uint32(0); i < c.MaximumRenditionKeyTokenCount; i++ {
		t := RenditionAttributeType(0)
		if err := binary.Read(buf, binary.LittleEndian, &t); err != nil {
			return nil, err
		}
		c.RenditionKeyTokens[i] = t.String()
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
	a.bom.ReadTree("APPEARANCEKEYS", func(k, d []byte) error {
		name := string(k)
		value := uint16(0)
		if err := binary.Read(bytes.NewBuffer(d), binary.BigEndian, &value); err != nil {
			return err
		}
		keys[name] = value
		return nil
	})
	return keys, nil
}
