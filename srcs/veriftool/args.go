// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package veriftool

import (
	"github.com/akamensky/argparse"
	"os"
	u "tools/srcs/common"
)

const (
	programArg    = "program"
	testFileArg   = "testFile"
	configFileArg = "configFile"
	optionsArg    = "options"
	unikraftArg   = "unikraft"
)

// ParseArguments parses arguments of the application.
//
// It returns an error if any, otherwise it returns nil.
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

	args.InitArgParse(p, args, u.STRING, "u", unikraftArg,
		&argparse.Options{Required: false, Help: "Unikraft Path"})

	return u.ParserWrapper(p, os.Args)
}
