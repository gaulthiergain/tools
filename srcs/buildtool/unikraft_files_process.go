// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package buildtool

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	u "tools/srcs/common"
)

// ---------------------------Create Include Folder-----------------------------

func createIncludeFolder(appFolder string) (*string, error) {

	includeFolder := appFolder + u.INCLUDEFOLDER
	if _, err := u.CreateFolder(includeFolder); err != nil {
		return nil, err
	}

	return &includeFolder, nil
}

// ----------------------------Set Workspace Folders-----------------------------
func setWorkspaceFolder(workspacePath string) error {

	_, err := u.CreateFolder(workspacePath)
	if err != nil {
		return err
	}

	return nil
}

func setUnikraftSubFolders(workspaceFolder string) error {

	unikraftFolder := workspaceFolder + u.UNIKRAFTFOLDER
	u.PrintInfo("Managing Unikraft main folder with apps and libs subfolders")

	// Create 'apps' and 'libs' subfolders
	if _, err := u.CreateFolder(workspaceFolder + u.APPSFOLDER); err != nil {
		return err
	}

	if _, err := u.CreateFolder(workspaceFolder + u.LIBSFOLDER); err != nil {
		return err
	}

	if _, err := os.Stat(unikraftFolder); os.IsNotExist(err) {
		url := "https://github.com/unikraft/unikraft.git"
		// Download git repo of unikraft
		if _, _, err := u.GitCloneRepository(url, workspaceFolder, true); err != nil {
			return err
		}
	}
	// Use staging branch
	if _, _, err := u.GitBranchStaging(unikraftFolder, false); err != nil {
		return err
	}

	return nil
}

// ---------------------------Check UNIKRAFT Folder-----------------------------

func containsUnikraftFolders(files []os.FileInfo) bool {

	if len(files) == 0 {
		return false
	}

	m := make(map[string]bool)
	m[u.APPSFOLDER], m[u.LIBSFOLDER], m[u.UNIKRAFTFOLDER] = false, false, false

	var folderName string
	for _, f := range files {
		folderName = f.Name() + u.SEP
		if _, ok := m[folderName]; ok {
			m[folderName] = true
		}
	}

	return m[u.APPSFOLDER] == true && m[u.LIBSFOLDER] && m[u.UNIKRAFTFOLDER]
}

// ---------------------------UNIKRAFT APP FOLDER-------------------------------

func createUnikraftApp(programName, workspacePath string) (*string, error) {

	var appFolder string
	if workspacePath[len(workspacePath)-1] != os.PathSeparator {
		appFolder = workspacePath + u.SEP + u.APPSFOLDER + programName + u.SEP
	} else {
		appFolder = workspacePath + u.APPSFOLDER + programName + u.SEP
	}

	created, err := u.CreateFolder(appFolder)
	if err != nil {
		return nil, err
	}

	if !created {
		u.PrintWarning(appFolder + " already exists.")
		appFolder = handleCreationApp(appFolder)
		if _, err := u.CreateFolder(appFolder); err != nil {
			return nil, err
		}
	}

	return &appFolder, nil
}

// -----------------------------Create App folder-------------------------------

func handleCreationApp(appFolder string) string {
	fmt.Println("Make your choice:\n1: Copy and overwrite files\n2: " +
		"Enter manually the name of the folder\n3: exit program")
	var input int
	for true {
		fmt.Print("Please enter your choice (0 to exit): ")
		if _, err := fmt.Scanf("%d", &input); err != nil {
			u.PrintWarning("Choice must be numeric! Try again")
		} else {
			switch input {
			case 1:
				return appFolder
			case 2:
				fmt.Print("Enter text: ")
				reader := bufio.NewReader(os.Stdin)
				text, _ := reader.ReadString('\n')
				appFolder = strings.Split(text, "\n")[0] + u.SEP
				return appFolder
			case 3:
				os.Exit(1)
			default:
				u.PrintWarning("Invalid input! Try again")
			}
		}
	}

	return appFolder
}

// -------------------------MOVE FILES TO APP FOLDER----------------------------

var srcLanguages = map[string]int{
	".c":   0,
	".cpp": 0,
	".cc":  0,
	".S":   0,
	".s":   0,
	".asm": 0,
	".py":  0,
	".go":  0,
}

func filterSourcesFiles(sourceFiles []string) []string {
	filterSrcFiles := make([]string, 0)
	for _, file := range sourceFiles {
		if !strings.Contains(file, "copy") &&
			!strings.Contains(file, "test") &&
			!strings.Contains(file, "unit") {
			filterSrcFiles = append(filterSrcFiles, file)
		}

	}
	return filterSrcFiles
}

// conformIncDirAndCopyFile conforms all the include directives from a C/C++ source file so that no
// directive contains a path to a header file but the header file name only (i.e., the last element
// of the path). It also copies the content of the source file in the same way as CopyFileContents.
//
// It returns an error if any, otherwise it returns nil.
func conformIncDirAndCopyFile(sourcePath, destPath string) (err error) {

	fileLines, err := u.ReadLinesFile(sourcePath)
	if err != nil {
		return err
	}

	// Find include directives using regexp
	var re = regexp.MustCompile(`(.*)(#include)(.*)(<|")(.*)(>|")(.*)`)

	for index := range fileLines {
		for _, match := range re.FindAllStringSubmatch(fileLines[index], -1) {

			// Only interested in include directives containing a path to a header file
			if !strings.Contains(match[0], "/") {
				continue
			}

			// Replace the path by its last element
			for i := 1; i < len(match); i++ {
				if match[i] == "<" || match[i] == "\"" {
					match[i+1] = filepath.Base(match[i+1])
					fileLines[index] = strings.Join(match[1:], "") + "\n"
					break
				}
			}
		}
	}

	// Write the modified content to a file in the unikernel folder
	err = u.WriteToFile(destPath, []byte(strings.Join(fileLines, "")))
	if err != nil {
		return err
	}

	return nil
}

func processSourceFiles(sourcesPath, appFolder, includeFolder string,
	sourceFiles, includesFiles []string) ([]string, error) {

	err := filepath.Walk(sourcesPath, func(path string, info os.FileInfo,
		err error) error {

		if !info.IsDir() {
			extension := filepath.Ext(info.Name())
			if _, ok := srcLanguages[extension]; ok {
				// Add source files to sourceFiles list
				sourceFiles = append(sourceFiles, info.Name())

				// Count the number of extension
				srcLanguages[extension] += 1

				// Copy source files to the appFolder
				if err = conformIncDirAndCopyFile(path, appFolder+info.Name()); err != nil {
					return err
				}
			} else if extension == ".h" || extension == ".hpp" || extension == ".hcc" {
				// Add source files to includesFiles list
				includesFiles = append(includesFiles, info.Name())

				// Copy header files to the INCLUDEFOLDER
				if err = conformIncDirAndCopyFile(path, includeFolder+info.Name()); err != nil {
					return err
				}
			} else {
				u.PrintWarning("Unsupported extension for file: " + info.Name())
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// If no source file, exit the program
	if len(sourceFiles) == 0 {
		return nil, errors.New("unable to find source files")
	}

	return sourceFiles, nil
}

func languageUsed() string {

	max := -1
	var mostUsedFiles string
	for key, value := range srcLanguages {
		if max < value {
			max = value
			mostUsedFiles = key
		}
	}

	return mostUsedFiles
}
