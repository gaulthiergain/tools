// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package buildtool

import (
	"os"
	u "tools/srcs/common"

	"github.com/akamensky/argparse"
)

const (
	programArg   = "program"
	workspaceArg = "workspace"
	sourcesArg   = "sources"
	objsArg      = "objects"
	makefileArg  = "makefile"
	configArg    = "config"
	patchArg     = "patch"
)

// ParseArguments parses arguments of the application.
//
// It returns an error if any, otherwise it returns nil.
func parseLocalArguments(p *argparse.Parser, args *u.Arguments) error {

	args.InitArgParse(p, args, u.STRING, "p", programArg,
		&argparse.Options{Required: true, Help: "Program name"})

	args.InitArgParse(p, args, u.STRING, "u", workspaceArg,
		&argparse.Options{Required: false, Help: "Workspace Path"})
	args.InitArgParse(p, args, u.STRING, "s", sourcesArg,
		&argparse.Options{Required: true, Help: "App Sources " +
			"Folder"})
	args.InitArgParse(p, args, u.BOOL, "o", objsArg,
		&argparse.Options{Required: false, Default: false, Help: "Add objects from external" +
			"build system Folder"})
	args.InitArgParse(p, args, u.STRING, "m", makefileArg,
		&argparse.Options{Required: false, Help: "Add additional properties " +
			"for Makefile"})
	args.InitArgParse(p, args, u.STRINGLIST, "c", configArg,
		&argparse.Options{Required: false, Help: "Add configuration files"})
	args.InitArgParse(p, args, u.STRING, "", patchArg,
		&argparse.Options{Required: false, Help: "Add patch files"})

	return u.ParserWrapper(p, os.Args)
}
