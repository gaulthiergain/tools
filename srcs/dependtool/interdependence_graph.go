package dependtool

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	u "tools/srcs/common"
)

// ---------------------------------Gather Data---------------------------------

// sourceFileIncludesAnalysis collects all the include directives from a C/C++ source file.
//
// It returns a slice containing the found header names.
func sourceFileIncludesAnalysis(sourceFile string) []string {

	var fileIncludes []string

	fileLines, err := u.ReadLinesFile(sourceFile)
	if err != nil {
		u.PrintErr(err)
	}

	// Find user-defined include directives using regexp
	var re = regexp.MustCompile(`(.*)(#include)(.*)("|<)(.*)("|>)(.*)`)

	for lineIndex := range fileLines {
		for _, match := range re.FindAllStringSubmatch(fileLines[lineIndex], -1) {

			// Find the last element of the path and append it to the list of found header names
			for i := 1; i < len(match); i++ {
				if match[i] == "\"" || match[i] == "<" {
					fileIncludes = append(fileIncludes, filepath.Base(match[i+1]))
					break
				}
			}
		}
	}

	return fileIncludes
}

// TODO REPLACE
// ExecuteCommand a single command without displaying the output.
//
// It returns a string which represents stdout and an error if any, otherwise
// it returns nil.
func ExecuteCommand(command string, arguments []string) (string, error) {
	out, err := exec.Command(command, arguments...).CombinedOutput()
	return string(out), err
}

// gccSourceFileIncludesAnalysis collects all the include directives from a C/C++ source file using
// the gcc preprocessor.
//
// It returns a slice containing the found header names.
func gccSourceFileIncludesAnalysis(sourceFile, outAppFolder string) ([]string, error) {

	var fileIncludes []string

	// gcc command
	outputStr, err := ExecuteCommand("gcc", []string{"-E", sourceFile, "-I", outAppFolder})

	// if gcc command returns an error, prune file: it contains non-standard libs
	if err != nil {
		return make([]string, 0), err
	}
	outputSlice := strings.Split(outputStr, "\n")

	for _, line := range outputSlice {

		// Only interested in headers not coming from the standard library
		if strings.Contains(line, "\""+outAppFolder) {
			line = strings.Split(line, "\""+outAppFolder)[1]
			includeDirective := filepath.Base(line[0:strings.Index(line[0:], "\"")])
			if !u.Contains(fileIncludes, includeDirective) {
				fileIncludes = append(fileIncludes, includeDirective)
			}
		}
	}

	return fileIncludes, nil
}

// pruneRemovableFiles prunes interdependence graph elements if the latter are unused header files.
func pruneRemovableFiles(interdependMap *map[string][]string) {

	for internalFile := range *interdependMap {

		// No removal of C/C++ source files
		if filepath.Ext(internalFile) != ".c" && filepath.Ext(internalFile) != ".cpp" &&
			filepath.Ext(internalFile) != ".cc" {

			// Lookup for files depending on the current header file
			depends := false
			for _, dependencies := range *interdependMap {
				for _, dependency := range dependencies {
					if internalFile == dependency {
						depends = true
						break
					}
				}
				if depends {
					break
				}
			}

			// Prune header file if unused
			if !depends {
				delete(*interdependMap, internalFile)
			}
		}
	}
}

// pruneElemFiles prunes interdependence graph elements if the latter contain the substring in
// argument.
func pruneElemFiles(interdependMap *map[string][]string, pruneElem string) {

	// Lookup for key elements containing the substring and prune them
	for internalFile := range *interdependMap {
		if strings.Contains(internalFile, pruneElem) {
			delete(*interdependMap, internalFile)

			// Lookup for key elements that depend on the key found above and prune them
			for file, dependencies := range *interdependMap {
				for _, dependency := range dependencies {
					if dependency == internalFile {
						pruneElemFiles(interdependMap, file)
					}
				}
			}
		}
	}
}

// requestUnikraftExtLibs collects all the GitHub repositories of Unikraft through the GitHub API
// and returns the whole list of Unikraft external libraries.
func requestUnikraftExtLibs() []string {

	var extLibsList, appsList []string

	// Only 2 Web pages of repos as for february 2023 (125 repos - 100 repos per page)
	nbPages := 2

	for i := 1; i <= nbPages; i++ {

		// HTTP Get request
		resp, err := http.Get("https://api.github.com/orgs/unikraft/repos?page=" +
			strconv.Itoa(i) + "&per_page=100")
		if err != nil {
			u.PrintErr(err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			u.PrintErr(err)
		}

		// Collect libs
		fileLines := strings.Split(string(body), "\"name\":\"lib-")
		for i := 1; i < len(fileLines); i++ {
			extLibsList = append(extLibsList, fileLines[i][0:strings.Index(fileLines[i][0:],
				"\"")])
		}

		// Collect apps
		fileLines = strings.Split(string(body), "\"name\":\"app-")
		for i := 1; i < len(fileLines); i++ {
			appsList = append(appsList, fileLines[i][0:strings.Index(fileLines[i][0:],
				"\"")])
		}
	}

	// Avoid libs that are also apps (e.g. nginx, redis)
	for i, lib := range extLibsList {
		if u.Contains(appsList, lib) {
			extLibsList = append(extLibsList[:i], extLibsList[i+1:]...)
		}
	}
	return extLibsList
}

// -------------------------------------Run-------------------------------------

// runInterdependAnalyser collects all the included headers names (i.e., dependencies) from each
// C/C++ source file of a program and builds an interdependence graph (dot file) between all these
// source files.
func interdependAnalyser(programPath, programName, outFolder string) string {

	// Find all program source files
	sourceFiles, err := findSourcesFiles(programPath)
	if err != nil {
		u.PrintErr(err)
	}

	// Create a folder and copy all source files into it for use with the gcc preprocessor
	outAppFolder := outFolder + programName + u.SEP
	_, err = u.CreateFolder(outAppFolder)
	if err != nil {
		u.PrintErr(err)
	}
	var outAppFiles []string
	for _, sourceFilePath := range sourceFiles {
		if err := u.CopyFileContents(sourceFilePath,
			outAppFolder+filepath.Base(sourceFilePath)); err != nil {
			u.PrintErr(err)
		}
		outAppFiles = append(outAppFiles, outAppFolder+filepath.Base(sourceFilePath))
	}

	// Analyse source files include directives and collect header names. Source files are first
	// analysed by gcc to make sure to avoid directives that are commented or subjected to a macro
	// and then "by hand" to sort the include directives of the gcc analysis (i.e., to avoid
	// include directives that are not present in the source file currently analysed).
	interdependMap := make(map[string][]string)
	for _, outAppFile := range outAppFiles {
		analysis := sourceFileIncludesAnalysis(outAppFile)
		gccAnalysis, err := gccSourceFileIncludesAnalysis(outAppFile, outAppFolder)
		if err != nil {
			continue
		}

		interdependMap[filepath.Base(outAppFile)] = make([]string, 0)
		for _, includeDirective := range gccAnalysis {
			if u.Contains(analysis, includeDirective) {
				interdependMap[filepath.Base(outAppFile)] =
					append(interdependMap[filepath.Base(outAppFile)], includeDirective)
			}
		}
	}

	// Prune interdependence graph
	extLibsList := requestUnikraftExtLibs()
	extLibsList = append(extLibsList, "win32", "test", "TEST")
	for _, extLib := range extLibsList {
		pruneElemFiles(&interdependMap, extLib)
	}
	pruneRemovableFiles(&interdependMap)

	// Remove pruned files from the out app folder
	for _, outAppFile := range outAppFiles {
		if _, ok := interdependMap[filepath.Base(outAppFile)]; !ok {
			os.Remove(outAppFile)
		}
	}

	// Create dot file
	u.GenerateGraph(programName, outFolder+programName, interdependMap, nil)

	return outAppFolder
}
