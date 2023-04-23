// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package buildtool

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	u "tools/srcs/common"

	"gopkg.in/AlecAivazis/survey.v1"
)

// STATES
const (
	compilerError = iota
	linkingError
	success
)

const pageSize = 10

// -----------------------------Generate Config---------------------------------

// generateConfigUk generates a 'Config.uk' file for the Unikraft build system.
//
// It returns an error if any, otherwise it returns nil.
func generateConfigUk(filename, programName string, matchedLibs []string) error {

	var sb strings.Builder
	sb.WriteString("### Invisible option for dependencies\n" +
		"config APP" + programName + "_DEPENDENCIES\n" + "\tbool\n" +
		"\tdefault y\n")

	for _, lib := range matchedLibs {
		sb.WriteString("\tselect " + lib + "\n")
	}

	// Save the content to Makefile.uk
	return u.WriteToFile(filename, []byte(sb.String()))
}

// ---------------------------Process make output-------------------------------

// checkMakeOutput checks if errors or warning are displayed during the
// execution of the 'make' command.
//
// It returns an integer that defines the result of 'make':
//
//	<SUCCESS, LINKING_ERROR, COMPILER_ERROR>
func checkMakeOutput(appFolder string, stderr *string) int {

	if stderr == nil {
		return success
	}

	// Linking errors during make
	if strings.Contains(*stderr, "undefined") {

		str := parseMakeOutput(*stderr)
		if len(str) > 0 {
			if err := u.WriteToFile(appFolder+"stub.c", []byte(str)); err != nil {
				u.PrintWarning(err)
			}
		}

		return linkingError
	}

	// Compiler errors during make
	if strings.Contains(*stderr, "error:") {

		return compilerError
	}

	return success
}

// parseMakeOutput parses the output of the 'make' command.
//
// It returns a string that contains stubs of undefined function(s).
func parseMakeOutput(output string) string {

	var sb strings.Builder
	sb.WriteString("#include <stdio.h>\n")

	undefinedSymbols := make(map[string]*string)
	var re = regexp.MustCompile(`(?mi).*undefined reference to\s\x60(.*)'`)
	for _, match := range re.FindAllStringSubmatch(output, -1) {
		if _, ok := undefinedSymbols[match[1]]; !ok {
			sb.WriteString("void ")
			sb.WriteString(match[1])
			sb.WriteString("(void){\n\tprintf(\"STUB\\n\");\n}\n\n")
			undefinedSymbols[match[1]] = nil
			u.PrintInfo("Add stub to function: " + match[1])
		}
	}

	return sb.String()
}

// addConfigFiles adds user-provided configuration files to the app unikernel folder.
func addConfigFiles(configFiles []string, selectedFiles *[]string, includeFolder,
	appFolder string) {

	for _, configFilePath := range configFiles {
		configFile := filepath.Base(configFilePath)
		fileExt := filepath.Ext(configFile)

		// Copy config file
		if fileExt == ".h" || fileExt == ".hpp" || fileExt == ".hcc" {
			if err := u.CopyFileContents(configFilePath, includeFolder+configFile); err != nil {
				u.PrintErr(err)
			}
		} else if fileExt == ".c" || fileExt == ".cpp" || fileExt == ".cc" {
			if err := u.CopyFileContents(configFilePath, appFolder+configFile); err != nil {
				u.PrintErr(err)
			}

			// Add Makefile.uk entry
			*selectedFiles = append(*selectedFiles, configFile)
		} else {
			u.PrintWarning("Unsupported extension for file: " + configFile)
		}
	}
}

// -------------------------------------Run-------------------------------------

// RunBuildTool runs the automatic build tool to build a unikernel of a
// given application.
func RunBuildTool(homeDir string, data *u.Data) {

	// Init and parse local arguments
	args := new(u.Arguments)
	p, err := args.InitArguments("--build",
		"The Build tool allows to help developers to port an app as unikernel")
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

	var workspacePath = homeDir + u.SEP + u.WORKSPACEFOLDER
	unikraftPath := workspacePath + u.UNIKRAFTFOLDER
	if len(*args.StringArg[workspaceArg]) > 0 {
		workspacePath = *args.StringArg[workspaceArg]
	}

	// Create workspace folder
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		err = setWorkspaceFolder(workspacePath)
		if err != nil {
			u.PrintErr(err)
		}
	} else {
		u.PrintInfo("Workspace folder already exists")
	}

	// Check if sources argument is set
	if len(*args.StringArg[sourcesArg]) == 0 {
		u.PrintErr("sources argument '-s' must be set")
	}

	// Check if the unikraft folder contains the 3 required folders
	if _, err := ioutil.ReadDir(workspacePath); err != nil {
		u.PrintErr(err)
	} else {
		err := setUnikraftSubFolders(workspacePath)
		if err != nil {
			u.PrintErr(err)
		}
	}

	// If data is not initialized, read output from dependency analysis tool
	if data == nil {
		u.PrintInfo("Initialize data")
		outFolder := homeDir + u.SEP + programName + "_" + u.OUTFOLDER
		if data, err = u.ReadDataJson(outFolder+programName, data); err != nil {
			u.PrintErr(err)
		}
	}

	// Create unikraft application path
	appFolderPtr, err := createUnikraftApp(programName, workspacePath)
	if err != nil {
		u.PrintErr(err)
	}
	appFolder := *appFolderPtr

	// Create the folder 'include' if it does not exist
	includeFolder, err := createIncludeFolder(appFolder)
	if err != nil {
		u.PrintErr(err)
	}

	// Get sources files
	sourcesPath := *args.StringArg[sourcesArg]

	// Copy all .h into the include folder
	sourceFiles, includesFiles := make([]string, 0), make([]string, 0)

	// Move source files to Unikraft folder
	if sourceFiles, err = processSourceFiles(sourcesPath, appFolder, *includeFolder,
		sourceFiles, includesFiles); err != nil {
		u.PrintErr(err)
	}

	// Filter source files to limit build errors (e.g., remove test files,
	// multiple main file, ...)
	filterSourceFiles := filterSourcesFiles(sourceFiles)

	// Prompt file selection
	prompt := &survey.MultiSelect{
		Message:  "Select the sources of the program",
		Options:  sourceFiles,
		Default:  filterSourceFiles,
		PageSize: pageSize,
	}

	var selectedFiles []string
	if err := survey.AskOne(prompt, &selectedFiles, nil); err != nil {
		panic(err)
	}

	// Conform file include directives to the new unikernel folder organisation
	if err := conformIncludeDirectives(appFolder); err != nil {
		u.PrintErr(err)
	}

	// Move config files to the unikernel folder
	addConfigFiles(*args.StringListArg[configArg], &selectedFiles, *includeFolder, appFolder)

	// Match micro-libs
	matchedLibs, externalLibs, err := matchLibs(unikraftPath+"lib"+u.SEP, data)
	if err != nil {
		u.PrintErr(err)
	}

	// Clone the external git repositories
	cloneLibsFolders(workspacePath, matchedLibs, externalLibs)

	// Match internal dependencies between micro-libs
	if err := searchInternalDependencies(workspacePath, &matchedLibs,
		externalLibs); err != nil {
		u.PrintErr(err)
	}

	for _, lib := range matchedLibs {
		u.PrintOk("Match lib: " + lib)
	}

	// Clone the external git repositories (if changed)
	cloneLibsFolders(workspacePath, matchedLibs, externalLibs)

	// Generate Makefiles
	if err := generateMake(programName, appFolder, workspacePath, *args.StringArg[makefileArg],
		matchedLibs, selectedFiles, externalLibs); err != nil {
		u.PrintErr(err)
	}

	// Delete Build folder
	deleteBuildFolder(appFolder)

	// Initialize config files
	initConfig(appFolder, matchedLibs)

	// Run make
	runMake(programName, appFolder)

}

// retFolderCompat modifies its string argument in order to replace its underscore by a dash when
// necessary for the searchInternalDependencies function to find the corresponding folder in the
// 'unikraft' folder.
//
// It returns its string argument whose underscore has been replaced by a dash if necessary,
// otherwise it returns its argument unchanged.
func retFolderForCompat(lib string) string {
	if strings.Contains(lib, "posix_") {
		return strings.ReplaceAll(lib, "posix_", "posix-")
	}

	return lib
}

func searchInternalDependencies(unikraftPath string, matchedLibs *[]string,
	externalLibs map[string]string) error {

	for _, lib := range *matchedLibs {

		// Get and read Config.UK from lib
		var configUk string

		if _, ok := externalLibs[lib]; ok {
			configUk = unikraftPath + u.LIBSFOLDER + lib + u.SEP + "Config.uk"
		} else {
			configUk = unikraftPath + u.UNIKRAFTFOLDER + "lib" + u.SEP + retFolderForCompat(lib) +
				u.SEP + "Config.uk"
		}

		lines, err := u.ReadLinesFile(configUk)
		if err != nil {
			return err
		}

		// Process Config.UK file
		mapConfig := make(map[string][]string)
		u.ProcessConfigUK(lines, true, mapConfig, nil)

		for config := range mapConfig {

			// Remove LIB prefix
			if strings.Contains(config, "LIB") {
				config = strings.TrimPrefix(config, "LIB")
			}

			// Check if matchedLibs already contains the lib
			config = strings.ToLower(config)
			if !u.Contains(*matchedLibs, config) {
				*matchedLibs = append(*matchedLibs, config)
			}
		}
	}

	return nil
}

func generateMake(programName, appFolder, workspacePath, makefile string,
	matchedLibs, sourceFiles []string, externalLibs map[string]string) error {
	// Generate Makefile
	if err := generateMakefile(appFolder+"Makefile", workspacePath,
		appFolder, matchedLibs, externalLibs); err != nil {
		return err
	}

	// Generate Config.uk
	if err := generateConfigUk(appFolder+"Config.uk",
		strings.ToUpper(programName), matchedLibs); err != nil {
		return err
	}

	// Get the file type for Unikraft flag
	fileType := languageUsed()

	// Generate Makefile.uk
	if err := generateMakefileUK(appFolder+"Makefile.uk", programName,
		fileType, makefile, sourceFiles); err != nil {
		return err
	}

	return nil
}

func deleteBuildFolder(appFolder string) {
	// Delete build folder if already exists
	if file, err := u.OSReadDir(appFolder); err != nil {
		u.PrintWarning(err)
	} else {
		for _, f := range file {
			if f.IsDir() && f.Name() == "build" {
				u.PrintWarning("build folder already exists. Delete it.")
				if err := os.RemoveAll(appFolder + "build"); err != nil {
					u.PrintWarning(err)
				}
			}
		}
	}
}

func initConfig(appFolder string, matchedLibs []string) {

	// Run make allNoConfig to generate a .config file
	if strOut, strErr, err := u.ExecuteWaitCommand(appFolder, "make", "allnoconfig"); err != nil {
		u.PrintErr(err)
	} else if len(*strErr) > 0 {
		u.PrintErr("error during generating .config: " + *strErr)
	} else if len(*strOut) > 0 && !strings.Contains(*strOut,
		"configuration written") {
		u.PrintWarning("Default .config cannot be generated")
	}

	// Parse .config
	kConfigMap := make(map[string]*KConfig)
	items := make([]*KConfig, 0)
	items, err := parseConfig(appFolder+".config", kConfigMap, items,
		matchedLibs)
	if err != nil {
		u.PrintErr(err)
	}

	// Update .config
	items = updateConfig(kConfigMap, items)

	// Write .config
	if err := writeConfig(appFolder+".config", items); err != nil {
		u.PrintErr(err)
	}
}

func runMake(programName, appFolder string) {
	// Run make
	stdout, stderr, _ := u.ExecuteRunCmd("make", appFolder, true)

	// Check the state of the make command
	state := checkMakeOutput(appFolder, stderr)
	if state == linkingError {

		// Add new stub.c in Makefile.uk
		d := "APP" + strings.ToUpper(programName) +
			"_SRCS-y += $(APP" + strings.ToUpper(programName) +
			"_BASE)/stub.c"
		if err := u.UpdateFile(appFolder+"Makefile.uk", []byte(d)); err != nil {
			u.PrintErr(err)
		}

		// Run make a second time
		stdout, stderr, _ = u.ExecuteRunCmd("make", appFolder, true)

		// Check the state of the make command
		checkMakeOutput(appFolder, stderr)
	}

	out := appFolder + programName

	// Save make output into warnings.txt if warnings are here
	if stderr != nil && strings.Contains(*stderr, "warning:") {
		if err := u.WriteToFile(out+"_warnings.txt", []byte(*stderr)); err != nil {
			u.PrintWarning(err)
		} else {
			u.PrintInfo("Warnings are written in file: " + out + "_warnings.txt")
		}
	}

	// Save make output into output.txt
	if stdout != nil {
		if err := u.WriteToFile(out+"_output.txt", []byte(*stdout)); err != nil {
			u.PrintWarning(err)
		} else {
			u.PrintInfo("Output is written in file: " + out + "_output.txt")
		}
	}

	if state == compilerError {
		u.PrintErr("Fix compilation errors")
	} else if state == success {
		u.PrintOk("Unikernel created in Folder: " + appFolder)
	}
}
