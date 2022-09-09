// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package main

import (
	"os"
	"os/user"
	"tools/srcs/alignertool"
	"tools/srcs/binarytool"
	"tools/srcs/buildtool"
	u "tools/srcs/common"
	"tools/srcs/crawlertool"
	"tools/srcs/dependtool"
	"tools/srcs/extractertool"
	"tools/srcs/veriftool"
)

func main() {

	// Init global arguments
	args := new(u.Arguments)
	parser, err := args.InitArguments("The UNICORE Toolchain",
		"Toolkit with provides several tools to analyse, compare and build unikernels")
	if err != nil {
		u.PrintErr(err)
	}

	// Parse arguments
	if err := args.ParseMainArguments(parser, args); err != nil {
		u.PrintErr(err)
	}

	// Checks if the toolchain must be completely executed
	all := false
	if !*args.BoolArg[u.DEP] && !*args.BoolArg[u.BUILD] &&
		!*args.BoolArg[u.VERIF] && !*args.BoolArg[u.PERF] {
		all = true
	}

	// Get user home folder
	usr, err := user.Current()
	if err != nil {
		u.PrintErr(err)
	}

	var data *u.Data

	if *args.BoolArg[u.CRAWLER] {
		u.PrintHeader1("(*) RUN CRAWLER UNIKRAFT ANALYSER")
		crawlertool.RunCrawler()
		return
	}

	if *args.BoolArg[u.BINARY] {
		u.PrintHeader1("(*) RUN BINARY UNIKRAFT ANALYSER")
		binarytool.RunBinaryAnalyser(usr.HomeDir)
		return
	}

	if *args.BoolArg[u.ALIGNER] {
		u.PrintHeader1("(*) RUN ALIGNER TOOL")
		alignertool.RunAligner(usr.HomeDir)
		return
	}

	if *args.BoolArg[u.EXTRACTER] {
		u.PrintHeader1("(*) RUN EXTRACTER TOOL")
		extractertool.RunExtracterTool(usr.HomeDir)
		return
	}

	if all || *args.BoolArg[u.DEP] {

		// Initialize data
		data = new(u.Data)

		u.PrintHeader1("(*) RUN DEPENDENCIES ANALYSER")
		dependtool.RunAnalyserTool(usr.HomeDir, data)
	}

	if all || *args.BoolArg[u.BUILD] {
		u.PrintHeader1("(*) SEMI-AUTOMATIC BUILD TOOL")
		buildtool.RunBuildTool(usr.HomeDir, data)
	}

	if all || *args.BoolArg[u.VERIF] {
		u.PrintHeader1("(*) OUTPUT VERIFICATION TOOL")
		veriftool.RunVerificationTool(usr.HomeDir)
	}

	if all || *args.BoolArg[u.PERF] {
		u.PrintHeader1("(*) PERFORMANCE OPTIMIZATION TOOL (see way-finder)")
		os.Exit(1)
	}
}
