// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package extractertool

import (
	"github.com/akamensky/argparse"
	"os"
	u "tools/srcs/common"
)

const (
	library      = "library"
	workspaceArg = "workspace"
)

// parseLocalArguments parses arguments of the application.
func parseLocalArguments(p *argparse.Parser, args *u.Arguments) error {

	args.InitArgParse(p, args, u.STRING, "l", library,
		&argparse.Options{Required: true, Help: "Library name"})
	args.InitArgParse(p, args, u.STRING, "u", workspaceArg,
		&argparse.Options{Required: false, Help: "Workspace Path"})
	return u.ParserWrapper(p, os.Args)
}
