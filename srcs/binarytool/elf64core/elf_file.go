// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package elf64core

import (
	"bufio"
	"encoding/binary"
	"os"
)

type ELF64File struct {
	Header           *ELF64Header
	SectionsTable    SectionsTable
	SegmentsTable    ProgramTable
	DynamicTable     DynamicTable
	SymbolsTables    []SymbolsTables
	RelaTables       []RelaTables
	NotesTables      []NotesTables
	FunctionsTables  []FunctionTables
	Raw              []byte
	IndexSections    map[string]int
	MapFctAddrName   map[uint64]string
	Name             string
	Endianness       binary.ByteOrder
	TextSectionIndex []int // slice since we can have several
}

func (elfFile *ELF64File) ReadElfBinaryFile(filename string) error {
	file, err := os.Open(filename)

	if err != nil {
		return err
	}
	defer file.Close()

	stats, statsErr := file.Stat()
	if statsErr != nil {
		return statsErr
	}

	var size = stats.Size()
	elfFile.Raw = make([]byte, size)

	buf := bufio.NewReader(file)
	_, err = buf.Read(elfFile.Raw)

	return err
}

func (elfFile *ELF64File) ParseAll(path, name string) error {

	elfFile.Name = name

	if err := elfFile.ReadElfBinaryFile(path + name); err != nil {
		return err
	}

	if err := elfFile.ParseElfHeader(); err != nil {
		return err
	}

	if err := elfFile.ParseSectionHeaders(); err != nil {
		return err
	}

	if err := elfFile.ParseSections(); err != nil {
		return err
	}

	if err := elfFile.ParseProgramHeaders(); err != nil {
		return err
	}

	if err := elfFile.parseFunctions(); err != nil {
		return err
	}

	return nil
}
