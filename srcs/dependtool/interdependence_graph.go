package dependtool

import (
	"fmt"
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

// findSourceFilesAndFolders puts together all the application C/C++ source files found on one hand
// and all the application (sub-)folder paths on the other hand.
//
// It returns two slices: one containing the found source file paths and one containing the found
// (sub-)folder paths, and an error if any, otherwise it returns nil.
func findSourceFilesAndFolders(workspace string) ([]string, []string, error) {

	var filenames []string
	var foldernames []string

	err := filepath.Walk(workspace,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Ignore hidden elements
			if string(info.Name()[0]) == "." {
				return nil
			}

			// Found (sub-)folder
			if info.IsDir() {
				foldernames = append(foldernames, "-I")
				foldernames = append(foldernames, path)
				return nil
			}

			// Found source file
			ext := filepath.Ext(info.Name())
			if ext == ".c" || ext == ".cpp" || ext == ".cc" || ext == ".h" || ext == ".hpp" ||
				ext == ".hcc" {
				filenames = append(filenames, path)
			}
			return nil
		})
	if err != nil {
		return nil, nil, err
	}
	return filenames, foldernames, nil
}

// checkIncDir checks that an inclusion directive path does not contain the two dots ".." which
// refer to the parent directory and that can cause trouble when pruning the map and generating the
// graph. If the path does contain these dots, it is simplified.
//
// It returns the simplified directive path or the path itself if it contains no dots.
func checkIncDir(directive string) string {

	// No problem
	if !strings.Contains(directive, "../") {
		return directive
	}

	// Must be simplified
	splitDir := strings.Split(directive, u.SEP)
	var i int
	for i = 0; i < len(splitDir); i++ {
		if splitDir[i] == ".." {

			// Dots at the beginning of the path
			if i == 0 {
				splitDir = splitDir[1:]
				i--

				// Dots somewhere else in the path
			} else {
				splitDir = append(splitDir[:i-1], splitDir[i+1:]...)
				i -= 2
			}
		}
	}
	return strings.Join(splitDir, u.SEP)
}

// sourceFileIncludesAnalysis collects all the inclusion directives from a C/C++ source file.
//
// It returns a slice containing all the relative paths contained in the inclusion directives.
func sourceFileIncludesAnalysis(sourceFile string) []string {

	var fileIncDir []string

	fileLines, err := u.ReadLinesFile(sourceFile)
	if err != nil {
		u.PrintErr(err)
	}

	// Find inclusion directives using regexp
	var re = regexp.MustCompile(`(.*)(#include)(.*)("|<)(.*)("|>)(.*)`)

	for lineIndex := range fileLines {
		for _, match := range re.FindAllStringSubmatch(fileLines[lineIndex], -1) {

			// Append the relative path to the list of relative paths
			for i := 1; i < len(match); i++ {
				if match[i] == "\"" || match[i] == "<" {
					fileIncDir = append(fileIncDir, checkIncDir(match[i+1]))
					break
				}
			}
		}
	}
	return fileIncDir
}

// TODO REPLACE
// ExecuteCommand a single command without displaying the output.
//
// It returns a string which represents stdout and an error if any, otherwise
// it returns nil.
func executeCommand(command string, arguments []string) (string, error) {

	out, err := exec.Command(command, arguments...).CombinedOutput()
	return string(out), err
}

// gppSourceFileIncludesAnalysis collects all the inclusion directives from a C/C++ source file
// using the gpp preprocessor.
//
// It returns a slice containing all the absolute paths contained in the inclusion directives.
func gppSourceFileIncludesAnalysis(sourceFile, programPath string,
	sourceFolders []string) ([]string, error) {

	var fileIncDir []string

	// g++ command
	outputStr, err := executeCommand("g++", append([]string{"-E", sourceFile}, sourceFolders...))

	// If command g++ returns an error, prune file: it contains non-standard libraries
	if err != nil {
		return nil, err
	}

	// Find inclusion directive paths
	outputSlice := strings.Split(outputStr, "\n")
	for _, line := range outputSlice {

		// If warnings or errors are present, ignore their paths
		if strings.Contains(line, ":") {
			continue
		}

		// Only interested in file paths not coming from the standard library
		if strings.Contains(line, programPath) {
			includeDirective :=
				checkIncDir(line[strings.Index(line, "\"")+1 : strings.LastIndex(line, "\"")])
			if !u.Contains(fileIncDir, includeDirective) {
				fileIncDir = append(fileIncDir, includeDirective)
			}
		}
	}
	return fileIncDir, nil
}

// pruneRemovableFiles prunes interdependence graph elements if the latter are unused header files.
func pruneRemovableFiles(interdependMap *map[string][]string) {

	for file := range *interdependMap {

		// No removal of C/C++ source files
		if filepath.Ext(file) != ".c" && filepath.Ext(file) != ".cpp" &&
			filepath.Ext(file) != ".cc" {

			// Lookup for files depending on the current header file
			depends := false
			for _, dependencies := range *interdependMap {
				for _, dependency := range dependencies {
					if file == dependency {
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
				delete(*interdependMap, file)
			}
		}
	}
}

// pruneElemFiles prunes interdependence graph elements if the latter contain the substring in
// argument.
func pruneElemFiles(interdependMap *map[string][]string, pruneElem string) {

	// Lookup for key elements containing the substring and prune them
	for key := range *interdependMap {
		if strings.Contains(key, pruneElem) {
			delete(*interdependMap, key)

			// Lookup for key elements that depend on the key found above and prune them
			for file, dependencies := range *interdependMap {
				for _, dependency := range dependencies {
					if dependency == key {
						pruneElemFiles(interdependMap, file)
					}
				}
			}
		}
	}
}

// requestUnikraftExtLibs collects all the GitHub repositories of Unikraft through the GitHub API
// and returns the whole list of Unikraft external libraries.
//
// It returns a slice containing all the libraries names.
func requestUnikraftExtLibs() []string {

	var extLibsList, appsList []string

	// Only 2 Web pages of repositories as for february 2023 (125 repos - 100 repos per page)
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

		// Collect libraries
		fileLines := strings.Split(string(body), "\"name\":\"lib-")
		for i := 1; i < len(fileLines); i++ {
			extLibsList = append(extLibsList, fileLines[i][0:strings.Index(fileLines[i][0:],
				"\"")])
		}

		// Collect applications
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

// runInterdependAnalyser collects all the inclusion directives (i.e., dependencies) from each
// C/C++ application source file and builds an interdependence graph (dot file and png image)
// between the source files that it deemed compilable with Unikraft.
//
// It returns the path to the folder containing the source files that have been kept for generating
// the graph.
func interdependAnalyser(programPath, programName, outFolder string) string {

	// Find all application source files and (sub-)folders
	sourceFiles, sourceFolders, err := findSourceFilesAndFolders(programPath)
	if err != nil {
		u.PrintErr(err)
	}
	u.PrintOk(fmt.Sprint(len(sourceFiles)) + " source files found")
	u.PrintOk(fmt.Sprint(len(sourceFolders)) + " source subfolders found")

	// Collect source file inclusion directives. Source files are first analysed by g++ to make
	// sure to avoid directives that are commented or subjected to a macro and then "by hand" to
	// sort the include directives of the g++ analysis (i.e. to avoid inclusion directives that are
	// not present in the source file currently analysed).
	interdependMap := make(map[string][]string)
	for _, sourceFile := range sourceFiles {
		analysis := sourceFileIncludesAnalysis(sourceFile)
		gppAnalysis, err := gppSourceFileIncludesAnalysis(sourceFile, programPath, sourceFolders)
		if err != nil {
			continue
		}

		// Build the interdependence map with the paths contained in the inclusion directives
		interdependMap[sourceFile] = make([]string, 0)
		for _, directive := range analysis {
			for _, gppDirective := range gppAnalysis {
				if strings.Contains(gppDirective, directive) {
					interdependMap[sourceFile] = append(interdependMap[sourceFile], gppDirective)
				}
			}
		}
	}

	// Prune the interdependence graph
	extLibsList := requestUnikraftExtLibs()
	extLibsList = append(extLibsList, "win32", "test", "TEST")
	for _, extLib := range extLibsList {
		pruneElemFiles(&interdependMap, extLib)
	}
	pruneRemovableFiles(&interdependMap)

	// Create a folder and copy all the kept source files into it for later use with build tool
	outAppFolder := outFolder + programName + u.SEP
	_, err = u.CreateFolder(outAppFolder)
	if err != nil {
		u.PrintErr(err)
	}
	for _, sourceFile := range sourceFiles {
		if _, ok := interdependMap[sourceFile]; ok {
			if err := u.CopyFileContents(sourceFile,
				outAppFolder+filepath.Base(sourceFile)); err != nil {
				u.PrintErr(err)
			}
		}
	}
	u.PrintOk(fmt.Sprint(len(interdependMap)) + " source files kept and copied to " + outAppFolder)

	// Change the absolute paths in the interdependence map into relative paths for more
	// readability in the png image
	graphMap := make(map[string][]string)
	for appFilePath := range interdependMap {
		appFile := strings.Split(appFilePath, programPath)[1][1:]
		graphMap[appFile] = make([]string, 0)
		for _, fileDepPath := range interdependMap[appFilePath] {
			graphMap[appFile] = append(graphMap[appFile], strings.Split(fileDepPath,
				programPath)[1][1:])
		}
	}

	// Create dot and png files
	u.GenerateGraph(programName, outFolder+programName, graphMap, nil)

	return outAppFolder
}
