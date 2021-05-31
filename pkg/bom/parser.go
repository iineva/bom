package bom

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/iineva/bom/pkg/helper"
)

type BomParser interface {
	// parse bom headers
	Parse() error

	// read all block names
	BlockNames() []string

	// read named block
	ReadBlock(name string) ([]byte, error)

	// read named tree block
	ReadTree(name string, entry func(k []byte, d []byte) error) error
}

type bom struct {
	r io.ReadSeeker

	header     *Header
	blockTable *BlockTable
	vars       []Var
}

var _ BomParser = (*bom)(nil)

var (
	ErrBlockLengthZero = errors.New("block length is zero")
	ErrNameNotMatch    = errors.New("name not match")
)

func New(r io.ReadSeeker) BomParser {
	return &bom{r: r}
}

func (b *bom) Parse() error {

	f := b.r

	header := &Header{}
	err := binary.Read(f, binary.BigEndian, header)
	if err != nil {
		log.Printf("read header error: %v", err)
		return err
	}
	if HeaderMagic != header.Magic.String() {
		err := fmt.Errorf("header magic not match header.Magic[8] = '%s'", header.Magic.String())
		log.Print(err)
		return err
	}
	b.header = header

	// blockTable
	f.Seek(int64(b.header.IndexOffset), 0)
	blockTable := &BlockTable{}
	// read table block length
	binary.Read(f, binary.BigEndian, &blockTable.NumberOfBlockTablePointers)
	// read table block pointers
	blockTable.BlockPointers = make([]*Pointer, blockTable.NumberOfBlockTablePointers)
	for i := 0; i < int(blockTable.NumberOfBlockTablePointers); i++ {
		p := &Pointer{}
		if err := binary.Read(f, binary.BigEndian, p); err != nil {
			return err
		}
		// First entry must always be a null entry
		if i == 0 && (p.Address != 0 || p.Length != 0) {
			return errors.New("first entry not be a null entry")
		}
		blockTable.BlockPointers[i] = p
	}
	b.blockTable = blockTable

	// read vars
	f.Seek(int64(header.VarsOffset), 0)
	vars := &Vars{}
	binary.Read(f, binary.BigEndian, &vars.Count)
	vars.List = make([]Var, vars.Count)
	for i := 0; i < int(vars.Count); i++ {
		v := Var{}
		binary.Read(f, binary.BigEndian, &v.Index)
		binary.Read(f, binary.BigEndian, &v.Length)

		// parse name
		name, err := helper.ReadString(f, int(v.Length))
		if err != nil {
			log.Printf("read var name error: %v", err)
			return err
		}
		v.Name = name

		vars.List[i] = v
	}
	b.vars = vars.List
	// log.Printf("vars: %+v", vars)

	return nil
}

// Block: get block with name
func (b *bom) ReadBlock(name string) ([]byte, error) {
	for _, v := range b.vars {

		if v.Name != name {
			continue
		}

		b, err := b.readBlock(v.Index)
		if err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	}
	return nil, ErrNameNotMatch
}

func (b *bom) BlockNames() []string {
	names := make([]string, len(b.vars))
	for i, v := range b.vars {
		names[i] = v.Name
	}
	return names
}

func (b *bom) readBlock(index uint32) (*bytes.Buffer, error) {
	p := b.blockTable.BlockPointers[index]
	if _, err := b.r.Seek(int64(p.Address), 0); err != nil {
		return nil, err
	}

	if p.Length == 0 {
		return nil, ErrBlockLengthZero
	}

	buf := &bytes.Buffer{}
	io.CopyN(buf, b.r, int64(p.Length))
	return buf, nil
}

// Block: get tree block with name
func (b *bom) ReadTree(name string, entry func(k []byte, d []byte) error) error {
	for _, v := range b.vars {

		if v.Name != name {
			continue
		}

		p := b.blockTable.BlockPointers[v.Index]
		if _, err := b.r.Seek(int64(p.Address), 0); err != nil {
			return err
		}

		if p.Length == 0 {
			return ErrBlockLengthZero
		}

		tree := TreeEntry{}
		binary.Read(b.r, binary.BigEndian, &tree)

		buf, err := b.readBlock(tree.Index)
		if err != nil {
			return err
		}

		ps := &Tree{}
		binary.Read(buf, binary.BigEndian, &ps.IsLeaf)
		binary.Read(buf, binary.BigEndian, &ps.Count)
		binary.Read(buf, binary.BigEndian, &ps.Forward)
		binary.Read(buf, binary.BigEndian, &ps.Backward)
		ps.List = make([]TreeIndex, ps.Count)
		for i := uint16(0); i < ps.Count; i++ {
			pi := TreeIndex{}
			binary.Read(buf, binary.BigEndian, &pi)
			ps.List[i] = pi

			// get key and data
			kbuf, err := b.readBlock(pi.KeyIndex)
			if err != nil {
				return err
			}
			dbuf, err := b.readBlock(pi.ValueIndex)
			if err != nil {
				return err
			}
			// loop callback entry
			if err := entry(kbuf.Bytes(), dbuf.Bytes()); err != nil {
				return err
			}
		}
		return nil
	}

	return ErrNameNotMatch
}
