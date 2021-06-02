package asset

import (
	"fmt"
	"image/png"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/iineva/bom/pkg/helper"
)

func TestAsset(t *testing.T) {

	// const fileName = "../bom/test_data/YouTube.car"
	// const fileName = "../bom/test_data/Instagram.car"
	const fileName = "../bom/test_data/Assets.car"
	f, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	b, err := NewWithReadSeeker(f)
	if err != nil {
		t.Fatal(err)
	}

	// name: BITMAPKEYS
	if err := b.BitmapKeys(); err != nil {
		t.Fatal(err)
	}

	// name: 'RENDITIONS'
	ri := 0
	if err := b.Renditions(func(cb *RenditionCallback) (stop bool) {
		if cb.Type == RenditionTypeImage && cb.Err == nil {
			fileName := fmt.Sprintf("%v-%v", ri, cb.Name)
			outf, _ := os.Create(fileName)
			defer outf.Close()
			if err := png.Encode(outf, cb.Image); err != nil {
				log.Print(err)
			}
		}
		log.Printf("%+v", cb)
		ri++
		return false
	}); err != nil {
		t.Fatal(err)
	}

	// name: 'CARHEADER'
	c, err := b.CarHeader()
	if err != nil {
		t.Fatal(err)
	} else {
		tc := &CarHeader{
			Tag:                helper.NewString4("RATC"),
			CoreuiVersion:      691,
			StorageVersion:     17,
			StorageTimestamp:   0,
			RenditionCount:     7,
			MainVersionString:  helper.NewString128("@(#)PROGRAM:CoreUI  PROJECT:CoreUI-691.2\n"),
			VersionString:      helper.NewString256("Xcode 12.5 (12E262) via IBCocoaTouchImageCatalogTool"),
			UUID:               helper.NewString16(""),
			AssociatedChecksum: 0,
			SchemaVersion:      2,
			ColorSpaceID:       1,
			KeySemantics:       2,
		}
		if !reflect.DeepEqual(c, tc) {
			t.Fail()
		}
	}

	// name: 'EXTENDED_METADATA'
	if c, err := b.ExtendedMetadata(); err != nil {
		t.Fatal(err)
	} else {
		tc := &CarextendedMetadata{
			Tag:                       helper.NewString4("META"),
			ThinningArguments:         helper.NewString256(""),
			DeploymentPlatformVersion: helper.NewString256("13.1"),
			DeploymentPlatform:        helper.NewString256("ios"),
			AuthoringTool:             helper.NewString256("@(#)PROGRAM:CoreThemeDefinition  PROJECT:CoreThemeDefinition-491\n"),
		}
		if !reflect.DeepEqual(c, tc) {
			t.Fail()
		}
	}

	// name: 'KEYFORMAT'
	if c, err := b.KeyFormat(); err != nil {
		t.Fatal(err)
	} else {
		tc := &RenditionKeyFmt{
			Tag:                           helper.NewString4("tmfk"),
			Version:                       0,
			MaximumRenditionKeyTokenCount: 7,
			RenditionKeyTokens:            []RenditionAttributeType{12, 15, 16, 9, 17, 1, 2},
		}
		if !reflect.DeepEqual(c, tc) {
			t.Fail()
		}
	}

	// name: 'APPEARANCEKEYS'
	if c, err := b.AppearanceKeys(); err != nil {
		t.Fatal(err)
	} else {
		if !reflect.DeepEqual(map[string]uint16{"UIAppearanceAny": 0}, c) {
			t.Fail()
		}
	}

	// name: 'FACETKEYS'
	if c, err := b.FacetKeys(); err != nil {
		t.Fatal(err)
	} else {
		tc := map[string]map[RenditionAttributeType]uint16hex{
			"AppIcon": {
				kRenditionAttributeType_Element:    0x0055,
				kRenditionAttributeType_Part:       0x00DC,
				kRenditionAttributeType_Identifier: 0x1AC1,
			},
			"test": {
				kRenditionAttributeType_Element:    0x0055,
				kRenditionAttributeType_Part:       0x00B5,
				kRenditionAttributeType_Identifier: 0x41A3,
			},
			"test2": {
				kRenditionAttributeType_Element:    0x0055,
				kRenditionAttributeType_Part:       0x00B5,
				kRenditionAttributeType_Identifier: 0x684F,
			},
			"test3": {
				kRenditionAttributeType_Element:    0x0055,
				kRenditionAttributeType_Part:       0x00B5,
				kRenditionAttributeType_Identifier: 0xF4B0,
			},
		}
		if !reflect.DeepEqual(tc, c) {
			t.Fail()
		}
	}

	// TODO:
	// name: 'BITMAPKEYS'

}
