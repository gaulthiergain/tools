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
	"path/filepath"
	"strconv"
	"tools/srcs/binarytool/elf64analyser"
	u "tools/srcs/common"
)

const diffPath = "diff" + u.SEP
const pagesPath = "pages" + u.SEP

type BinaryManager struct {
	Unikernels []*Unikernel
}

// RunBinaryAnalyser allows to run the binary analyser tool (which is out of the
// UNICORE toolchain).
func RunBinaryAnalyser(homeDir string) {

	// Init and parse local arguments
	args := new(u.Arguments)
	p, err := args.InitArguments("--binary",
		"The Binary tool allows to help developers to gather stats on unikernels binaries")
	if err != nil {
		u.PrintErr(err)
	}
	if err := parseLocalArguments(p, args); err != nil {
		u.PrintErr(err)
	}

	// Check if a json file is used or if it is via command line
	manager := new(BinaryManager)

	if len(*args.StringArg[rootArg]) > 0 {
		manager.Unikernels = make([]*Unikernel, 0)
		mapping := false
		if *args.BoolArg[mappingArg] {
			mapping = true
		}
		files, err := os.ReadDir(*args.StringArg[rootArg])
		if err != nil {
			u.PrintErr(err)
		}
		for _, file := range files {
			if file.IsDir() {

				basename := filepath.Join(*args.StringArg[rootArg], file.Name())
				if _, err := os.Stat(filepath.Join(basename, "build")); !os.IsNotExist(err) {
					// A build folder exist
					basename = filepath.Join(basename, "build")
				}

				manager.Unikernels = append(manager.Unikernels, &Unikernel{
					BuildPath:      basename,
					DisplayMapping: mapping,
				})
			}
		}
	} else if len(*args.StringArg[filesArg]) > 0 {
		var err error
		manager.Unikernels, err = ReadJsonFile(*args.StringArg[filesArg])
		if err != nil {
			u.PrintErr(err)
		}

	} else {
		u.PrintErr(errors.New("argument(s) must be provided"))
	}

	var comparison elf64analyser.ComparisonElf
	comparison.GroupFileSegment = make([]*elf64analyser.ElfFileSegment, 0)

	var UnikernelsPath []string
	UnikernelsPath = append(UnikernelsPath, "-u")
	var SplitSects []string
	SplitSects = append(SplitSects, "-l")

	for i, uk := range manager.Unikernels {

		uk.Analyser = new(elf64analyser.ElfAnalyser)

		if len(uk.BuildPath) > 0 {

			if _, err := os.Stat(filepath.Join(uk.BuildPath, "build")); !os.IsNotExist(err) {
				// A build folder exist
				uk.BuildPath = filepath.Join(uk.BuildPath, "build")
			} else {
				u.PrintWarning("Cannot find 'build/' folder, skip this configuration...")
			}

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

		if uk.CompareGroup > 0 {
			UnikernelsPath = append(UnikernelsPath, uk.BuildPath+uk.Kernel)
			for _, sp := range uk.SplitSections {
				SplitSects = append(SplitSects, sp)
			}
		}
	}

	u.PrintInfo("Analysing the following sections of " + strconv.Itoa(len(UnikernelsPath)-1) + " unikernels. This may take some time...")
	for i, sect := range SplitSects {
		if i > 0 {
			println("- " + sect)
		}
	}
	script := filepath.Join(os.Getenv("GOPATH"), "src", "tools", "srcs", "binarytool", "scripts", "uk_elf_sharing.py")
	out, err := u.ExecuteCommand(script, append(UnikernelsPath, SplitSects...))
	if err != nil {
		u.PrintErr(err)
	}
	u.PrintOk("Finish to analyse " + strconv.Itoa(len(UnikernelsPath)-1) + " unikernels")
	u.PrintInfo("Displaying stats")

	println(out)

	u.PrintOk("Diff files have been saved to ./diff/")
	u.PrintOk("Pages files have been saved to ./pages/")
}
