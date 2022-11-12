// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package dependtool

import (
	"bufio"
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	u "tools/srcs/common"
)

// ---------------------------------Gather Data---------------------------------

func addSymbols(symbols []elf.Symbol, data *u.StaticData, systemCalls map[string]int) {
	for _, s := range symbols {
		if _, isSyscall := systemCalls[s.Name]; isSyscall {
			data.SystemCalls[s.Name] = systemCalls[s.Name]
		} else {
			data.Symbols[s.Name] = s.Library
		}
	}
}

// gatherStaticSymbols gathers symbols of a given application.
//
// It returns an error if any, otherwise it returns nil.
func gatherStaticSymbols(elfFile *elf.File, dynamicCompiled bool, data *u.StaticData) error {

	// Get the list of system calls
	systemCalls := initSystemCalls()

	symbols, err := elfFile.Symbols()
	if err != nil {
		if !dynamicCompiled {
			// Skip message for dynamically compiled binary
			u.PrintWarning(err)
		}
	} else {
		addSymbols(symbols, data, systemCalls)
	}

	// Additional symbols when dynamically compiled
	if dynamicCompiled {

		symbols, err = elfFile.DynamicSymbols()
		if err != nil {
			u.PrintWarning(err)
		} else {
			addSymbols(symbols, data, systemCalls)
		}

		importedSymbols, err := elfFile.ImportedSymbols()
		if err != nil {
			u.PrintWarning(err)
		} else {
			for _, s := range importedSymbols {
				if _, isSyscall := systemCalls[s.Name]; isSyscall {
					data.SystemCalls[s.Name] = systemCalls[s.Name]
				} else {
					data.Symbols[s.Name] = s.Library
				}
			}
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
			if err := executeDependAptCache(packageName, data, v); err != nil {
				u.PrintWarning(err)
			}
			if _, ok := data.Dependencies[packageName]; !ok {
				data.Dependencies[packageName] = []string{""}
			}
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
			printDep, fullDeps, 5)

	}

	fmt.Println("----------------------------------------------")
	return nil
}

// findSourcesFiles puts together all C/C++ source files found in a given application folder.
//
// It returns a slice containing the found source file names and an error if any, otherwise it
// returns nil.
func findSourcesFiles(workspace string) ([]string, error) {

	var filenames []string

	err := filepath.Walk(workspace,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			ext := filepath.Ext(info.Name())
			if ext == ".c" || ext == ".cpp" {
				filenames = append(filenames, path)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return filenames, nil
}

// TODO REPLACE
// ExecuteCommand a single command without displaying the output.
//
// It returns a string which represents stdout and an error if any, otherwise
// it returns nil.
func ExecuteCommand(command string, arguments []string) (string, error) {
	out, err := exec.Command(command, arguments...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// addSourceFileSymbols adds all the symbols present in 'output' to the static data field in
// 'data'.
func addSourceFileSymbols(output string, data *u.Data) {
	outputTab := strings.Split(output, ",")

	// Get the list of system calls
	systemCalls := initSystemCalls()

	for _, s := range outputTab {
		if _, isSyscall := systemCalls[s]; isSyscall {
			data.StaticData.SystemCalls[s] = systemCalls[s]
		} else {
			data.StaticData.Symbols[s] = ""
		}
	}
}

// extractPrototype executes the parserClang.py script on each source file to extracts all possible
// symbols of each of these files.
//
// It returns an error if any, otherwise it returns nil.
func extractPrototype(sourcesFiltered []string, data *u.Data) error {

	for _, f := range sourcesFiltered {
		script := filepath.Join(os.Getenv("GOPATH"), "src", "tools", "srcs", "dependtool",
			"parserClang.py")
		output, err := ExecuteCommand("python3", []string{script, "-q", "-t", f})
		if err != nil {
			u.PrintWarning("Incomplete analysis with file " + f)
			continue
		}
		addSourceFileSymbols(output, data)
	}
	return nil
}

// gatherSourceFileSymbols gathers symbols of source files from a given application folder.
//
// It returns an error if any, otherwise it returns nil.
func gatherSourceFileSymbols(data *u.Data, programPath string) error {

	tmp := strings.Split(programPath, "/")
	i := 2
	for ; i < len(tmp); i++ {
		if tmp[len(tmp)-i] == "apps" {
			break
		}
	}
	folderPath := strings.Join(tmp[:len(tmp)-i+2], "/")

	files, err := findSourcesFiles(folderPath)
	if err != nil {
		u.PrintErr(err)
	}

	if err := extractPrototype(files, data); err != nil {
		u.PrintErr(err)
	}
	return nil
}

// -------------------------------------Run-------------------------------------

// staticAnalyser runs the static analysis to get shared libraries,
// system calls and library calls of a given application.
func staticAnalyser(elfFile *elf.File, isDynamic, isLinux bool, args u.Arguments, data *u.Data,
	programPath string) {

	programName := *args.StringArg[programArg]
	fullDeps := *args.BoolArg[fullDepsArg]
	fullStaticAnalysis := *args.BoolArg[fullStaticAnalysis]

	staticData := &data.StaticData

	// If the program is a binary, runs static analysis tools
	if len(programPath) > 0 {

		// Init symbols members
		staticData.Symbols = make(map[string]string)
		staticData.SystemCalls = make(map[string]int)
		staticData.SharedLibs = make(map[string][]string)

		if isLinux {
			// Gather Data from binary file
			u.PrintHeader2("(*) Gathering symbols from binary file")
			if err := gatherStaticSymbols(elfFile, isDynamic, staticData); err != nil {
				u.PrintWarning(err)
			}
		}

		u.PrintHeader2("(*) Gathering shared libraries from binary file")
		if isLinux {
			// Cannot use "elfFile.ImportedLibraries()" since we need the ".so" path
			// So in that case, we need to rely on ldd
			if err := gatherStaticSharedLibsLinux(programPath, staticData,
				fullDeps); err != nil {
				u.PrintWarning(err)
			}

			if err := elfFile.Close(); err != nil {
				u.PrintWarning(err)
			}
		} else {
			if err := gatherStaticSharedLibsMac(programPath, staticData,
				fullDeps); err != nil {
				u.PrintWarning(err)
			}
		}
	}

	// Detect symbols from source files
	u.PrintHeader2("(*) Gathering symbols from source files")
	if err := gatherSourceFileSymbols(data, programPath); err != nil {
		u.PrintWarning(err)
	}

	// Detect symbols from shared libraries
	if fullStaticAnalysis && isLinux {
		u.PrintHeader2("(*) Gathering symbols and system calls of shared libraries from binary" +
			"file")
		for key, path := range staticData.SharedLibs {
			if len(path) > 0 {
				fmt.Printf("\t-> Analysing %s - %s\n", key, path[0])
				libElf, err := getElf(path[0])
				if err != nil {
					u.PrintWarning(err)
				}
				if err := gatherStaticSymbols(libElf, true, staticData); err != nil {
					u.PrintWarning(err)
				}
				if err := libElf.Close(); err != nil {
					u.PrintWarning(err)
				}
			}
		}
	}

	if isLinux {
		// Gather Data from apt-cache
		u.PrintHeader2("(*) Gathering dependencies from apt-cache depends")
		if err := gatherDependencies(programName, staticData, fullDeps); err != nil {
			u.PrintWarning(err)
		}
	}
}
