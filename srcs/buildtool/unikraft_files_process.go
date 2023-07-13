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

// ---------------------------Create Patches Folder-----------------------------

func createPatchFolder(appFolder string) (*string, error) {

	patchesFolder := appFolder + "patches" + u.SEP
	if _, err := u.CreateFolder(patchesFolder); err != nil {
		return nil, err
	}

	return &patchesFolder, nil
}

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
	for {
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
}

// -------------------------MOVE FILES TO APP FOLDER----------------------------

var srcLanguages = map[string]int{
	".c":   0,
	".cpp": 0,
	".cc":  0,
	/*".S":   0,
	".s":   0,
	".asm": 0,
	".py":  0,
	".go":  0,*/
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

// addAndApplyPatchFiles copies all the user-provided patch files to the unikernel directory,
// conforms them to the unikernel directory format so that all paths in the patch files are paths
// to source files located in the unikernel folder and applies the patches.
//
// It returns an error if any, otherwise it returns nil.
func addAndApplyPatchFiles(patchPath string, patchFolder, appFolder string) error {

	// Copy and conform patch files
	err := filepath.Walk(patchPath, func(filePath string, info os.FileInfo,
		err error) error {
		if !info.IsDir() {
			extension := filepath.Ext(info.Name())
			if extension == ".patch" {
				if err = conformPatchFile(filePath, patchFolder+info.Name()); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Initialise git repo to be able to apply patches
	_, _, _ = u.ExecuteRunCmd("git", appFolder, true, "init")
	_, _, _ = u.ExecuteRunCmd("git", appFolder, true, "add", ".")
	_, _, _ = u.ExecuteRunCmd("git", appFolder, true, "commit", "-m", "first commit")

	// Apply patches
	err = filepath.Walk(patchPath, func(filePath string, info os.FileInfo,
		err error) error {
		if !info.IsDir() {
			_, _, _ = u.ExecuteRunCmd("git", appFolder, true, "am", patchFolder+info.Name())
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// conformPatchFile copies all the user-provided patch files to the unikernel directory and
// conforms them to the unikernel directory format so that all paths in the patch files are paths
// to source files located in the unikernel folder.
//
// It returns an error if any, otherwise it returns nil.
func conformPatchFile(patchPath, newPatchPath string) error {

	patchLines, err := u.ReadLinesFile(patchPath)
	if err != nil {
		return err
	}

	// Find paths in patch file using regexp
	re1 := regexp.MustCompile(`( )(.*)( \| )(.*)`)
	re2 := regexp.MustCompile(`(diff --git )(a/)?(.*)( )(b/)?(.*)`)
	re3 := regexp.MustCompile(`(--- )(a/)?(.*)`)
	re4 := regexp.MustCompile(`(\+\+\+ )(b/)?(.*)`)

	for lineIndex := range patchLines {

		// All paths to files to be modified by the patch are listed under "---"
		if patchLines[lineIndex] == "---\n" {
			lineIndex++
			for ; !strings.Contains(patchLines[lineIndex], "changed"); lineIndex++ {
				for _, match := range re1.FindAllStringSubmatch(patchLines[lineIndex], -1) {
					conformPatchPath(&match, 2)
					patchLines[lineIndex] = strings.Join(match[1:], "") + "\n"
				}
			}
		}

		// All diff lines contain paths to files to be modified by the patch
		if len(patchLines[lineIndex]) > 10 && patchLines[lineIndex][:10] == "diff --git" {
			for _, match := range re2.FindAllStringSubmatch(patchLines[lineIndex], -1) {
				conformPatchPath(&match, 3)
				conformPatchPath(&match, 6)
				patchLines[lineIndex] = strings.Join(match[1:], "") + "\n"
			}

			// Same observation for the two lines following the index line
			for _, match := range re3.FindAllStringSubmatch(patchLines[lineIndex+2], -1) {
				conformPatchPath(&match, 3)
				patchLines[lineIndex+2] = strings.Join(match[1:], "") + "\n"
			}
			for _, match := range re4.FindAllStringSubmatch(patchLines[lineIndex+3], -1) {
				conformPatchPath(&match, 3)
				patchLines[lineIndex+3] = strings.Join(match[1:], "") + "\n"
			}
		}
	}

	// Write the modified content to a file in the unikernel folder
	err = u.WriteToFile(newPatchPath, []byte(strings.Join(patchLines, "")))
	if err != nil {
		return err
	}

	return nil
}

// conformPatchPath conforms a path in a user-provided patch file to the unikernel directory format
// so that this path describes a source file located in the unikernel folder.
func conformPatchPath(match *[]string, index int) {
	extension := filepath.Ext((*match)[index])
	if extension == ".h" || extension == ".hpp" || extension == ".hcc" {
		(*match)[index] = "include" + u.SEP + filepath.Base((*match)[index])
	} else {
		(*match)[index] = filepath.Base((*match)[index])
	}
}

// conformIncludeDirectives conforms all the user-defined include directives of all C/C++ source
// files to the unikernel directory format so that all these directives are paths to source files
// located in the include folder of the unikernel directory.
//
// It returns an error if any, otherwise it returns nil.
func conformIncludeDirectives(sourcePath string) error {

	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo,
		err error) error {
		if !info.IsDir() {
			extension := filepath.Ext(info.Name())
			if extension == ".h" || extension == ".hpp" || extension == ".hcc" {
				if err = conformFile(path, true); err != nil {
					return err
				}
			} else if extension == ".c" || extension == ".cpp" || extension == ".cc" {
				if err = conformFile(path, false); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// conformFile conforms all the user-defined include directives of a C/C++ source file to the
// unikernel directory format so that all these directives are paths to source files located in the
// include folder.
//
// It returns an error if any, otherwise it returns nil.
func conformFile(filePath string, isHeader bool) (err error) {

	fileLines, err := u.ReadLinesFile(filePath)
	if err != nil {
		return err
	}

	// Find include directives using regexp
	re := regexp.MustCompile(`(.*)(#include)(.*)("|<)(.*)("|>)(.*)`)

	for lineIndex := range fileLines {
		for _, match := range re.FindAllStringSubmatch(fileLines[lineIndex], -1) {

			// Only interested in included files not coming from the standard library
			if incDirContainsStdFold(fileLines[lineIndex]) {
				break
			}

			for i := 1; i < len(match); i++ {
				if match[i] == "\"" || match[i] == "<" {

					// Determine the included source file extension to know what the path to it
					// must be in the current file
					var extIsHeader bool
					extension := filepath.Ext(match[i+1])
					if extension == ".h" || extension == ".hpp" || extension == ".hcc" {
						extIsHeader = true
					} else if extension == ".c" || extension == ".cpp" || extension == ".cc" {
						extIsHeader = false
					} else {

						// C++ header
						break
					}

					// Modify the include path
					match[i] = "\""
					if isHeader && !extIsHeader {
						match[i+1] = "../" + filepath.Base(match[i+1])
					} else if !isHeader && extIsHeader {
						match[i+1] = "include" + u.SEP + filepath.Base(match[i+1])
					} else {
						match[i+1] = filepath.Base(match[i+1])
					}
					match[i+2] = "\""
					fileLines[lineIndex] = strings.Join(match[1:], "") + "\n"
					break
				}
			}

			break
		}
	}

	// Write the modified content to a file in the unikernel folder
	err = u.WriteToFile(filePath, []byte(strings.Join(fileLines, "")))
	if err != nil {
		return err
	}

	return nil
}

// incDirContainsStdFold determines if an include directive is a path to a standard header.
//
// It returns true if it is the case, false otherwise.
func incDirContainsStdFold(fileLine string) bool {

	// Standard header list
	stdHeaders := []string{
		"<aio.h>", "<libgen.h>", "<spawn.h>", "<sys/time.h>",
		"<arpa/inet.h>", "<limits.h>", "<stdarg.h>", "<sys/times.h>",
		"<assert.h>", "<locale.h>", "<stdbool.h>", "<sys/types.h>",
		"<complex.h>", "<math.h>", "<stddef.h>", "<sys/uio.h>",
		"<cpio.h>", "<monetary.h>", "<stdint.h>", "<sys/un.h>",
		"<ctype.h>", "<mqueue.h>", "<stdio.h>", "<sys/utsname.h>",
		"<dirent.h>", "<ndbm.h>", "<stdlib.h>", "<sys/wait.h>",
		"<dlfcn.h>", "<net/if.h>", "<string.h>", "<syslog.h>",
		"<errno.h>", "<netdb.h>", "<strings.h>", "<tar.h>",
		"<fcntl.h>", "<netinet/in.h>", "<stropts.h>", "<termios.h>",
		"<fenv.h>", "<netinet/tcp.h>", "<sys/ipc.h>", "<tgmath.h>",
		"<float.h>", "<nl_types.h>", "<sys/mman.h>", "<time.h>",
		"<fmtmsg.h>", "<poll.h>", "<sys/msg.h>", "<trace.h>",
		"<fnmatch.h>", "<pthread.h>", "<sys/resource.h>", "<ulimit.h>",
		"<ftw.h>", "<pwd.h>", "<sys/select.h>", "<unistd.h>",
		"<glob.h>", "<regex.h>", "<sys/sem.h>", "<utime.h>",
		"<grp.h>", "<sched.h>", "<sys/shm.h>", "<utmpx.h>",
		"<iconv.h>", "<search.h>", "<sys/socket.h>", "<wchar.h>",
		"<inttypes.h>", "<semaphore.h>", "<sys/stat.h>", "<wctype.h>",
		"<iso646.h>", "<setjmp.h>", "<sys/statvfs.h>", "<wordexp.h>",
		"<langinfo.h>", "<signal.h>", "<curses.h>", "<term.h>",
		"<uncntrl.h>", "<linux/"}

	for _, header := range stdHeaders {
		if strings.Contains(fileLine, header) {
			return true
		}
	}

	return false
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
				if err = u.CopyFileContents(path, appFolder+info.Name()); err != nil {
					return err
				}
			} else if extension == ".h" || extension == ".hpp" || extension == ".hcc" {
				// Add source files to includesFiles list
				includesFiles = append(includesFiles, info.Name())

				// Copy header files to the INCLUDEFOLDER
				if err = u.CopyFileContents(path, includeFolder+info.Name()); err != nil {
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
