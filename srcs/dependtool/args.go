// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package dependtool

import (
	"github.com/akamensky/argparse"
	"os"
	u "tools/srcs/common"
)

const (
	programArg         = "program"
	testFileArg        = "testFile"
	configFileArg      = "configFile"
	optionsArg         = "options"
	waitTimeArg        = "waitTime"
	saveOutputArg      = "saveOutput"
	fullDepsArg        = "fullDeps"
	fullStaticAnalysis = "fullStaticAnalysis"
	typeAnalysis       = "typeAnalysis"
)

// parseLocalArguments parses arguments of the application.
func parseLocalArguments(p *argparse.Parser, args *u.Arguments) error {

	args.InitArgParse(p, args, u.STRING, "p", programArg,
		&argparse.Options{Required: true, Help: "Program name"})
	args.InitArgParse(p, args, u.STRING, "t", testFileArg,
		&argparse.Options{Required: false, Help: "Path of the test file"})
	args.InitArgParse(p, args, u.STRING, "c", configFileArg,
		&argparse.Options{Required: false, Help: "Path of the config file"})
	args.InitArgParse(p, args, u.STRING, "o", optionsArg,
		&argparse.Options{Required: false, Default: "", Help: "Extra options for " +
			"launching program"})

	args.InitArgParse(p, args, u.INT, "w", waitTimeArg,
		&argparse.Options{Required: false, Default: 60, Help: "Time wait (" +
			"sec) for external tests (default: 60 sec)"})

	args.InitArgParse(p, args, u.BOOL, "", saveOutputArg,
		&argparse.Options{Required: false, Default: false,
			Help: "Save results as TXT file and graphs as PNG file"})
	args.InitArgParse(p, args, u.BOOL, "", fullDepsArg,
		&argparse.Options{Required: false, Default: false,
			Help: "Show dependencies of dependencies"})
	args.InitArgParse(p, args, u.BOOL, "", fullStaticAnalysis,
		&argparse.Options{Required: false, Default: false,
			Help: "Full static analysis (analyse shared libraries too)"})
	args.InitArgParse(p, args, u.INT, "", typeAnalysis,
		&argparse.Options{Required: false, Default: 0,
			Help: "Kind of analysis (0: both; 1: static; 2: dynamic)"})

	return u.ParserWrapper(p, os.Args)
}
