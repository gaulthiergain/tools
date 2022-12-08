package extractertool

import (
	"errors"
	"io"
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
import "C"

const HTTP = "http"
const URL = "URL"

type MicroLibFile struct {
	Functions []MicroLibsFunction `json:"functions"`
}

type MicroLibsFunction struct {
	Name string `json:"name"`
}

type Variables struct {
	name  string
	value string
}

func getMakefileSources(content string, mapSources map[string]string) {
	var re = regexp.MustCompile(`(?m)\/.*\.c|\/.*\.h|\/.*\.cpp|\/.*\.hpp|\/.*\.cc|\/.*\.hcc`)
	for _, match := range re.FindAllString(content, -1) {
		vars := strings.Split(match, "/")
		mapSources[vars[len(vars)-1]] = match
	}
}

func findVariables(content string, mapVariables map[string]*Variables) {
	var re = regexp.MustCompile(`(?m)\$\([A-Z0-9_\-]*\)`)

	for _, match := range re.FindAllString(content, -1) {

		if _, ok := mapVariables[match]; !ok {

			v := &Variables{
				name:  match[2 : len(match)-1],
				value: "",
			}

			regexVar := regexp.MustCompile("(?m)" + v.name + "[ \t]*=.*$")
			for _, matchVar := range regexVar.FindAllString(content, -1) {
				v.value = matchVar
				break
			}

			mapVariables[match] = v
		}
	}
}

func resolveVariables(mapVariables map[string]*Variables) {
	for _, value := range mapVariables {

		var re = regexp.MustCompile(`(?m)\$\([A-Z0-9_\-]*\)`)

		resolved := false
		varString := ""
		for _, match := range re.FindAllString(value.value, -1) {
			vars := strings.Split(mapVariables[match].value, "=")
			if len(vars) > 1 {
				varString = vars[1]
			} else {
				varString = mapVariables[match].value
			}

			value.value = strings.Replace(value.value, match, varString, -1)
			resolved = true
		}
		if !resolved {
			vars := strings.Split(value.value, "=")
			if len(vars) > 1 {
				varString = vars[1]
			}
			value.value = varString
		}
	}
}

func detectURL(mapVariables map[string]*Variables) *string {
	for key, value := range mapVariables {
		if strings.Contains(key, URL) && strings.Contains(value.value, HTTP) {
			spaceDel := strings.Join(strings.Split(value.value, " "), "")
			vars := strings.Split(spaceDel, "=")
			if len(vars) > 1 {
				return &vars[1]
			}
			return &value.value
		}
	}
	return nil
}

// TODO REPLACE
func CreateFolder(path string) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err = os.Mkdir(path, 0755); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func findSourcesFiles(workspace string) ([]string, error) {

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

func saveSymbols(output string, mapSymbols map[string]string, libName string) {
	if strings.Contains(output, "\n") {
		output = strings.TrimSuffix(output, "\n")
	}

	symbols := strings.Split(output, ",")
	for _, s := range symbols {
		if len(s) > 0 {
			if _, ok := mapSymbols[s]; !ok {
				if s == "main" || strings.Contains(s, "test") || strings.Contains(s, "TEST") {
					u.PrintWarning("Ignore function: " + s)
				} else {
					mapSymbols[s] = libName
				}
			}
		}
	}
}

func extractPrototype(sourcesFiltered []string, mapSymbols map[string]string, libName string) error {

	for _, f := range sourcesFiltered {
		script := filepath.Join(os.Getenv("GOPATH"), "src", "tools", "srcs", "extractertool", "parserClang.py")
		output, err := ExecuteCommand("python3", []string{script, "-q", "-t", f})
		if err != nil {
			u.PrintWarning("Incomplete analysis with file " + f)
			continue
		}
		saveSymbols(output, mapSymbols, libName)
	}
	return nil
}

func filterSourcesFiles(files []string, mapSources map[string]string) []string {
	var sourcesFiltered []string
	for _, f := range files {
		vars := strings.Split(f, "/")
		if len(vars) > 1 {
			filename := vars[len(vars)-1]
			if _, ok := mapSources[filename]; ok {
				sourcesFiltered = append(sourcesFiltered, f)
			}
		}
	}
	return sourcesFiltered
}

func RunExtracterTool(homeDir string) {

	// Init and parse local arguments
	args := new(u.Arguments)
	p, err := args.InitArguments("--extracter",
		"The extracter tool allows to extract all the symbols (functions) of an external/internal library")
	if err != nil {
		u.PrintErr(err)
	}
	if err := parseLocalArguments(p, args); err != nil {
		u.PrintErr(err)
	}

	var workspacePath = homeDir + u.SEP + u.WORKSPACEFOLDER
	if len(*args.StringArg[workspaceArg]) > 0 {
		workspacePath = *args.StringArg[workspaceArg]
	}

	libpath := *args.StringArg[library]
	lib := libpath
	if filepath.IsAbs(libpath) {
		lib = filepath.Base(libpath)
	} else {
		libpath = filepath.Join(workspacePath, u.LIBSFOLDER, lib)
	}

	file, err := ioutil.ReadFile(filepath.Join(libpath, "Makefile.uk"))
	if err != nil {
		u.PrintErr(err)
	}
	mapVariables := make(map[string]*Variables)
	content := string(file)

	mapSources := make(map[string]string)
	getMakefileSources(content, mapSources)
	findVariables(content, mapVariables)
	resolveVariables(mapVariables)
	url := detectURL(mapVariables)
	folderName := lib + "_sources_folder"
	var archiveName string
	var sourcesFiltered []string

	if url != nil {
		var fileExtension string
		urlSplit := strings.Split(*url, "/")
		if urlSplit[len(urlSplit)-1] == "download" {
			fileExtension = filepath.Ext(urlSplit[len(urlSplit)-2])
		} else {
			fileExtension = filepath.Ext(*url)
		}

		created, err := CreateFolder(folderName)
		if err != nil {
			u.PrintErr(err)
		}

		var files []string

		if fileExtension == ".gz" {
			archiveName = lib + "_sources.tar" + fileExtension

		} else {
			archiveName = lib + "_sources" + fileExtension
		}

		if created {
			u.PrintInfo(*url + " is found. Download the lib sources...")
			err := DownloadFile(archiveName, *url)
			if err != nil {
				u.PrintErr(err)
			}
			u.PrintOk(*url + " successfully downloaded.")

			u.PrintInfo("Extracting " + archiveName + "...")
			if fileExtension == ".zip" {
				files, err = Unzip(archiveName, folderName)
				if err != nil {
					_ = os.Remove(archiveName)
					_ = os.RemoveAll(folderName)
					u.PrintErr(err.Error() + ". Corrupted archive. Please try again.")
				}

			} else if fileExtension == ".tar" || fileExtension == ".gz" || fileExtension == ".tgz" {
				files, err = unTarGz(archiveName, folderName)
				if err != nil {
					_ = os.Remove(archiveName)
					_ = os.RemoveAll(folderName)
					u.PrintErr(err.Error() + ". Corrupted archive. Please try again.")
				}

			} else {
				u.PrintErr(errors.New("unknown extension for archive"))
			}
		}

		u.PrintInfo("Inspecting folder " + folderName + " for sources...")
		folderFiles, err := findSourcesFiles(folderName)
		if err != nil {
			u.PrintErr(err)
		}

		sourcesFiltered = filterSourcesFiles(files, mapSources)
		sourcesFiltered = append(sourcesFiltered, folderFiles...)
	}

	libpathFiles, err := findSourcesFiles(libpath)
	if err != nil {
		u.PrintErr(err)
	}
	sourcesFiltered = append(sourcesFiltered, libpathFiles...)

	u.PrintInfo("Find " + strconv.Itoa(len(sourcesFiltered)) + " files to analyse")

	mapSymbols := make(map[string]string)
	u.PrintInfo("Extracting symbols from all sources of " + lib + ". This may take some times...")
	if err := extractPrototype(sourcesFiltered, mapSymbols, lib); err != nil {
		u.PrintErr(err)
	}

	mf := MicroLibFile{}
	mf.Functions = make([]MicroLibsFunction, len(mapSymbols))
	i := 0
	for k, _ := range mapSymbols {
		mf.Functions[i].Name = k
		i++
	}

	u.PrintOk(strconv.Itoa(len(mapSymbols)) + " symbols from " + lib + " have been extracted.")

	var filename string
	if url != nil {
		filename = filepath.Join(os.Getenv("GOPATH"), "src", "tools", "libs", "external", lib)

	} else {
		filename = filepath.Join(os.Getenv("GOPATH"), "src", "tools", "libs", "internal", lib)
	}

	if err := u.RecordDataJson(filename, mf); err != nil {
		u.PrintErr(err)
	} else {
		u.PrintOk("Symbols file have been written to " + filename + ".json")
	}

	if url != nil {
		u.PrintInfo("Remove folders " + archiveName + " and " + folderName)
		_ = os.Remove(archiveName)
		_ = os.RemoveAll(folderName)
	}
}
