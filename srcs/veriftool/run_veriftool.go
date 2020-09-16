// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package veriftool

import (
	"errors"
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	u "tools/srcs/common"
)

const stdinCmd = "[STDIN]"
const testCmd = "[TEST]"

func RunVerificationTool() {

	// Init and parse local arguments
	args := new(u.Arguments)
	p, err := args.InitArguments()
	if err != nil {
		u.PrintErr(err)
	}
	if err := parseLocalArguments(p, args); err != nil {
		u.PrintErr(err)
	}

	// Get program Name
	programName := *args.StringArg[programArg]

	// Take base path if absolute path is used
	if filepath.IsAbs(programName) {
		programName = filepath.Base(programName)
	}

	unikraftPath := *args.StringArg[unikraftArg]
	if len(unikraftPath) == 0 {
		u.PrintErr("Unikraft folder must exist! Run the build tool before " +
			"using the verification tool")
	}

	// Get the app folder
	var appFolder string
	if unikraftPath[len(unikraftPath)-1] != os.PathSeparator {
		appFolder = unikraftPath + u.SEP + u.APPSFOLDER + programName + u.SEP
	} else {
		appFolder = unikraftPath + u.APPSFOLDER + programName + u.SEP
	}

	// Get the build folder
	buildAppFolder := appFolder + u.BUILDFOLDER

	// Get KVM image
	var kvmUnikernel string
	if file, err := u.OSReadDir(buildAppFolder); err != nil {
		u.PrintWarning(err)
	} else {
		for _, f := range file {
			if !f.IsDir() && strings.Contains(f.Name(), u.KVM_IMAGE) &&
				len(filepath.Ext(f.Name())) == 0 {
				kvmUnikernel = f.Name()
			}
		}
	}

	// Kvm unikernel image
	if len(kvmUnikernel) == 0 {
		u.PrintWarning(errors.New("no KVM image found"))
	}

	// Read test
	argStdin := ""
	if len(*args.StringArg[testFileArg]) > 0 {

		var err error
		var cmdTests []string
		cmdTests, err = u.ReadLinesFile(*args.StringArg[testFileArg])
		if err != nil {
			u.PrintWarning("Cannot find test files" + err.Error())
		}
		if strings.Contains(cmdTests[0], stdinCmd) {
			argStdin = strings.Join(cmdTests[1:], "")
			argStdin += "\n"
		} else if strings.Contains(cmdTests[0], testCmd) {
			//todo add for other tests
		}
	}

	// Test KVM app unikernel
	unikernelFilename := appFolder + "output_" + kvmUnikernel + ".txt"
	if err := testUnikernel(buildAppFolder+kvmUnikernel, unikernelFilename,
		[]byte(argStdin)); err != nil {
		u.PrintWarning("Impossible to write the output of verification to " +
			unikernelFilename)
	}

	// Test general app
	appFilename := appFolder + "output_" + programName + ".txt"
	if err := testApp(programName, appFilename, []byte(argStdin)); err != nil {
		u.PrintWarning("Impossible to write the output of verification to " +
			unikernelFilename)
	}

	u.PrintInfo("Comparison output:")

	// Compare both output
	fmt.Println(compareOutput(unikernelFilename, appFilename))

}

func compareOutput(unikernelFilename, appFilename string) string {
	f1, err := ioutil.ReadFile(unikernelFilename)
	if err != nil {
		u.PrintErr(err)
	}

	f2, err := ioutil.ReadFile(appFilename)
	if err != nil {
		u.PrintErr(err)
	}

	dmp := diffmatchpatch.New()

	diffs := dmp.DiffMain(string(f2), string(f1), false)

	return dmp.DiffPrettyText(diffs)
}

func testApp(programName, outputFile string, argsStdin []byte) error {
	bOut, _ := u.ExecuteRunCmdStdin(programName, argsStdin)

	return u.WriteToFile(outputFile, bOut)
}

func testUnikernel(kvmUnikernel, outputFile string, argsStdin []byte) error {
	argsQemu := []string{"-nographic", "-vga", "none", "-device",
		"isa-debug-exit", "-kernel", kvmUnikernel}

	bOut, _ := u.ExecuteRunCmdStdin("qemu-system-x86_64", argsStdin, argsQemu...)

	return u.WriteToFile(outputFile, bOut)
}
