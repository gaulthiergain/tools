package dependtool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	u "tools/srcs/common"
)

// ---------------------------------Gather Data---------------------------------

// findSourcesFiles puts together all C/C++ source files found in a given application folder.
//
// It returns a slice containing the found source file names and an error if any, otherwise it
// returns nil.
func findSourceFiles(workspace string) ([]string, error) {

	var filenames []string

	err := filepath.Walk(workspace,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			ext := filepath.Ext(info.Name())
			if ext == ".c" || ext == ".cpp" || ext == ".cc" || ext == ".h" || ext == ".hpp" ||
				ext == ".hcc" {
				filenames = append(filenames, path)
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return filenames, nil
}

// addSourceFileSymbols adds all the symbols present in 'output' to the static data field in
// 'data'.
func addSourceFileSymbols(output string, data *u.SourcesData) {

	outputTab := strings.Split(output, ",")

	// Get the list of system calls
	systemCalls := initSystemCalls()

	for _, s := range outputTab {
		if _, isSyscall := systemCalls[s]; isSyscall {
			data.SystemCalls[s] = systemCalls[s]
		} else {
			data.Symbols[s] = ""
		}
	}
}

// extractPrototype executes the parserClang.py script on each source file to extracts all possible
// symbols of each of these files.
//
// It returns an error if any, otherwise it returns nil.
func extractPrototype(sourcesFiltered []string, data *u.SourcesData) error {

	for _, f := range sourcesFiltered {
		script := filepath.Join(os.Getenv("GOPATH"), "src", "tools", "srcs", "dependtool",
			"parserClang.py")
		output, err := u.ExecuteCommand("python3", []string{script, "-q", "-t", f})
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
func gatherSourceFileSymbols(data *u.SourcesData, programPath string) error {

	sourceFiles, err := findSourceFiles(programPath)
	if err != nil {
		u.PrintErr(err)
	}
	u.PrintOk(fmt.Sprint(len(sourceFiles)) + " source files found")

	if err := extractPrototype(sourceFiles, data); err != nil {
		u.PrintErr(err)
	}

	return nil
}

// -------------------------------------Run-------------------------------------

// staticAnalyser runs the static analysis to get system calls and library calls of a given
// application.
func sourcesAnalyser(data *u.Data, programPath string) {

	sourcesData := &data.SourcesData

	// Init symbols members
	sourcesData.Symbols = make(map[string]string)
	sourcesData.SystemCalls = make(map[string]int)

	// Detect symbols from source files
	u.PrintHeader2("(*) Gathering symbols from source files")
	if err := gatherSourceFileSymbols(sourcesData, programPath); err != nil {
		u.PrintWarning(err)
	}
}
