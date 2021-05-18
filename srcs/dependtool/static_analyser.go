// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package dependtool

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"
	u "tools/srcs/common"
)

// ---------------------------------Gather Data---------------------------------

// gatherStaticSymbols gathers symbols of a given application.
//
// It returns an error if any, otherwise it returns nil.
func gatherStaticSymbols(programPath string, data *u.StaticData) error {

	// Use 'readelf' to get symbols
	if output, err := u.ExecuteCommand("readelf", []string{"-s",
		programPath}); err != nil {
		return err
	} else {
		parseReadELF(output, data)
	}
	return nil
}

// gatherStaticSymbols gathers system calls of a given application.
//
// It returns an error if any, otherwise it returns nil.
func gatherStaticSystemCalls(programPath, argument string, data *u.StaticData) error {

	var args []string
	if len(argument) > 0 {
		args = []string{argument, programPath}
	} else {
		args = []string{programPath}
	}

	// Use 'nm' to get symbols and system calls
	if output, err := u.ExecuteCommand("nm", args); err != nil {
		return err
	} else {
		if strings.ToLower(runtime.GOOS) == "linux" {
			parseNMLinux(output, data)
		} else {
			parseNMMac(output, data)
		}
	}
	return nil
}

// gatherStaticSymbols gathers shared libs of a given application.
//
// It returns an error if any, otherwise it returns nil.
func gatherStaticSharedLibsLinux(programPath string, data *u.StaticData,
	v bool) error {

	// Use 'ldd' to get shared libraries
	if output, err := u.ExecutePipeCommand("ldd " + programPath +
		" | awk '/ => / { print $1,$3 }'"); err != nil {
		return err
	} else {
		// Init SharedLibs
		lddGlMap := make(map[string][]string)
		_ = parseLDD(output, data.SharedLibs, lddGlMap, v)
	}
	return nil
}

func gatherStaticSharedLibsMac(programPath string, data *u.StaticData,
	v bool) error {

	// Use 'ldd' to get shared libraries
	if output, err := u.ExecutePipeCommand("otool -L " + programPath +
		" | awk '{ print $1 }'"); err != nil {
		return err
	} else {
		// Init SharedLibs
		lddGlMap := make(map[string][]string)
		_ = parseLDDMac(output, data.SharedLibs, lddGlMap, v)
	}
	return nil
}

// gatherDependencies gathers dependencies of a given application.
//
// It returns an error if any, otherwise it returns nil.
func gatherDependencies(programName string, data *u.StaticData, v bool) error {

	//  Use 'apt-cache pkgnames' to get the name of the package
	output, err := u.ExecuteCommand("apt-cache",
		[]string{"pkgnames", programName})
	if err != nil {
		return err
	}

	// If the name of the package is know, execute apt-cache depends
	if len(output) > 0 {
		// Parse package name
		packageName := parsePackagesName(output)

		if len(packageName) > 0 {
			return executeDependAptCache(packageName, data, v)
		}
	} else {
		// Enter manually the name of the package
		u.PrintWarning(programName + " not found in apt-cache")
		var output string
		for len(output) == 0 {
			fmt.Print("Please enter manually the name of the package " +
				"(empty string to exit): ")
			scanner := bufio.NewScanner(os.Stdin)
			if err := scanner.Err(); err != nil {
				return err
			}

			if scanner.Scan() {

				// Get the new package name
				input := scanner.Text()
				if input == "" {
					break
				}

				output, err = u.ExecuteCommand("apt-cache",
					[]string{"pkgnames", input})
				if err != nil {
					return err
				}
			}
		}

		if len(output) == 0 {
			u.PrintWarning("Skip dependencies analysis from apt-cache depends")
		} else {
			packageName := parsePackagesName(output)
			return executeDependAptCache(packageName, data, v)
		}
	}

	return nil
}

// executeDependAptCache gathers dependencies by executing 'apt-cache depends'.
//
// It returns an error if any, otherwise it returns nil.
func executeDependAptCache(programName string, data *u.StaticData,
	fullDeps bool) error {

	//  Use 'apt-cache depends' to get dependencies
	if output, err := u.ExecutePipeCommand("apt-cache depends " +
		programName + " | awk '/Depends/ { print $2 }'"); err != nil {
		return err
	} else {
		// Init Dependencies (from apt cache depends)
		data.Dependencies = make(map[string][]string)
		dependenciesMap := make(map[string][]string)
		printDep := make(map[string][]string)
		_ = parseDependencies(output, data.Dependencies, dependenciesMap,
			printDep, fullDeps, 0)
	}

	fmt.Println("----------------------------------------------")
	return nil
}

// -------------------------------------Run-------------------------------------

// staticAnalyser runs the static analysis to get shared libraries,
// system calls and library calls of a given application.
//
func staticAnalyser(args u.Arguments, data *u.Data, programPath string) {

	programName := *args.StringArg[programArg]
	fullDeps := *args.BoolArg[fullDepsArg]

	staticData := &data.StaticData

	// If the program is a binary, runs static analysis tools
	if len(programPath) > 0 {
		// Gather Data from binary file

		// Init symbols members
		staticData.Symbols = make(map[string]string)
		staticData.SystemCalls = make(map[string]int)
		staticData.SharedLibs = make(map[string][]string)

		if strings.ToLower(runtime.GOOS) == "linux" {
			u.PrintHeader2("(*) Gathering symbols from binary file")
			if err := gatherStaticSymbols(programPath, staticData); err != nil {
				u.PrintWarning(err)
			}
		}

		u.PrintHeader2("(*) Gathering symbols & system calls from binary file")
		if err := gatherStaticSystemCalls(programPath, "-D", staticData); err != nil {
			// Check without the dynamic argument
			if err := gatherStaticSystemCalls(programPath, "", staticData); err != nil {
				u.PrintWarning(err)
			}
		}

		u.PrintHeader2("(*) Gathering shared libraries from binary file")
		if strings.ToLower(runtime.GOOS) == "linux" {
			if err := gatherStaticSharedLibsLinux(programPath, staticData,
				fullDeps); err != nil {
				u.PrintWarning(err)
			}
		} else {
			if err := gatherStaticSharedLibsMac(programPath, staticData,
				fullDeps); err != nil {
				u.PrintWarning(err)
			}
		}
	}

	if strings.ToLower(runtime.GOOS) == "linux" {
		// Gather Data from apt-cache
		u.PrintHeader2("(*) Gathering dependencies from apt-cache depends")
		if err := gatherDependencies(programName, staticData, fullDeps); err != nil {
			u.PrintWarning(err)
		}
	}
}
