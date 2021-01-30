// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package binarytool

import (
	"github.com/akamensky/argparse"
	"os"
	u "tools/srcs/common"
)

const (
	filesArg   = "file"
	mappingArg = "mapping"
	listArg    = "list"
)

// ParseArguments parses arguments of the application.
//
// It returns an error if any, otherwise it returns nil.
func parseLocalArguments(p *argparse.Parser, args *u.Arguments) error {

	args.InitArgParse(p, args, u.BOOL, "m", mappingArg,
		&argparse.Options{Required: false, Default: false,
			Help: "Display libraries mapping (required -l argument)"})

	args.InitArgParse(p, args, u.STRING, "l", listArg,
		&argparse.Options{Required: false, Help: "A list of build directories " +
			"to analyse (sep: ,)"})
	args.InitArgParse(p, args, u.STRING, "f", filesArg,
		&argparse.Options{Required: false, Help: "Json file that contains " +
			"the information for the binary analyser"})

	return u.ParserWrapper(p, os.Args)
}
