// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package elf64core

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"os"
	"text/tabwriter"
	u "tools/srcs/common"
)

type RelaTables struct {
	nbEntries int
	name      string
	dataRela  []*dataRela
}

type dataRela struct {
	name      *string
	elf64Rela Elf64Rela
}

type Elf64Rela struct {
	Offset      uint64
	Type        uint32
	SymbolIndex uint32
	Addend      int64
}

func (elfFile *ELF64File) addRela(index int, relas []Elf64Rela) error {

	var relocationTables RelaTables
	relocationTables.nbEntries = len(relas)
	relocationTables.name = elfFile.SectionsTable.DataSect[index].Name
	relocationTables.dataRela = make([]*dataRela, relocationTables.nbEntries)

	for i := range relas {

		relocationTables.dataRela[i] = &dataRela{
			elf64Rela: relas[i],
			name:      nil,
		}
	}

	elfFile.RelaTables = append(elfFile.RelaTables, relocationTables)

	return nil
}

func (elfFile *ELF64File) parseRelocations(index int) error {

	content, err := elfFile.GetSectionContent(uint16(index))

	if err != nil {
		return fmt.Errorf("failed reading relocation table: %s", err)
	}

	rela := make([]Elf64Rela, len(content)/binary.Size(Elf64Rela{}))
	if err := binary.Read(bytes.NewReader(content),
		elfFile.Endianness, rela); err != nil {
		return err
	}

	if err := elfFile.addRela(index, rela); err != nil {
		return err
	}

	return nil
}

func (elfFile *ELF64File) resolveRelocSymbolsName() error {
	for _, table := range elfFile.RelaTables {
		for _, s := range table.dataRela {
			t := 0
			if s.elf64Rela.Type == uint32(elf.R_X86_64_JMP_SLOT) ||
				s.elf64Rela.Type == uint32(elf.R_X86_64_GLOB_DAT) ||
				s.elf64Rela.Type == uint32(elf.R_X86_64_COPY) &&
					len(elfFile.SymbolsTables) > 1 {
				t++
			}

			symName, err := elfFile.SymbolsTables[t].getSymbolName(s.elf64Rela.SymbolIndex)
			if err != nil {
				return err
			}
			s.name = &symName
		}
	}
	return nil
}

func (table *RelaTables) displayRelocations(w *tabwriter.Writer) {
	_, _ = fmt.Fprintf(w, "\nRelocation section '%s' contains %d entries:\n",
		table.name, table.nbEntries)
	_, _ = fmt.Fprintln(w, "Offset\tInfo\tType\tValue")
	for _, r := range table.dataRela {
		_, _ = fmt.Fprintf(w, "%.6x\t%.6d\t%s\t%s %x\n",
			r.elf64Rela.Offset, r.elf64Rela.SymbolIndex,
			rx86_64Strings[r.elf64Rela.Type], *r.name,
			r.elf64Rela.Addend)
	}
}

func (elfFile *ELF64File) DisplayRelocationTables() {

	if len(elfFile.RelaTables) == 0 {
		u.PrintWarning("Relocation table(s) are empty")
		return
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Println("-----------------------------------------------------------------------")

	for _, table := range elfFile.RelaTables {
		table.displayRelocations(w)
	}
	_ = w.Flush()
}
