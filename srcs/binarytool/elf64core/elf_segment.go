// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package elf64core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"text/tabwriter"
	u "tools/srcs/common"
)

type ProgramTable struct {
	nbEntries   int
	name        string
	dataProgram []*dataProgram
}

type dataProgram struct {
	sectionsPtr  []*DataSections
	elf64program ELF64ProgramHeader
}

type ELF64ProgramHeader struct {
	Type            uint32
	Flags           uint32
	FileOffset      uint64
	VirtualAddress  uint64
	PhysicalAddress uint64
	FileSize        uint64
	MemorySize      uint64
	Align           uint64
}

func (elfFile *ELF64File) addProgram(programs []ELF64ProgramHeader) error {

	elfFile.SegmentsTable = ProgramTable{}
	elfFile.SegmentsTable.nbEntries = len(programs)
	elfFile.SegmentsTable.dataProgram = make([]*dataProgram,
		elfFile.SegmentsTable.nbEntries)

	for i := range programs {
		elfFile.SegmentsTable.dataProgram[i] = &dataProgram{
			elf64program: programs[i],
		}
	}

	return nil
}

func (elfFile *ELF64File) mapSectionSegments() {
	for _, p := range elfFile.SegmentsTable.dataProgram {
		for _, s := range elfFile.SectionsTable.DataSect {
			if s.Elf64section.FileOffset >= p.elf64program.FileOffset &&
				s.Elf64section.FileOffset+s.Elf64section.Size <= p.elf64program.FileOffset+p.elf64program.FileSize {
				p.sectionsPtr = append(p.sectionsPtr, s)
			}
		}
	}
}

func (elfFile *ELF64File) ParseProgramHeaders() error {

	if elfFile.Header.ProgramHeaderEntries == 0 {
		return nil
	}

	offset := elfFile.Header.ProgramHeaderOffset
	if offset >= uint64(len(elfFile.Raw)) {
		return fmt.Errorf("invalid elf64section header offset: 0x%x", offset)
	}

	data := bytes.NewReader(elfFile.Raw[offset:])

	programs := make([]ELF64ProgramHeader, elfFile.Header.ProgramHeaderEntries)
	err := binary.Read(data, elfFile.Endianness, programs)
	if err != nil {
		return fmt.Errorf("failed reading elf64section header table: %s", err)
	}

	if err := elfFile.addProgram(programs); err != nil {
		return err
	}

	elfFile.mapSectionSegments()

	return nil
}

func (table *ProgramTable) DisplayProgramHeader() {

	if len(table.dataProgram) == 0 {
		fmt.Println("Program header is empty")
		return
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Println("-----------------------------------------------------------------------")
	_, _ = fmt.Fprintln(w, "Nr\tType\tOffset\tVirtAddr\tPhysAddr\tFileSiz\tMemSiz\tFlg\tAlign")
	for i, p := range table.dataProgram {
		_, _ = fmt.Fprintf(w, "[%.2d]\t%s\t%.6x\t%.6x\t%.6x\t%.6x\t%.6x\t%.6x\t0x%x\n", i,
			ptStrings[p.elf64program.Type],
			p.elf64program.FileOffset, p.elf64program.VirtualAddress,
			p.elf64program.PhysicalAddress, p.elf64program.FileSize,
			p.elf64program.MemorySize, p.elf64program.Flags, p.elf64program.Align)
	}
	_ = w.Flush()
}

func (table *ProgramTable) DisplaySegmentSectionMapping() {

	if len(table.dataProgram) == 0 {
		u.PrintWarning("Mapping between segments and sections is empty")
		return
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Println("-----------------------------------------------------------------------")
	for i, p := range table.dataProgram {
		_, _ = fmt.Fprintf(w, "[%.2d] ", i)
		for _, s := range p.sectionsPtr {
			_, _ = fmt.Fprintf(w, "%s ", s.Name)
		}
		_, _ = fmt.Fprintf(w, "\n")
	}
	_ = w.Flush()
}
