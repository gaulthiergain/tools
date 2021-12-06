// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package binarytool

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"tools/srcs/binarytool/elf64analyser"
	"tools/srcs/binarytool/elf64core"
	u "tools/srcs/common"
)

const (
	makefile = "Makefile"
	config   = "config"
	objExt   = ".o"
	ldExt    = ".ld.o"
	dbgExt   = ".dbg"
)

type Unikernels struct {
	Unikernel []Unikernel `json:"unikernels"`
}

type Unikernel struct {
	BuildPath            string   `json:"buildPath"`
	Kernel               string   `json:"kernel"`
	SectionSplit         string   `json:"splitSection"`
	DisplayMapping       bool     `json:"displayMapping"`
	DisplayStatSize      bool     `json:"displayStatSize"`
	IgnoredPlats         []string `json:"ignoredPlats"`
	DisplayElfFile       []string `json:"displayElfFile"`
	DisplaySectionInfo   []string `json:"displaySectionInfo"`
	FindSectionByAddress []string `json:"findSectionByAddress"`
	CompareGroup         int      `json:"compareGroup"`

	ElfFile  *elf64core.ELF64File
	ListObjs []*elf64core.ELF64File
	Analyser *elf64analyser.ElfAnalyser
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
		u.PrintInfo("Use specified ELF file: " + uk.ElfFile.Name)
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
