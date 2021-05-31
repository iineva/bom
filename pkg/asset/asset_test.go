package asset

import (
	"os"
	"reflect"
	"testing"

	"github.com/iineva/bom/pkg/helper"
)

func TestAsset(t *testing.T) {

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
	if c, err := b.KeyFormat(); err != nil {
		t.Fatal(err)
	} else {
		tc := &RenditionKeyFmt{
			Tag:                           helper.NewString4("tmfk"),
			Version:                       0,
			MaximumRenditionKeyTokenCount: 7,
			RenditionKeyTokens:            []string{"Scale", "Idiom", "Subtype", "Dimension 2", "Identifier", "Element", "Part"},
		}
		if !reflect.DeepEqual(c, tc) {
			t.Fail()
		}
	}

	if c, err := b.AppearanceKeys(); err != nil {
		t.Fatal(err)
	} else {
		if !reflect.DeepEqual(map[string]uint16{"UIAppearanceAny": 0}, c) {
			t.Fail()
		}
	}
}
