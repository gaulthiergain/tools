// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package binarytool

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"tools/srcs/binarytool/elf64analyser"
	"tools/srcs/binarytool/ukManager"
	u "tools/srcs/common"
)

const diffPath = "diff" + u.SEP
const pagesPath = "pages" + u.SEP

// RunBinaryAnalyser allows to run the binary analyser tool (which is out of the
// UNICORE toolchain).
func RunBinaryAnalyser(homeDir string) {

	// Init and parse local arguments
	args := new(u.Arguments)
	p, err := args.InitArguments()
	if err != nil {
		u.PrintErr(err)
	}
	if err := parseLocalArguments(p, args); err != nil {
		u.PrintErr(err)
	}

	// Check if a json file is used or if it is via command line
	manager := new(ukManager.Manager)
	manager.MicroLibs = make(map[string]*ukManager.MicroLib)
	if len(*args.StringArg[listArg]) > 0 {
		manager.Unikernels = make([]*ukManager.Unikernel, len(*args.StringArg[listArg]))
		mapping := false
		if *args.BoolArg[mappingArg] {
			mapping = true
		}
		list := strings.Split(*args.StringArg[listArg], ",")
		for i, arg := range list {
			manager.Unikernels[i] = &ukManager.Unikernel{
				BuildPath:      arg,
				DisplayMapping: mapping,
			}
		}
	} else if len(*args.StringArg[filesArg]) > 0 {
		var err error
		manager.Unikernels, err = ukManager.ReadJsonFile(*args.StringArg[filesArg])
		if err != nil {
			u.PrintErr(err)
		}
	} else {
		u.PrintErr(errors.New("argument(s) must be provided"))
	}

	var comparison elf64analyser.ComparisonElf
	comparison.GroupFileSegment = make([]*elf64analyser.ElfFileSegment, 0)

	for i, uk := range manager.Unikernels {

		uk.Analyser = new(elf64analyser.ElfAnalyser)
		if len(uk.BuildPath) > 0 {
			if uk.BuildPath[len(uk.BuildPath)-1] != os.PathSeparator {
				uk.BuildPath += u.SEP
			}
			if err := uk.GetFiles(); err != nil {
				u.PrintErr(err)
			}

			// Perform the inspection of micro-libs since we have the buildPath
			uk.Analyser.InspectMappingList(uk.ElfFile, uk.ListObjs)
		} else {
			if err := uk.GetKernel(); err != nil {
				u.PrintErr(err)
			}
		}

		if len(uk.DisplayElfFile) > 0 {
			uk.DisplayElfInfo()
		}

		if uk.DisplayMapping && len(uk.BuildPath) > 0 {
			fmt.Printf("==========[(%d): %s]==========\n", i, uk.BuildPath)
			uk.Analyser.DisplayMapping()
			fmt.Println("=====================================================")
		}

		if uk.DisplayStatSize {
			uk.Analyser.DisplayStatSize(uk.ElfFile)
		}

		if len(uk.DisplaySectionInfo) > 0 {
			uk.Analyser.DisplaySectionInfo(uk.ElfFile, uk.DisplaySectionInfo)
		}

		if len(uk.FindSectionByAddress) > 0 {
			uk.Analyser.FindSectionByAddress(uk.ElfFile, uk.FindSectionByAddress)
		}

		manager.ComputeAlignment(*uk)

		/*if uk.CompareGroup > 0 {

			foundSection := false
			section := uk.SectionSplit
			for _, s := range uk.ElfFile.SectionsTable.DataSect {
				if s.Name == section {
					foundSection = true
					break
				}
			}

			if foundSection && len(uk.SectionSplit) > 0 {

				path := homeDir + u.SEP + pagesPath
				if _, err := os.Stat(path); os.IsNotExist(err) {
					err := os.Mkdir(path, os.ModePerm)
					if err != nil {
						u.PrintErr(err)
					}
				}

				u.PrintInfo(fmt.Sprintf("Splitting %s section of %s into pages...", section, uk.ElfFile.Name))
				uk.Analyser.SplitIntoPagesBySection(uk.ElfFile, section)

				out := path + section[1:] + u.SEP

				if _, err := os.Stat(out); os.IsNotExist(err) {
					err := os.Mkdir(out, os.ModePerm)
					if err != nil {
						u.PrintErr(err)
					}
				}

				if err := elf64analyser.SavePagesToFile(uk.Analyser.ElfPage, out+uk.ElfFile.Name+".txt", false); err != nil {
					u.PrintErr(err)
				}
				u.PrintOk(fmt.Sprintf("Pages of section %s (%s) are saved into %s", section, uk.ElfFile.Name, out))

				comparison.GroupFileSegment = append(comparison.GroupFileSegment,
					&elf64analyser.ElfFileSegment{Filename: uk.ElfFile.Name,
						NbPages: len(uk.Analyser.ElfPage), Pages: uk.Analyser.ElfPage})
			} else if len(uk.SectionSplit) > 0 {
				u.PrintWarning("Section '" + section + "' is not found in the ELF file")
			}
		}*/
	}

	manager.PerformAlignement()

	/*
		if uk.ComputeLibsMapping && len(uk.LibsMapping) > 0 {

					if err != nil {
						u.PrintErr(err)
					} else {
						uk.Analyser.ComputeAlignedMapping(uk.ElfFile, uk.LibsMapping)
					}
				}
	*/

	if len(comparison.GroupFileSegment) > 1 {

		// Perform the comparison
		path := homeDir + u.SEP + diffPath
		if _, err := os.Stat(path); os.IsNotExist(err) {
			err := os.Mkdir(path, os.ModePerm)
			if err != nil {
				u.PrintErr(err)
			}
		}

		comparison.ComparePageTables()
		if err := comparison.DiffComparison(path); err != nil {
			u.PrintWarning(err)
		}
		comparison.DisplayComparison()
	}
}
