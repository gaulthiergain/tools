// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package elf64core

import (
	"debug/elf"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	u "tools/srcs/common"
)

type FunctionTables struct {
	Name      string
	NbEntries int
	Functions []ELF64Function
}

type ELF64Function struct {
	Name         string
	Addr         uint64
	Size         uint64
}

func (elfFile *ELF64File) getIndexFctTable(ndx uint16) int {

	if int(ndx) > len(elfFile.SectionsTable.DataSect) {
		return -1
	}

	sectionName := elfFile.SectionsTable.DataSect[ndx].Name
	for i, t := range elfFile.FunctionsTables {
		if strings.Compare(t.Name, sectionName) == 0 {
			return i
		}
	}

	return -1
}

func (elfFile *ELF64File) detectSizeSymbol(symbolsTable []ELF64Function, index int) uint64 {

	if index+1 == len(symbolsTable) {
		textIndex := elfFile.IndexSections[TextSection]
		textSection := elfFile.SectionsTable.DataSect[textIndex]
		size := textSection.Elf64section.FileOffset + textSection.Elf64section.Size
		return size - symbolsTable[index].Addr
	}
	return symbolsTable[index+1].Addr - symbolsTable[index].Addr
}

func (elfFile *ELF64File) inspectFunctions() error {

	for _, table := range elfFile.SymbolsTables {
		for _, s := range table.dataSymbols {
			k := elfFile.getIndexFctTable(s.elf64sym.Shndx)
			if k != -1 && s.elf64sym.Value > 0 {
				if !isInSlice(elfFile.Name, s.name) {
					function := ELF64Function{Name: s.name, Addr: s.elf64sym.Value,
						Size: s.elf64sym.Size}
					elfFile.FunctionsTables[k].Functions =
						append(elfFile.FunctionsTables[k].Functions, function)
				}
			} else if s.TypeSymbol == byte(elf.STT_FUNC) {
				// If it is a func where the start address starts at 0
				function := ELF64Function{Name: s.name, Addr: s.elf64sym.Value,
					Size: s.elf64sym.Size}
				elfFile.FunctionsTables[k].Functions =
					append(elfFile.FunctionsTables[k].Functions, function)
			}
		}
	}

	return nil
}

func (elfFile *ELF64File) parseFunctions() error {

	if _, ok := elfFile.IndexSections[TextSection]; ok {

		if err := elfFile.inspectFunctions(); err != nil {
			return err
		}

	} else {
		return errors.New("no text section detected")
	}

	for _, table := range elfFile.FunctionsTables {
		// sort Functions

		sort.Slice(table.Functions, func(i, j int) bool {
			return table.Functions[i].Addr < table.Functions[j].Addr
		})

		for i, f := range table.Functions {
			if f.Size == 0 {
				f.Size = elfFile.detectSizeSymbol(table.Functions, i)
			}

			// Special case where symbol of same address can be in different order
			// between the ELF and the object file
			if i < len(table.Functions)-1 && table.Functions[i].Addr == table.Functions[i+1].Addr {
				if strings.Compare(table.Functions[i].Name, table.Functions[i+1].Name) > 0 {
					swap(i, table.Functions)
				}
			}
		}
	}

	return nil
}

func swap(index int, x []ELF64Function) {
	x[index], x[index+1] = x[index+1], x[index]
}

func (table *FunctionTables) displayFunctions(w *tabwriter.Writer, fullDisplay bool) {

	_, _ = fmt.Fprintf(w, "\nTable section '%s' contains %d entries:\n",
		table.Name, table.NbEntries)
	_, _ = fmt.Fprintf(w, "Name:\tAddr:\tSize\tRaw:\n")
	for _, f := range table.Functions {
		_, _ = fmt.Fprintf(w, "%s\t%6.x\t%6.x\t%s\n", f.Name, f.Addr, f.Size)

	}
}

func (elfFile *ELF64File) DisplayFunctionsTables(fullDisplay bool) {

	if len(elfFile.FunctionsTables) == 0 {
		u.PrintWarning("Functions table(s) is/are empty")
		return
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Println("-----------------------------------------------------------------------")

	for _, table := range elfFile.FunctionsTables {
		table.displayFunctions(w, fullDisplay)
	}
	_ = w.Flush()
}
