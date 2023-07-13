// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package veriftool

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	u "tools/srcs/common"
	"tools/srcs/dependtool"
)

func RunVerificationTool(homeDir string) {

	// Init and parse local arguments
	args := new(u.Arguments)
	p, err := args.InitArguments("--verif",
		"The Output verifier tool allows to compare the output of a program ported as unikernel")
	if err != nil {
		u.PrintErr(err)
	}
	if err := parseLocalArguments(p, args); err != nil {
		u.PrintErr(err)
	}

	// Get program path
	programPath, err := u.GetProgramPath(&*args.StringArg[programArg])
	if err != nil {
		u.PrintErr("Could not determine program path", err)
	}

	// Get program Name
	programName := *args.StringArg[programArg]

	// Take base path if absolute path is used
	if filepath.IsAbs(programName) {
		programName = filepath.Base(programName)
	}

	var workspacePath = homeDir + u.SEP + u.WORKSPACEFOLDER
	if len(*args.StringArg[workspaceArg]) > 0 {
		workspacePath = *args.StringArg[workspaceArg]
	}

	// Get the app folder
	var appFolder string
	if workspacePath[len(workspacePath)-1] != os.PathSeparator {
		appFolder = workspacePath + u.SEP + u.APPSFOLDER + programName + u.SEP
	} else {
		appFolder = workspacePath + u.APPSFOLDER + programName + u.SEP
	}

	// Get the build folder
	buildAppFolder := appFolder + u.BUILDFOLDER

	// Get KVM image
	var kvmUnikernelPath, kvmUnikernel string
	if file, err := u.OSReadDir(buildAppFolder); err != nil {
		u.PrintWarning(err)
	} else {
		for _, f := range file {
			if !f.IsDir() && strings.Contains(f.Name(), u.KVM_IMAGE) &&
				len(filepath.Ext(f.Name())) == 0 {
				kvmUnikernel = f.Name()
				kvmUnikernelPath = filepath.Join(buildAppFolder, f.Name())
			}
		}
	}

	// Kvm unikernel image
	if len(kvmUnikernel) == 0 {
		u.PrintWarning(errors.New("no KVM image found"))
	}

	// Filepath of output
	unikernelFilename := appFolder + "output_" + kvmUnikernel + ".txt"
	appFilename := appFolder + "output_" + programName + ".txt"

	// Read test
	argStdin := ""
	if len(*args.StringArg[testFileArg]) > 0 {
		testingStruct := &dependtool.Testing{}
		if len(*args.StringArg[testFileArg]) > 0 {
			var err error
			testingStruct, err = dependtool.ReadTestFileJson(*args.StringArg[testFileArg])
			if err != nil {
				u.PrintWarning("Cannot find test file: " + err.Error())
			}
		}
		option := ""
		if len(*args.StringArg[optionsArg]) > 0 {
			option = *args.StringArg[optionsArg]
		}

		str_b := ""
		for i := 0; i < 45; i++ {
			str_b += "\n"
		}

		outStr, _ := dependtool.RunVerifCommandTester(programPath, programName, option, testingStruct)
		if err := u.WriteToFile(appFilename, []byte(str_b+outStr)); err != nil {
			u.PrintWarning("Impossible to write the output of verification to " +
				appFilename)
		} else {
			u.PrintInfo("Output of general application written to " + appFilename)
		}

		option = "-nographic -vga none -device isa-debug-exit -kernel " + kvmUnikernelPath

		outStr, _ = dependtool.RunVerifCommandTester("qemu-system-x86_64", "qemu-system-x86_64", option, testingStruct)
		if err := u.WriteToFile(unikernelFilename, []byte(str_b+outStr)); err != nil {
			u.PrintWarning("Impossible to write the output of verification to " +
				kvmUnikernel)
		} else {
			u.PrintInfo("Output of unikernel written to " + unikernelFilename)
		}

	} else {
		// No test file
		if err := testUnikernel(buildAppFolder+kvmUnikernel, unikernelFilename,
			[]byte(argStdin)); err != nil {
			u.PrintWarning("Impossible to write the output of verification to " +
				unikernelFilename)
		}

		// Test general app
		if err := testApp(programName, appFilename, []byte(argStdin)); err != nil {
			u.PrintWarning("Impossible to write the output of verification to " +
				appFilename)
		} else {
			u.PrintInfo("Output of general application writtent to " + appFilename)
		}
	}

	c := askForConfirmation("Do you want to see a diff between the two output")
	if c {
		u.PrintInfo("Comparison output:")

		// Compare both output
		fmt.Println(compareOutput(unikernelFilename, appFilename))
	}
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

// askForConfirmation asks the user for confirmation. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user.
func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}
