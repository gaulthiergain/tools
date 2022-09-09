// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package alignertool

import (
	"errors"
	"os"
	"path/filepath"
	"tools/srcs/alignertool/ukManager"
	"tools/srcs/binarytool/elf64analyser"
	u "tools/srcs/common"
)

// RunAligner allows to run the aligner tool (which is out of the
// UNICORE toolchain).
func RunAligner(homeDir string) {

	// Init and parse local arguments
	args := new(u.Arguments)
	p, err := args.InitArguments("--aligner",
		"The Aligner tool allows to align several unikernels for memory deduplication")
	if err != nil {
		u.PrintErr(err)
	}
	if err := parseLocalArguments(p, args); err != nil {
		u.PrintErr(err)
	}

	// Check if a json file is used or if it is via command line
	manager := new(ukManager.Manager)
	manager.MicroLibs = make(map[string]*ukManager.MicroLib)
	if len(*args.StringArg[rootArg]) > 0 {
		manager.Unikernels = make([]*ukManager.Unikernel, 0)
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

				manager.Unikernels = append(manager.Unikernels, &ukManager.Unikernel{
					BuildPath: basename,
				})
			}
		}
	} else {
		u.PrintErr(errors.New("argument(s) must be provided"))
	}

	for _, uk := range manager.Unikernels {

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

		manager.ComputeAlignment(*uk)
	}

	manager.PerformAlignement()
}
