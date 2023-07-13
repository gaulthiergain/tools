// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package buildtool

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	u "tools/srcs/common"
)

const (
	JSON      = ".json"
	prefixUrl = "https://github.com/unikraft/"
)

type MicroLibFile struct {
	Filename   string
	IsInternal bool
	Functions  []MicroLibsFunction `json:"functions"`
}

type MicroLibsFunction struct {
	Name           string   `json:"name"`
	ReturnValue    string   `json:"return_value"`
	FullyQualified string   `json:"fully_qualified"`
	ArgsName       []string `json:"args_name"`
	ArgsType       []string `json:"args_type"`
	Headers        []string `json:"headers"`
	NbArgs         int      `json:"nb_args"`
	Usage          int      `json:"usage"`
}

// -----------------------------Match micro-libs--------------------------------

// processSymbols adds symbols within the 'exportsyms.uk' file into a map.
func processSymbols(microLib, output string, mapSymbols map[string][]string) {

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if len(line) > 0 && !strings.Contains(line, "#") &&
			strings.Compare(line, "none") != 0 {
			mapSymbols[line] = append(mapSymbols[line], microLib)
		}
	}
}

// fetchSymbolsInternalLibs fetches all symbols within 'exportsyms.uk' files
// from Unikraft's internal libs and add them into a map.
//
// It returns an error if any, otherwise it returns nil.
func fetchSymbolsInternalLibs(folder string,
	microLibs map[string][]string) error {

	files, err := ioutil.ReadDir(folder)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == JSON {
			microLibFile, err := readMicroLibJson(path.Join(folder, file.Name()))
			microLibFile.IsInternal = true
			if err != nil {
				return err
			}
			libName := strings.Replace(file.Name(), JSON, "", -1)
			u.PrintInfo("Retrieving symbols of internal lib: " + libName)
			for _, functions := range microLibFile.Functions {
				microLibs[functions.Name] = append(microLibs[functions.Name], libName)
			}
		}
	}
	return nil
}

// readMicroLibJson reads symbols from external microlibs stored in json files.
//
// It returns a list of MicroLibFile and an error if any, otherwise it returns nil.
func readMicroLibJson(filename string) (*MicroLibFile, error) {

	jsonFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var functions = &MicroLibFile{Filename: filepath.Base(strings.Replace(filename, JSON, "", -1))}
	if err := json.Unmarshal(byteValue, &functions); err != nil {
		return nil, err
	}

	return functions, nil
}

// fetchSymbolsExternalLibs fetches all symbols files from Unikraft's external libs
// and add them into a map.
//
// It returns a list of symbols and an error if any, otherwise it returns nil.
func fetchSymbolsExternalLibs(folder string,
	microLibs map[string][]string) (map[string]string, error) {

	files, err := ioutil.ReadDir(folder)
	if err != nil {
		return nil, err
	}

	externalLibs := make(map[string]string, len(files))
	for _, file := range files {
		if filepath.Ext(file.Name()) == JSON {
			microLibFile, err := readMicroLibJson(path.Join(folder, file.Name()))
			microLibFile.IsInternal = false
			if err != nil {
				return nil, err
			}
			libName := strings.Replace(file.Name(), JSON, "", -1)
			u.PrintInfo("Retrieving symbols of external lib: " + libName)
			for _, functions := range microLibFile.Functions {
				microLibs[functions.Name] = append(microLibs[functions.Name], libName)
			}
			externalLibs[libName] = prefixUrl + libName + ".git"
		}
	}
	return externalLibs, nil
}

// putJsonSymbolsTogether puts the json file symbols and system calls resulting from the static,
// dynamic and source files analyses together into a map structure.
//
// It returns the map containing all the symbols and system calls.
func putJsonSymbolsTogether(data *u.Data) map[string]string {
	dataMap := make(map[string]string)

	for k, v := range data.StaticData.Symbols {
		dataMap[k] = v
	}

	for k := range data.StaticData.SystemCalls {
		dataMap[k] = ""
	}

	for k, v := range data.DynamicData.Symbols {
		dataMap[k] = v
	}

	for k := range data.DynamicData.SystemCalls {
		dataMap[k] = ""
	}

	for k, v := range data.SourcesData.Symbols {
		dataMap[k] = v
	}

	for k := range data.SourcesData.SystemCalls {
		dataMap[k] = ""
	}

	return dataMap
}

// retNameCompat modifies its string argument in order to replace its underscore by a dash when
// necessary.
//
// It returns its string argument whose underscore has been replaced by a dash if necessary,
// otherwise it returns its argument unchanged.
func retNameForCompat(value string) string {
	if strings.Contains(value, "posix-") {
		return strings.ReplaceAll(value, "posix-", "posix_")
	}

	return value
}

// matchSymbols performs the matching between Unikraft's micro-libs and
// libraries used by a given application based on the list of symbols that both
// contain.
//
// It returns a list of micro-libs that are required by the application
func matchSymbols(matchedLibs []string, data map[string]string,
	microLibs map[string][]string) []string {
	for key := range data {
		if values, ok := microLibs[key]; ok {
			for _, value := range values {
				if !u.Contains(matchedLibs, retNameForCompat(value)) {
					matchedLibs = append(matchedLibs, retNameForCompat(value))
				}
			}
		}
	}

	return matchedLibs
}

// matchLibs performs the matching between Unikraft's micro-libs and
// libraries used by a given application
//
// It returns a list of micro-libs that are required by the application and an
// error if any, otherwise it returns nil.
func matchLibs(unikraftLibs string, data *u.Data) ([]string, map[string]string, error) {

	mapSymbols := make(map[string][]string)

	matchedLibs := make([]string, 0)

	folder := filepath.Join(os.Getenv("GOPATH"), "src", "tools", "libs", "internal")
	if err := fetchSymbolsInternalLibs(folder, mapSymbols); err != nil {
		return nil, nil, err
	}

	// Get list of libs from libs/external
	folder = filepath.Join(os.Getenv("GOPATH"), "src", "tools", "libs", "external")
	externalLibs, err := fetchSymbolsExternalLibs(folder, mapSymbols)
	if err != nil {
		return nil, nil, err
	}

	dataMap := putJsonSymbolsTogether(data)

	// Perform the symbol matching
	matchedLibs = matchSymbols(matchedLibs, dataMap, mapSymbols)

	return matchedLibs, externalLibs, nil
}

// -----------------------------Clone micro-libs--------------------------------

// cloneGitRepo clones a specific git repository that hosts an external
// micro-libs on http://github.com/
//
// It returns an error if any, otherwise it returns nil.
func cloneGitRepo(url, unikraftPathLibs, lib string) error {

	u.PrintInfo("Clone git repository " + url)
	if _, _, err := u.GitCloneRepository(url, unikraftPathLibs, true); err != nil {
		return err
	}
	u.PrintOk("Git repository " + url + " has been cloned into " +
		unikraftPathLibs)

	u.PrintInfo("Git branch " + url)
	if _, _, err := u.GitBranchStaging(unikraftPathLibs+lib, false); err != nil {
		return err
	}

	return nil
}

// cloneLibsFolders clones all the needed micro-libs that are needed by a
// given application
func cloneLibsFolders(workspacePath string, matchedLibs []string,
	externalLibs map[string]string) {

	for _, lib := range matchedLibs {
		if value, ok := externalLibs[lib]; ok {
			exists, _ := u.Exists(workspacePath + u.LIBSFOLDER + lib)
			if !exists {
				// If the micro-libs is not in the local host, clone it
				if err := cloneGitRepo(value, workspacePath+u.LIBSFOLDER, lib); err != nil {
					u.PrintWarning(err)
				}
			} else {
				u.PrintInfo("Library " + lib + " already exists in folder" +
					workspacePath + u.LIBSFOLDER)
			}
		}
	}
}
