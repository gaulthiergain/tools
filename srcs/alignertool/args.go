// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package alignertool

import (
	"github.com/akamensky/argparse"
	"os"
	u "tools/srcs/common"
)

const (
	rootArg = "root"
)

// ParseArguments parses arguments of the application.
//
// It returns an error if any, otherwise it returns nil.
func parseLocalArguments(p *argparse.Parser, args *u.Arguments) error {

	args.InitArgParse(p, args, u.STRING, "r", rootArg,
		&argparse.Options{Required: false, Help: "The root folder which contains unikernels directories "})

	return u.ParserWrapper(p, os.Args)
}
