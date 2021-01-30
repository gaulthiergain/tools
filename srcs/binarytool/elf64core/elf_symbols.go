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

type SymbolsTables struct {
	nbEntries   int
	name        string
	dataSymbols []*dataSymbols
}

type dataSymbols struct {
	elf64sym   ELF64Symbols
	name       string
	TypeSymbol byte
}

type ELF64Symbols struct {
	Name  uint32
	Info  byte
	Other byte
	Shndx uint16
	Value uint64
	Size  uint64
}

func (elfFile *ELF64File) addSymbols(index int, symbols []ELF64Symbols) error {

	var symbolsTables SymbolsTables
	symbolsTables.nbEntries = len(symbols)
	symbolsTables.name = elfFile.SectionsTable.DataSect[index].Name
	symbolsTables.dataSymbols = make([]*dataSymbols, symbolsTables.nbEntries)

	var nameString string
	var err error
	for j, s := range symbols {

		if s.Info == byte(elf.STT_SECTION) {
			// This is a section, save its name
			nameString = elfFile.SectionsTable.DataSect[s.Shndx].Name
		} else {
			nameString, err = elfFile.GetSectionName(s.Name, uint16(index+1))
			if err != nil {
				return err
			}
		}

		symbolsTables.dataSymbols[j] = &dataSymbols{
			elf64sym:   s,
			name:       nameString,
			TypeSymbol: (s.Info) & 0xf,
		}
	}

	elfFile.SymbolsTables = append(elfFile.SymbolsTables, symbolsTables)

	return nil
}

func (elfFile *ELF64File) parseSymbolsTable(index int) error {

	content, err := elfFile.GetSectionContent(uint16(index))

	if err != nil {
		return fmt.Errorf("failed reading string table: %s", err)
	}

	if content[len(content)-1] != 0 {
		return fmt.Errorf("the string table isn't null-terminated")
	}

	symbols := make([]ELF64Symbols, len(content)/binary.Size(ELF64Symbols{}))
	if err := binary.Read(bytes.NewReader(content),
		elfFile.Endianness, symbols); err != nil {
		return err
	}

	if err := elfFile.addSymbols(index, symbols); err != nil {
		return err
	}

	return nil
}

func (table *SymbolsTables) displaySymbols(w *tabwriter.Writer) {
	_, _ = fmt.Fprintf(w, "\nSymbol table %s contains %d entries:\n\n",
		table.name, table.nbEntries)

	_, _ = fmt.Fprintf(w, "\nNum:\tValue\tSize\tName\tType\n")

	for i, s := range table.dataSymbols {
		_, _ = fmt.Fprintf(w, "%d:\t%.6x\t%d\t%s\t%s (%d)\n", i,
			s.elf64sym.Value, s.elf64sym.Size, s.name,
			sttStrings[s.TypeSymbol], s.TypeSymbol)
	}
}

func (elfFile *ELF64File) DisplaySymbolsTables() {

	if len(elfFile.SymbolsTables) == 0 {
		u.PrintWarning("Symbols table(s) are empty")
		return
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Println("-----------------------------------------------------------------------")

	for _, table := range elfFile.SymbolsTables {
		table.displaySymbols(w)
	}
	_ = w.Flush()
}

func (table *SymbolsTables) getSymbolName(index uint32) (string, error) {
	if table.dataSymbols == nil {
		return "", fmt.Errorf("symbol table is empty")
	}

	if uint32(table.nbEntries) <= index {
		return "", fmt.Errorf("invalid index %d", index)
	}

	return table.dataSymbols[index].name, nil
}
