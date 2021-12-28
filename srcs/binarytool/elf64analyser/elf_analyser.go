// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package elf64analyser

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"tools/srcs/binarytool/elf64core"
	u "tools/srcs/common"
)

type ElfAnalyser struct {
	ElfLibs    []ElfLibs
	ElfPage    []*ElfPage
	MapElfLibs map[string]*ElfLibs
}

type ElfLibs struct {
	Name      string
	StartAddr uint64
	EndAddr   uint64
	Size      uint64
	NbSymbols int

	RodataSize uint64
	DataSize   uint64
	BssSize    uint64
}

func (analyser *ElfAnalyser) DisplayMapping() {

	if len(analyser.ElfLibs) == 0 {
		fmt.Println("Mapping is empty")
		return
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	fmt.Println("-----------------------------------------------------------------------")
	_, _ = fmt.Fprintln(w, "Name \tStart \tEnd \tSize \tNbSymbols\tSizeDiv\tnbDiv")
	for _, lib := range analyser.ElfLibs {

		var name = lib.Name
		if strings.Contains(lib.Name, string(os.PathSeparator)) {
			split := strings.Split(lib.Name, string(os.PathSeparator))
			name = split[len(split)-1]
		}

		_, _ = fmt.Fprintf(w, "%s \t0x%x \t0x%x \t0x%x\t%d\t%f\t%f\n",
			name, lib.StartAddr, lib.EndAddr, lib.Size,
			lib.NbSymbols, float32(lib.Size)/float32(PageSize),
			float32(lib.StartAddr)/float32(PageSize))
	}
	_ = w.Flush()
}

func filterFunctions(objFuncsAll []elf64core.ELF64Function, elfFuncsAll []elf64core.ELF64Function) []*elf64core.ELF64Function {

	i := 0
	filteredFuncs := make([]*elf64core.ELF64Function, len(objFuncsAll))
	for j, _ := range elfFuncsAll {

		if strings.Compare(objFuncsAll[i].Name, elfFuncsAll[j].Name) == 0 {
			filteredFuncs[i] = &elfFuncsAll[j]
			i++
		} else {
			// Reset the counter if we do not have consecutive functions
			i = 0
			// Special case where we can skip a function if we do not check again
			if strings.Compare(objFuncsAll[i].Name, elfFuncsAll[j].Name) == 0 {
				filteredFuncs[i] = &elfFuncsAll[j]
				i++
			}
		}

		if i == len(objFuncsAll) {
			return filteredFuncs
		}
	}
	return nil
}

func compareFunctions(elf *elf64core.ELF64File, obj *elf64core.ELF64File) (uint64, uint64, int) {

	// Obj: Merge all functions table(s) in one slice for simplicity
	objFuncs := make([]elf64core.ELF64Function, 0)
	for i := len(obj.FunctionsTables) - 1; i >= 0; i-- {
		if strings.Compare(obj.FunctionsTables[i].Name, elf64core.BootTextSection) != 0 {
			// Ignore the '.text.boot' and '.unlikely' sections since it can be split through
			// different places
			if !strings.Contains(obj.FunctionsTables[i].Name, elf64core.UnlikelySection) {
				objFuncs = append(objFuncs, obj.FunctionsTables[i].Functions...)
			}
		}

	}
	// Elf: Merge all functions table(s) in one slice for simplicity
	elfFuncs := make([]elf64core.ELF64Function, 0)
	for i := len(elf.FunctionsTables) - 1; i >= 0; i-- {
		if strings.Compare(elf.FunctionsTables[i].Name, elf64core.BootTextSection) != 0 {
			// Ignore the '.text.boot' section since it can be split through
			// different places
			elfFuncs = append(elfFuncs, elf.FunctionsTables[i].Functions...)
		}
	}

	// Add functions into a map for better search
	mapObjFuncs := make(map[string]*elf64core.ELF64Function)
	for i := 0; i < len(objFuncs); i++ {
		mapObjFuncs[objFuncs[i].Name] = &objFuncs[i]
	}

	elfFuncsAll := make([]elf64core.ELF64Function, 0)
	mapArrayFuncs := make(map[string]uint64, 0)
	for _, elfFunc := range elfFuncs {

		if _, ok := mapObjFuncs[elfFunc.Name]; ok {
			// Check if the function is already in mapArrayFuncs
			val, ok := mapArrayFuncs[elfFunc.Name]
			// Do not add duplicate functions (check on addresses)
			if !ok {
				mapArrayFuncs[elfFunc.Name] = elfFunc.Addr
				elfFuncsAll = append(elfFuncsAll, elfFunc)
			} else if val != elfFunc.Addr {
				elfFuncsAll = append(elfFuncsAll, elfFunc)
			}

		}
	}

	if len(elfFuncsAll) == 0 {
		u.PrintWarning(fmt.Sprintf("Cannot extract mapping of lib %s: No function", obj.Name))
		return 0, 0, 0
	}

	if len(elfFuncsAll) != len(objFuncs) {
		// We do not have the same set of functions, need to filter it.
		filteredFuncs := filterFunctions(objFuncs, elfFuncsAll)
		if filteredFuncs == nil {
			u.PrintWarning(fmt.Sprintf("Cannot extract mapping of lib %s: Different size", obj.Name))
			return 0, 0, 0
		}
		return filteredFuncs[0].Addr, filteredFuncs[len(filteredFuncs)-1].Size +
			filteredFuncs[len(filteredFuncs)-1].Addr, len(filteredFuncs)
	}

	return elfFuncsAll[0].Addr, elfFuncsAll[len(elfFuncsAll)-1].Size +
		elfFuncsAll[len(elfFuncsAll)-1].Addr, len(elfFuncsAll)
}

func getSectionSize(name string, obj *elf64core.ELF64File) uint64 {
	if index, ok := obj.IndexSections[name]; ok {
		return obj.SectionsTable.DataSect[index].Elf64section.Size
	}
	return 0
}

func (analyser *ElfAnalyser) InspectMappingList(elf *elf64core.ELF64File,
	objs []*elf64core.ELF64File) {

	if len(objs) == 0 {
		return
	}

	analyser.ElfLibs = make([]ElfLibs, len(objs))
	analyser.MapElfLibs = make(map[string]*ElfLibs, len(analyser.ElfLibs))
	for i, obj := range objs {

		start, end, nbSymbols := compareFunctions(elf, obj)
		lib := ElfLibs{
			Name:      obj.Name,
			StartAddr: start,
			EndAddr:   end,
			Size:      end - start,
			NbSymbols: nbSymbols,
			// Get size of data, rodata and bss from object file
			RodataSize: getSectionSize(".rodata", obj),
			DataSize:   getSectionSize(".data", obj),
			BssSize:    getSectionSize(".bss", obj),
		}
		analyser.ElfLibs[i] = lib
		// Map for direct access
		analyser.MapElfLibs[lib.Name] = &lib
	}

	// sort functions by start address.
	sort.Slice(analyser.ElfLibs, func(i, j int) bool {
		return analyser.ElfLibs[i].StartAddr < analyser.ElfLibs[j].StartAddr
	})
}

/*
func (analyser *ElfAnalyser) SplitIntoPagesBySection(elfFile *elf64core.ELF64File, sectionName string) {

	if len(analyser.ElfPage) == 0 {
		analyser.ElfPage = make([]*ElfPage, 0)
	}

	if strings.Contains(sectionName, elf64core.TextSection) {
		// An ELF might have several text sections
		for _, indexSection := range elfFile.TextSectionIndex {
			analyser.computePage(elfFile, elfFile.SectionsTable.DataSect[indexSection].Name, indexSection)
		}
	} else if indexSection, ok := elfFile.IndexSections[sectionName]; ok {
		analyser.computePage(elfFile, sectionName, indexSection)
	} else {
		u.PrintWarning(fmt.Sprintf("Cannot split section %s into pages", sectionName))
	}
}


func CreateNewPage(startAddress uint64, k int, raw []byte) *ElfPage {
	byteArray := make([]byte, PageSize)
	b := raw
	if cpd := copy(byteArray, b); cpd == 0 {
		u.PrintWarning("0 bytes were copied")
	}
	page := &ElfPage{
		number:           k,
		startAddress:     startAddress,
		contentByteArray: byteArray,
	}
	h := sha256.New()
	h.Write(page.contentByteArray)
	page.hash = hex.EncodeToString(h.Sum(nil))
	return page
}

func (analyser *ElfAnalyser) computePage(elfFile *elf64core.ELF64File, section string, indexSection int) {
	offsetTextSection := elfFile.SectionsTable.DataSect[indexSection].Elf64section.FileOffset
	k := 0
	for i := offsetTextSection; i < offsetTextSection+elfFile.SectionsTable.DataSect[indexSection].Elf64section.Size; i += PageSize {

		end := i + PageSize
		if end >= uint64(len(elfFile.Raw)) {
			end = uint64(len(elfFile.Raw) - 1)
		}
		page := CreateNewPage(i, k, elfFile.Raw[i:end])
		page.sectionName = section
		analyser.ElfPage = append(analyser.ElfPage, page)
		k++
	}
}
*/
