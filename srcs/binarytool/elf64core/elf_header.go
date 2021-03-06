// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package elf64core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"
)

const (
	identLength  = 16
	littleEndian = 1
)

type ELF64Header struct {
	Ident                  [identLength]byte
	Type                   uint16
	Machine                uint16
	Version                uint32
	EntryPoint             uint64
	ProgramHeaderOffset    uint64
	SectionHeaderOffset    uint64
	Flags                  uint32
	HeaderSize             uint16
	ProgramHeaderEntrySize uint16
	ProgramHeaderEntries   uint16
	SectionHeaderEntrySize uint16
	SectionHeaderEntries   uint16
	SectionNamesTable      uint16
}

func (elfFile *ELF64File) ParseElfHeader() error {

	elfFile.Header = new(ELF64Header)

	// Check the size
	if len(elfFile.Raw) < identLength {
		return errors.New("invalid size")
	}

	// Check the magic number
	if elfFile.Raw[0] != 0x7f && elfFile.Raw[1] != 0x45 &&
		elfFile.Raw[2] != 0x4c && elfFile.Raw[3] != 0x46 {
		return errors.New("invalid ELF file")
	}

	// Check the type
	if elfFile.Raw[4] != 0x02 {
		return errors.New("elf32 is not supported")
	}

	if elfFile.Raw[5] == littleEndian {
		elfFile.Endianness = binary.LittleEndian
	} else {
		elfFile.Endianness = binary.BigEndian
	}

	data := bytes.NewReader(elfFile.Raw)
	err := binary.Read(data, elfFile.Endianness, elfFile.Header)
	if err != nil {
		return err
	}

	return nil
}

func (elfHeader ELF64Header) DisplayHeader() {

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Println("-----------------------------------------------------------------------")
	_, _ = fmt.Fprintf(w, "Magic Header:\t")
	for _, hex := range elfHeader.Ident {
		_, _ = fmt.Fprintf(w, "%.2x ", hex)
	}

	_, _ = fmt.Fprintf(w, "\nType:\t%x\n", elfHeader.Type)
	_, _ = fmt.Fprintf(w, "Machine:\t%d\n", elfHeader.Machine)
	_, _ = fmt.Fprintf(w, "Version:\t0x%x\n", elfHeader.Version)
	_, _ = fmt.Fprintf(w, "EntryPoint:\t0x%x\n", elfHeader.EntryPoint)
	_, _ = fmt.Fprintf(w, "ProgramHeaderOffset:\t%d\n", elfHeader.ProgramHeaderOffset)
	_, _ = fmt.Fprintf(w, "SectionHeaderOffset:\t%d\n", elfHeader.SectionHeaderOffset)
	_, _ = fmt.Fprintf(w, "Flags:\t0x%x\n", elfHeader.Flags)
	_, _ = fmt.Fprintf(w, "HeaderSize:\t%d\n", elfHeader.HeaderSize)
	_, _ = fmt.Fprintf(w, "ProgramHeaderEntrySize:\t%d\n", elfHeader.ProgramHeaderEntrySize)
	_, _ = fmt.Fprintf(w, "ProgramHeaderEntries:\t%d\n", elfHeader.ProgramHeaderEntries)
	_, _ = fmt.Fprintf(w, "SectionHeaderEntrySize:\t%d\n", elfHeader.SectionHeaderEntrySize)
	_, _ = fmt.Fprintf(w, "SectionHeaderEntries:\t%d\n", elfHeader.SectionHeaderEntries)
	_, _ = fmt.Fprintf(w, "SectionNamesTable:\t%d\n", elfHeader.SectionNamesTable)

	_ = w.Flush()
}
