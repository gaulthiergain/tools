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

type DynamicTable struct {
	nbEntries int
	name      string
	elf64dyn  []Elf64Dynamic
}

type Elf64Dynamic struct {
	Tag   uint64
	Value uint64
}

func (elfFile *ELF64File) addDynamicEntry(index int, dynEntries []Elf64Dynamic) error {

	elfFile.DynamicTable = DynamicTable{}
	elfFile.DynamicTable.nbEntries = len(dynEntries)
	elfFile.DynamicTable.name = elfFile.SectionsTable.DataSect[index].Name
	elfFile.DynamicTable.elf64dyn = make([]Elf64Dynamic, 0)

	for _, s := range dynEntries {
		elfFile.DynamicTable.elf64dyn = append(elfFile.DynamicTable.elf64dyn, s)
		if s.Tag == uint64(elf.DT_NULL) {
			return nil
		}
	}
	return nil
}

func (elfFile *ELF64File) parseDynamic(index int) error {

	content, err := elfFile.GetSectionContent(uint16(index))

	if err != nil {
		return fmt.Errorf("failed reading relocation table: %s", err)
	}

	dynEntries := make([]Elf64Dynamic, len(content)/binary.Size(Elf64Dynamic{}))
	if err := binary.Read(bytes.NewReader(content),
		elfFile.Endianness, dynEntries); err != nil {
		return err
	}

	if err := elfFile.addDynamicEntry(index, dynEntries); err != nil {
		return err
	}

	return nil
}

func (table *DynamicTable) DisplayDynamicEntries() {

	if len(table.elf64dyn) == 0 {
		u.PrintWarning("Dynamic table is empty")
		return
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Println("-----------------------------------------------------------------------")

	fmt.Printf("%s table contains %d entries:\n\n", table.name, table.nbEntries)
	_, _ = fmt.Fprintln(w, "Nr\tTag\tType\tValue")
	for i, s := range table.elf64dyn {
		_, _ = fmt.Fprintf(w, "%d:\t%.8x\t%s\t%x\n", i, s.Tag,
			dtStrings[s.Tag], s.Value)
	}
	_ = w.Flush()
}
