// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package ukManager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"tools/srcs/binarytool/elf64analyser"
	"tools/srcs/binarytool/elf64core"
	u "tools/srcs/common"
)

const (
	makefile = "Makefile"
	config   = "config"
	ldExt    = ".ld.o"
	dbgExt   = ".dbg"
)

type Unikernels struct {
	Unikernel []Unikernel `json:"unikernels"`
}

type Unikernel struct {
	BuildPath          string `json:"buildPath"`
	Kernel             string `json:"kernel"`
	SectionSplit       string `json:"splitSection"`
	DisplayMapping     bool   `json:"displayMapping"`
	DisplayStatSize    bool   `json:"displayStatSize"`
	ComputeLibsMapping bool   `json:"computeLibsMapping"`

	IgnoredPlats       []string `json:"ignoredPlats"`
	DisplayElfFile     []string `json:"displayElfFile"`
	DisplaySectionInfo []string `json:"displaySectionInfo"`

	FindSectionByAddress []string `json:"findSectionByAddress"`
	CompareGroup         int      `json:"compareGroup"`

	// Used to generate new link.lds file
	ComputeTextAddr string   `json:"computeTextAddr"`
	LibsMapping     []string `json:"LibsMapping"`

	ElfFile  *elf64core.ELF64File
	ListObjs []*elf64core.ELF64File
	Analyser *elf64analyser.ElfAnalyser

	alignedLibs *AlignedLibs
	strBuilder  strings.Builder
}

type AlignedLibs struct {
	startValueUk       uint64
	startValueInit     uint64
	AllCommonMicroLibs []*MicroLib
	OnlyFewMicroLibs   []*MicroLib
	SingleMicroLibs    []*MicroLib
}

func ReadJsonFile(path string) ([]*Unikernel, error) {
	jsonFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	unikernels := new(Unikernels)
	if err := json.Unmarshal(byteValue, unikernels); err != nil {
		return nil, err
	}

	uks := make([]*Unikernel, len(unikernels.Unikernel))
	for i, _ := range unikernels.Unikernel {
		uks[i] = &unikernels.Unikernel[i]
	}
	unikernels = nil
	return uks, nil
}

func parseFile(path, name string) (*elf64core.ELF64File, error) {
	var elfFile *elf64core.ELF64File
	elfFile = new(elf64core.ELF64File)
	if err := elfFile.ParseAll(path, name); err != nil {
		return nil, err
	}
	return elfFile, nil
}

func (uk *Unikernel) GetKernel() error {
	var err error
	uk.ElfFile, err = parseFile("", uk.Kernel)
	if err != nil {
		return err
	}
	return nil
}

func (uk *Unikernel) GetFiles() error {
	files, err := ioutil.ReadDir(uk.BuildPath)
	if err != nil {
		return err
	}

	uk.ListObjs = make([]*elf64core.ELF64File, 0)
	foundExec := false
	for _, f := range files {

		if f.IsDir() || strings.Contains(f.Name(), makefile) ||
			strings.Contains(f.Name(), config) {
			continue
		}

		if strings.Contains(f.Name(), ldExt) &&
			!stringInSlice(f.Name(), uk.IgnoredPlats) {
			objFile, err := parseFile(uk.BuildPath, f.Name())
			if err != nil {
				return err
			}

			uk.ListObjs = append(uk.ListObjs, objFile)
		} else if filepath.Ext(strings.TrimSpace(f.Name())) == dbgExt &&
			!stringInSlice(f.Name(), uk.IgnoredPlats) && !foundExec {

			execName := f.Name()
			if len(uk.Kernel) > 0 {
				execName = uk.Kernel
			}
			uk.ElfFile, err = parseFile(uk.BuildPath, execName)
			if err != nil {
				return err
			}
			foundExec = true
		}
	}

	if len(uk.Kernel) > 0 {
		u.PrintInfo("Use specified ELF file: " + uk.ElfFile.Name + "(" + uk.BuildPath + ")")
	} else {
		u.PrintInfo("Use ELF file found in build folder: " + uk.ElfFile.Name)
	}
	return nil
}

func (uk *Unikernel) displayAllElfInfo() {
	uk.ElfFile.Header.DisplayHeader()
	uk.ElfFile.SectionsTable.DisplaySections()
	uk.ElfFile.DisplayRelocationTables()
	uk.ElfFile.DisplaySymbolsTables()
	uk.ElfFile.DynamicTable.DisplayDynamicEntries()
	uk.ElfFile.SegmentsTable.DisplayProgramHeader()
	uk.ElfFile.SegmentsTable.DisplaySegmentSectionMapping()
	uk.ElfFile.DisplayNotes()
	uk.ElfFile.DisplayFunctionsTables(false)
}

func (uk *Unikernel) DisplayElfInfo() {

	if len(uk.DisplayElfFile) == 1 && uk.DisplayElfFile[0] == "all" {
		uk.displayAllElfInfo()
	} else {
		for _, d := range uk.DisplayElfFile {
			if d == "header" {
				uk.ElfFile.Header.DisplayHeader()
			} else if d == "sections" {
				uk.ElfFile.SectionsTable.DisplaySections()
			} else if d == "relocations" {
				uk.ElfFile.DisplayRelocationTables()
			} else if d == "symbols" {
				uk.ElfFile.DisplaySymbolsTables()
			} else if d == "dynamics" {
				uk.ElfFile.DynamicTable.DisplayDynamicEntries()
			} else if d == "segments" {
				uk.ElfFile.SegmentsTable.DisplayProgramHeader()
			} else if d == "mapping" {
				uk.ElfFile.SegmentsTable.DisplaySegmentSectionMapping()
			} else if d == "notes" {
				uk.ElfFile.DisplayNotes()
			} else if d == "functions" {
				uk.ElfFile.DisplayFunctionsTables(false)
			} else {
				u.PrintWarning("No display configuration found for argument: " + d)
			}
		}
	}
}

func (uk *Unikernel) InitAlignment() {
	uk.alignedLibs = &AlignedLibs{
		startValueUk:       0,
		startValueInit:     0,
		AllCommonMicroLibs: make([]*MicroLib, 0),
		OnlyFewMicroLibs:   make([]*MicroLib, 0),
		SingleMicroLibs:    make([]*MicroLib, 0),
	}
}

func (uk *Unikernel) AddAlignedMicroLibs(startValue uint64, lib *MicroLib) {
	if _, ok := uk.Analyser.MapElfLibs[lib.name]; ok {
		lib.startAddr = startValue
		uk.alignedLibs.OnlyFewMicroLibs = append(uk.alignedLibs.OnlyFewMicroLibs, lib)
	}
}

func (uk *Unikernel) AddSingleMicroLibs(startValue uint64, lib *MicroLib) {
	if _, ok := uk.Analyser.MapElfLibs[lib.name]; ok {

		uk.alignedLibs.SingleMicroLibs = append(uk.alignedLibs.SingleMicroLibs, lib)
		if uk.alignedLibs.startValueInit == 0 {
			uk.alignedLibs.startValueInit = startValue
			uk.alignedLibs.startValueUk = startValue
		}
		uk.alignedLibs.startValueUk += lib.size
	}
}

func (uk *Unikernel) writeTextAlignment(startValue uint64) {

	uk.strBuilder = strings.Builder{}
	uk.strBuilder.WriteString("SECTIONS\n{\n")
	uk.strBuilder.WriteString(fmt.Sprintf(" . = 0x%x;\n", startValue))
	for _, lib := range uk.alignedLibs.AllCommonMicroLibs {
		uk.strBuilder.WriteString(fmt.Sprintf(" .text.%s : {\n\t %s(.text);\n }\n", strings.Replace(lib.name, ldExt, "", -1), lib.name))
	}

	for _, lib := range uk.alignedLibs.OnlyFewMicroLibs {
		uk.strBuilder.WriteString(fmt.Sprintf(" .text.%s 0x%x: {\n\t %s(.text);\n }\n", strings.Replace(lib.name, ldExt, "", -1), lib.startAddr, lib.name))
	}

	uk.strBuilder.WriteString(fmt.Sprintf(" . = 0x%x;\n", uk.alignedLibs.startValueInit))

	for _, lib := range uk.alignedLibs.SingleMicroLibs {
		uk.strBuilder.WriteString(fmt.Sprintf(" .text.%s : {\n\t %s(.text);\n }\n", strings.Replace(lib.name, ldExt, "", -1), lib.name))
	}

	uk.strBuilder.WriteString("}\n")
}

func (uk *Unikernel) sectionsObjs(linkerInfo LinkerInfo) string {

	// 0: rodata, 1: data, 2: bss
	strBuilder := [3]strings.Builder{}
	for _, obj := range uk.alignedLibs.AllCommonMicroLibs {
		if obj.name == ukbootMain {
			// Ignore ukbootMain
			continue
		}
		strBuilder[0].WriteString(fmt.Sprintf(". = ABSOLUTE(0x%x);%s (.rodata);\n", obj.sectionSize.rodataAddr, obj.name))
		strBuilder[1].WriteString(fmt.Sprintf(". = ABSOLUTE(0x%x);%s (.data);\n", obj.sectionSize.dataAddr, obj.name))
		strBuilder[2].WriteString(fmt.Sprintf(". = ABSOLUTE(0x%x);%s (.bss);\n", obj.sectionSize.bssAddr, obj.name))
	}

	for _, obj := range uk.alignedLibs.OnlyFewMicroLibs {
		strBuilder[0].WriteString(fmt.Sprintf(". = ABSOLUTE(0x%x);%s (.rodata);\n", obj.sectionSize.rodataAddr, obj.name))
		strBuilder[1].WriteString(fmt.Sprintf(". = ABSOLUTE(0x%x);%s (.data);\n", obj.sectionSize.dataAddr, obj.name))
		strBuilder[2].WriteString(fmt.Sprintf(". = ABSOLUTE(0x%x);%s (.bss);\n", obj.sectionSize.bssAddr, obj.name))
	}

	// Add ukbootMain before single microlibs
	strBuilder[0].WriteString(fmt.Sprintf("%s (.rodata);\n", ukbootMain))
	strBuilder[1].WriteString(fmt.Sprintf("%s (.data);\n", ukbootMain))
	strBuilder[2].WriteString(fmt.Sprintf("%s (.bss);\n", ukbootMain))

	for _, obj := range uk.alignedLibs.SingleMicroLibs {
		strBuilder[0].WriteString(fmt.Sprintf("%s (.rodata);\n", obj.name))
		strBuilder[1].WriteString(fmt.Sprintf("%s (.data);\n", obj.name))
		strBuilder[2].WriteString(fmt.Sprintf("%s (.bss);\n", obj.name))
	}

	arrLoc := []string{inner_rodata, inner_data, inner_bss}
	for i, builder := range strBuilder {
		linkerInfo.ldsString = strings.Replace(linkerInfo.ldsString, arrLoc[i], builder.String(), -1)
	}

	return linkerInfo.ldsString
}

func (uk *Unikernel) writeLdsToFile(filename string, linkerInfo LinkerInfo) {

	// Replace rodata, data,bss from object section
	linkerInfo.ldsString = uk.sectionsObjs(linkerInfo)
	if err := os.WriteFile(filename, []byte(uk.strBuilder.String()+linkerInfo.ldsString), 0644); err != nil {
		u.PrintErr(err)
	}
}
